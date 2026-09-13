package balancer

import (
	"net/url"
	"sync/atomic"
)

type BackendServer struct {
	addr           *url.URL
	health         atomic.Bool
	circuitBreaker *CircuitBreaker
}

func CreateBackendServer(urlStr string) (*BackendServer, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	cb := NewCircuitBreaker(3, 5)
	server := &BackendServer{addr: parsedURL, circuitBreaker: cb}
	server.health.Store(true)
	return server, nil
}

func (b *BackendServer) IsHealthy() bool {
	return b.health.Load()
}

func (b *BackendServer) SetHealth(healthy bool) {
	b.health.Store(healthy)
}

func getAliveServersCount(servers []*BackendServer) int {
	count := 0
	for _, server := range servers {
		if server.IsHealthy() {
			count++
		}
	}
	return count
}

func (b *BackendServer) IsAvailable() bool {
	return b.circuitBreaker.AllowRequest()
}
