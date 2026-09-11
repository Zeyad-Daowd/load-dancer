package balancer

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const failuresThreshold = 3

func checkEndpoint(client *http.Client, endpoint string) bool {
	resp, err := client.Get(endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func checkServerHealth(client *http.Client, s *BackendServer, failures int) int {
	// Perform health check (e.g., send a GET request to the backend server)
	endpoint := s.addr.String() + "/health"

	if !checkEndpoint(client, endpoint) {
		failures += 1
		if failures >= failuresThreshold { // mark as unhealthy after 3 consecutive failures
			s.SetHealth(false)
		}
	} else {
		s.SetHealth(true)
		failures = 0 // reset failure count on success
	}
	return failures
}
func HealthCheck(ctx context.Context, servers []*BackendServer, period time.Duration) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	wg := sync.WaitGroup{}
	for _, server := range servers {
		wg.Add(1)
		go func(ctx context.Context, s *BackendServer) {
			defer wg.Done()
			ticker := time.NewTicker(period)
			failures := 0
			//initial run
			failures = checkServerHealth(client, s, failures)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					log.Print("Stopping health check for server: ", s.addr.String())
					return
				case <-ticker.C:
					failures = checkServerHealth(client, s, failures)
				}
			}
		}(ctx, server)
	}
	<-ctx.Done()
	wg.Wait()
}
