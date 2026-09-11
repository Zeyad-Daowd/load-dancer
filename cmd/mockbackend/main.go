package main

import (
	"flag"
	"log"
	"net/http"
	"net/url"
	"time"
)

type Server struct {
	addr  *url.URL
	delay bool
}

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.delay {
			time.Sleep(3 * time.Second)
		}
		log.Println("Got request to backend server:", s.addr.String())
		w.Write([]byte("Hello from backend server: " + s.addr.String() + "\n"))
	}
}
func (s *Server) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got health check to backend server:", s.addr.String())
		w.WriteHeader(http.StatusOK)
	}
}
func main() {
	// check if -delay flag is provided
	delayFlag := flag.Bool("delay", false, "simulate a slow backend with a 3s delay")
	flag.Parse()
	if flag.NArg() < 1 {
		log.Fatal("Please provide at backend URL as a command line argument.")
	}
	urlString := flag.Arg(0)
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		log.Fatalf("Error parsing backend URL: %v", err)
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

	log.Fatal(srv.ListenAndServe())
}
