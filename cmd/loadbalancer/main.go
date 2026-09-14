package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	balancer "github.com/zeyad-daowd/load-dancer/internal/balancer"
	control "github.com/zeyad-daowd/load-dancer/internal/control"
	"github.com/zeyad-daowd/load-dancer/internal/middleware"
)

func main() {
	//TODO: make backends register for load balancing
	sigtermCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	servers := []*balancer.BackendServer{}
	// for _, urlStr := range []string{"http://localhost:8001", "http://localhost:8002", "http://localhost:8003"} {
	// 	server, err := balancer.CreateBackendServer(urlStr)
	// 	if err != nil {
	// 		slog.Error("Error creating backend server", "url", urlStr, "error", err)
	// 	} else {
	// 		servers = append(servers, server)
	// 	}
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	healthCheckPeriod := 1 * time.Second
	rr := balancer.NewRoundRobin(servers)
	controller := control.NewBackendController(rr, ":8081", healthCheckPeriod)
	go func() {
		err := controller.Srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Error serving control server", "error", err)
			stop()
		}
	}()

	mux := http.NewServeMux()
	go balancer.HealthCheck(ctx, servers, healthCheckPeriod) // check health every 1 second
	mux.Handle("GET /", middleware.LoggingMiddleware(rr.Handler()))

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,   // time to read just the request headers
		ReadTimeout:       10 * time.Second,  // time to read the full request
		WriteTimeout:      10 * time.Second,  // time to write the response
		IdleTimeout:       120 * time.Second, // how long a keep-alive connection may sit idle
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Error serving server", "error", err)
			stop()
		}
	}()

	<-sigtermCtx.Done()
	slog.Info("Shutting down server gracefully...")
	// for requests
	cancel()
	wg := sync.WaitGroup{}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	// gracefully shutdown the server giving it 10 seconds to finish ongoing requests
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("Error shutting down server", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := controller.Srv.Shutdown(shutdownCtx)
		if err != nil {
			slog.Error("Error shutting down control server", "error", err)
		}
	}()
	wg.Wait()
	slog.Info("Server shutdown complete")
}
