package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zeyad-daowd/load-dancer/internal/orderservice/config"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
	handler "github.com/zeyad-daowd/load-dancer/internal/orderservice/handler"
	registration "github.com/zeyad-daowd/load-dancer/internal/orderservice/registration"
	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

const LoadBalancerEnabled = false

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		slog.Error("Please provide a backend URL as a command line argument.")
		os.Exit(1)
	}
	urlString := flag.Arg(0)
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		slog.Error("Error parsing backend URL", "error", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	controlPlaneSecret := cfg.ControlPlaneSecret
	pool, err := connectToDB(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	queries := db.New(pool)
	fmt.Println("Connected to database successfully:", cfg.DatabaseURL, queries)
	// nil causes slog to log from INFO level and above
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	orderService := service.NewOrderService(queries, pool)
	h := handler.NewOrderServiceHandler(logger, orderService, parsedURL.String())
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("POST /users", h.CreateUser) //admin auth later
	mux.HandleFunc("GET /products", h.GetProducts)
	mux.HandleFunc("POST /products", h.AddProduct)                      //admin auth later
	mux.HandleFunc("GET /products/{id}/stock", h.GetStock)              //admin auth later
	mux.HandleFunc("POST /products/{id}/stock", h.IncreaseProductStock) //admin auth later
	mux.HandleFunc("POST /orders/{orderID}/items", h.AddOrderItem)
	mux.HandleFunc("PUT /orders/{orderID}/items/{productID}", h.EditOrderItem)
	mux.HandleFunc("POST /orders", h.CreateOrder)
	mux.HandleFunc("POST /orders/{orderID}/checkout", h.CheckoutOrder)
	mux.HandleFunc("GET /orders/{orderID}", h.GetOrder)
	mux.HandleFunc("GET /users/{userId}/orders", h.GetUserOrders)
	mux.HandleFunc("POST /categories", h.AddCategory)                             //admin auth later
	mux.HandleFunc("POST /products/{productID}/categories", h.AddProductCategory) //admin auth later
	srv := &http.Server{
		Addr:              parsedURL.Host,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,   // time to read just the request headers
		ReadTimeout:       10 * time.Second,  // time to read the full request
		WriteTimeout:      10 * time.Second,  // time to write the response
		IdleTimeout:       120 * time.Second, // how long a keep-alive connection may sit idle
	}

	slog.Info("Starting backend server", "url", parsedURL.String())
	uniqueID := uuid.New()
	loadBalancerURL := cfg.LoadBalancerURL
	sigtermCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Error serving server", "error", err)
			stop()
		}
	}()
	if LoadBalancerEnabled {
		err = registration.RegisterBackend(parsedURL.String(), uniqueID, loadBalancerURL, controlPlaneSecret)
		if err != nil {
			slog.Error("Error sending server registration request", "error", err)
			stop()
		}
	}

	<-sigtermCtx.Done()
	slog.Info("Shutting down server gracefully...")
	if LoadBalancerEnabled {
		err = registration.DeregisterBackend(uniqueID, loadBalancerURL, controlPlaneSecret)
		if err != nil {
			slog.Error("Error sending server delete request", "error", err)
		}
	}
	// for requests
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	// gracefully shutdown the server giving it 10 seconds to finish ongoing requests
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		slog.Error("Error shutting down server", "error", err)
	}

	slog.Info("Server Shutdown complete. Server is now offline.")

}

func connectToDB(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
