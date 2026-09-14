package balancer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
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

func (rr *RoundRobin) selectServer() *BackendServer {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	currentServer := rr.current
	maxAttempts := len(rr.servers)
	attempts := 0
	for (rr.servers[currentServer].IsHealthy() == false || rr.servers[currentServer].IsAvailable() == false) && attempts < maxAttempts {
		currentServer = (currentServer + 1) % len(rr.servers)
		attempts++
	}
	if attempts == maxAttempts {
		return nil
	}
	rr.current = (currentServer + 1) % len(rr.servers)
	return rr.servers[currentServer]
}
func (rr *RoundRobin) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rr.mutex.Lock()
		noServers := len(rr.servers) == 0
		rr.mutex.Unlock()
		if noServers {
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

type backendStatus struct {
	URL       string `json:"url"`
	Healthy   bool   `json:"healthy"`
	Available bool   `json:"available"`
}

func (rr *RoundRobin) GetServers() []backendStatus {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	backends := make([]backendStatus, len(rr.servers))
	for i, server := range rr.servers {
		backends[i] = backendStatus{
			URL:       server.addr.String(),
			Healthy:   server.IsHealthy(),
			Available: server.IsAvailable(),
		}
	}
	return backends
}

var ErrExistingBackend = errors.New("backend server already exists")

func (rr *RoundRobin) AddServer(urlStr string, healthCheckPeriod time.Duration) error {
	server, err := CreateBackendServer(urlStr)
	if err != nil {
		return err
	}
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	for _, existingServer := range rr.servers {
		if existingServer.addr.String() == server.addr.String() {
			slog.Warn("Attempted to add a backend server that already exists", "url", urlStr)
			return ErrExistingBackend
		}
	}
	rr.servers = append(rr.servers, server)
	go HealthCheck(context.Background(), []*BackendServer{server}, healthCheckPeriod)
	slog.Info("Added new backend server", "url", urlStr)
	return nil
}
