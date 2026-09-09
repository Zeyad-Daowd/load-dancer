package balancer

import (
	"net/http"
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
func HealthCheck(servers []*BackendServer, period time.Duration) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	for _, server := range servers {
		go func(s *BackendServer) {
			ticker := time.NewTicker(period)
			failures := 0
			//initial run
			failures = checkServerHealth(client, s, failures)
			defer ticker.Stop()
			for range ticker.C {
				failures = checkServerHealth(client, s, failures)
			}
		}(server)
	}
	//TODO: add a mechanism (context) to stop the health check goroutines gracefully when the load balancer shuts down
	select {}
}
