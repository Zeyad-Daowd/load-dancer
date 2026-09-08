package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Server struct {
	addr *url.URL
}

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request to backend server:", s.addr.String())
		w.Write([]byte("Hello from backend server: " + s.addr.String() + "\n"))
	}
}
func main() {
	// parse backend urls from command line arguments
	if len(os.Args) < 2 {
		log.Fatal("Please provide at backend URL as a command line argument.")
	}
	urlString := os.Args[1]
	parsedURL, err := url.Parse(urlString)
	if err != nil {
		log.Fatalf("Error parsing backend URL: %v", err)
	}
	server := &Server{addr: parsedURL}
	mux := http.NewServeMux()
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
