package balancer

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"sync"
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
	proxy          *httputil.ReverseProxy
}

func (rr *RoundRobin) selectServer() int {
	rr.mutex.Lock()
	currentServer := rr.current
	maxAttempts := len(rr.servers)
	attempts := 0
	for (rr.servers[currentServer].IsHealthy() == false || rr.servers[currentServer].IsAvailable() == false) && attempts < maxAttempts {
		currentServer = (currentServer + 1) % len(rr.servers)
		attempts++
	}
	if attempts == maxAttempts {
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
		rr.ServeHTTP(w, r)
	}
}

func NewRoundRobin(servers []*BackendServer) *RoundRobin {
	// reverse proxy to forward requests
	proxy := httputil.ReverseProxy{
		Director: func(req *http.Request) {},
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("Error proxying request to backend server", "url", r.URL.String(), "error", err, "errorType", fmt.Sprintf("%T", err))
		if errors.Is(err, ErrNoHealthyBackends) {
			http.Error(w, "No healthy backend servers available", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "backend unavailable", http.StatusBadGateway)
		}
	}
	rr := &RoundRobin{
		servers: servers,
		current: 0,
		mutex:   sync.Mutex{},
		retryTransport: &RetryBalancerTransport{
			MaxAttempts:   3,
			Balancer:      nil,
			BaseTransport: &baseTransport,
		},
		proxy: &proxy,
	}
	rr.retryTransport.Balancer = rr
	rr.proxy.Transport = rr.retryTransport
	return rr
}

func (rr *RoundRobin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rr.proxy.ServeHTTP(w, r)
}
