package main

import (
	"log"
	"net/http"
	"time"

	balancer "github.com/zeyad-daowd/load-dancer/internal/balancer"
)

func main() {
	//TODO: make backends register for load balancing
	servers := []*balancer.BackendServer{}
	for _, urlStr := range []string{"http://localhost:8001", "http://localhost:8002", "http://localhost:8003"} {
		server, err := balancer.CreateBackendServer(urlStr)
		if err != nil {
			log.Fatalf("Error creating backend server: %v", err)
		}
		servers = append(servers, server)
	}
	mux := http.NewServeMux()
	rr := balancer.NewRoundRobin(servers)
	mux.HandleFunc("GET /", rr.Handler())

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,   // time to read just the request headers
		ReadTimeout:       10 * time.Second,  // time to read the full request
		WriteTimeout:      10 * time.Second,  // time to write the response
		IdleTimeout:       120 * time.Second, // how long a keep-alive connection may sit idle
	}

	log.Fatal(srv.ListenAndServe())
}
