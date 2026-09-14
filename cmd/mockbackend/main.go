package main

import (
	"bytes"
	"context"
	"encoding/json"
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
)

type Server struct {
	addr  *url.URL
	delay bool
}

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.delay {
			time.Sleep(30 * time.Second)
		}
		slog.Info("Got request to backend server", "url", s.addr.String())
		w.Write([]byte("Hello from backend server: " + s.addr.String() + "\n"))
	}
}
func (s *Server) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Got health check to backend server", "url", s.addr.String())
		w.WriteHeader(http.StatusOK)
	}
}

type RegisterBackendRequest struct {
	URL      string    `json:"url"`
	UniqueID uuid.UUID `json:"uniqueID"`
}

func attemptRegistration(fullURL string, requestBody []byte) (success bool, err error) {
	resp, err := http.Post(fullURL, "application/json", bytes.NewReader(requestBody))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close() // closes as soon as this function returns

	return resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict, nil
}

func SendServerRegistrationRequest(serverURL string, uniqueID uuid.UUID, controllerURL string) error {

	reqPayload := RegisterBackendRequest{URL: serverURL, UniqueID: uniqueID}
	requestBody, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}
	fullURL := controllerURL + "/backends/register"
	maxRetries := 5
	retries := 0
	// Send the POST request to the controller
	for retries < maxRetries {
		success, err := attemptRegistration(fullURL, requestBody)
		if err != nil {
			slog.Error("Error sending registration request", "error", err)
		} else if success {
			slog.Info("Successfully registered backend server", "serverURL", serverURL, "controllerURL", controllerURL)
			return nil
		} else {
			slog.Warn("Failed to register backend server, will retry", "serverURL", serverURL, "controllerURL", controllerURL)
		}
		retries++
		time.Sleep(2 * time.Second) // wait before retrying
	}
	slog.Error("Failed to register backend server breaking out after max attempts", "serverURL", serverURL, "controllerURL", controllerURL)
	return fmt.Errorf("failed to register backend server after %d attempts", maxRetries)
}
func attemptDelete(client *http.Client, fullURL string) (success bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := client.Do(req)
	if err != nil && resp != nil && resp.StatusCode == http.StatusNotFound {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

func SendServerDeleteRequest(uniqueID uuid.UUID, controllerURL string) error {
	fullURL, err := url.JoinPath(controllerURL, "backends", uniqueID.String())
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	maxRetries := 5
	retryDelay := 2 * time.Second
	client := http.DefaultClient

	for retries := 0; retries < maxRetries; retries++ {
		success, err := attemptDelete(client, fullURL)

		if err != nil {
			slog.Error("Error sending delete request", "error", err, "attempt", retries+1)
		} else if success {
			slog.Info("Successfully deleted backend server", "id", uniqueID, "controllerURL", controllerURL)
			return nil
		} else {
			slog.Warn("Failed to delete backend server, will retry", "id", uniqueID, "controllerURL", controllerURL)
		}
		time.Sleep(retryDelay)
	}

	slog.Error("Failed to delete backend server after max attempts", "id", uniqueID, "controllerURL", controllerURL)
	return fmt.Errorf("failed to delete backend server after %d attempts", maxRetries)
}
func main() {
	// check if -delay flag is provided
	delayFlag := flag.Bool("delay", false, "simulate a slow backend with a 3s delay")
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
	server := &Server{addr: parsedURL, delay: *delayFlag}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", server.HealthHandler())
	mux.HandleFunc("GET /", server.Handler())
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
	loadBalancerURL := "http://localhost:8081"
	go SendServerRegistrationRequest(parsedURL.String(), uniqueID, loadBalancerURL)
	sigtermCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	// gracefully shutdown the server giving it 10 seconds to finish ongoing requests
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		slog.Error("Error shutting down server", "error", err)
	}

	err = SendServerDeleteRequest(uniqueID, loadBalancerURL)
	if err != nil {
		slog.Error("Error sending server delete request", "error", err)
	}
	slog.Info("Server Shutdown complete. Server is now offline.")

}
