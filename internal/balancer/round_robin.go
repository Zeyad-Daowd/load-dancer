package balancer

import (
	"log/slog"
	"net/http"
	"sync"

	middleware "github.com/zeyad-daowd/load-dancer/internal/middleware"
)

type contextKey string

const (
	BackendServerKey contextKey = "backendServerKey"
)

type RoundRobin struct {
	servers        []*BackendServer
	mutex          sync.Mutex
	current        int
	retryTransport *RetryBalancerTransport
}

func (rr *RoundRobin) selectServer() int {
	rr.mutex.Lock()
	currentServer := rr.current
	maxAttempts := len(rr.servers)
	attempts := 0
	for rr.servers[currentServer].IsHealthy() == false && attempts < maxAttempts {
		currentServer = (currentServer + 1) % len(rr.servers)
		attempts++
	}
	if rr.servers[currentServer].IsHealthy() == false {
		rr.mutex.Unlock()
		return -1
	}
	rr.current = (currentServer + 1) % len(rr.servers)
	rr.mutex.Unlock()
	return currentServer
}
func (rr *RoundRobin) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(rr.servers) == 0 {
			http.Error(w, "No backend servers available", http.StatusServiceUnavailable)
			return
		}
		currentServer := rr.selectServer()
		if currentServer == -1 {
			http.Error(w, "No backend servers available", http.StatusServiceUnavailable)
			return
		}
		slog.Info("Forwarding request to backend server", "url", rr.servers[currentServer].addr.String(), "requestId", middleware.GetRequestId(r.Context()))
		rr.servers[currentServer].ServeHTTP(w, r)
	}
}

func NewRoundRobin(servers []*BackendServer) *RoundRobin {
	rr := &RoundRobin{
		servers: servers,
		current: 0,
		mutex:   sync.Mutex{},
		retryTransport: &RetryBalancerTransport{
			MaxAttempts:   3,
			Balancer:      nil,
			BaseTransport: &baseTransport,
		},
	}
	rr.retryTransport.Balancer = rr

	for _, server := range servers {
		server.proxy.Transport = rr.retryTransport
	}
	return rr
}
