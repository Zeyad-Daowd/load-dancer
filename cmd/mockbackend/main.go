package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"
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
	URL string `json:"url"`
}

func SendServerRegistrationRequest(serverURL string, controllerURL string) error {

	reqPayload := RegisterBackendRequest{URL: serverURL}
	requestBody, err := json.Marshal(reqPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}
	fullURL := controllerURL + "/backends/register"
	maxRetries := 5
	retries := 0
	// Send the POST request to the controller
	resp, err := http.Post(fullURL, "application/json", bytes.NewReader(requestBody))
	for err != nil || resp == nil || (resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict) {
		slog.Error("Error sending registration request, retrying...", "error", err)
		time.Sleep(2 * time.Second) // wait for 2 seconds before retrying
		resp, err = http.Post(fullURL, "application/json", bytes.NewReader(requestBody))
		retries++
		if retries >= maxRetries {
			slog.Error("Failed to register backend server after maximum retries", "serverURL", serverURL, "controllerURL", controllerURL)
			return fmt.Errorf("failed to register backend server after %d retries", maxRetries)
		}
	}
	slog.Info("Successfully registered backend server", "serverURL", serverURL, "controllerURL", controllerURL)
	return nil
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
	go SendServerRegistrationRequest(parsedURL.String(), "http://localhost:8081")
	// TODO: add graceful shutdown
	slog.Error("Error serving backend server", "error", srv.ListenAndServe())
	os.Exit(1)
}
