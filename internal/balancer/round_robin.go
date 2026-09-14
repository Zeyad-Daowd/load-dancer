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

	"github.com/google/uuid"
)

type contextKey string

const (
	BackendServerKey contextKey = "backendServerKey"
)

type RoundRobin struct {
	servers          []*BackendServer
	mutex            sync.Mutex
	current          int
	retryTransport   *RetryBalancerTransport
	proxy            *httputil.ReverseProxy
	serversMap       map[uuid.UUID]*BackendServer
	serversCancelMap map[uuid.UUID]context.CancelFunc
}

func (rr *RoundRobin) selectServer() *BackendServer {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	if len(rr.servers) == 0 {
		return nil
	}
	currentServer := rr.current % len(rr.servers)
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
		proxy:            &proxy,
		serversMap:       make(map[uuid.UUID]*BackendServer),
		serversCancelMap: make(map[uuid.UUID]context.CancelFunc),
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

func (rr *RoundRobin) AddServer(ctx context.Context, urlStr string, uniqueID uuid.UUID, healthCheckPeriod time.Duration) error {
	server, err := CreateBackendServer(urlStr)
	if err != nil {
		return err
	}
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	_, exists := rr.serversMap[uniqueID]
	if exists {
		slog.Warn("Attempted to add a backend server that already exists", "url", urlStr, "uniqueID", uniqueID.String())
		return ErrExistingBackend
	}
	rr.serversMap[uniqueID] = server
	rr.servers = append(rr.servers, server)
	ctxChild, cancel := context.WithCancel(ctx)
	rr.serversCancelMap[uniqueID] = cancel
	go HealthCheck(ctxChild, []*BackendServer{server}, healthCheckPeriod)
	slog.Info("Added new backend server", "url", urlStr)
	return nil
}

var ErrBackendNotFound = errors.New("backend server not found for deletion")

func (rr *RoundRobin) RemoveServer(uniqueID uuid.UUID) error {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	server, exists := rr.serversMap[uniqueID]
	if !exists {
		slog.Warn("Attempted to remove a backend server that does not exist", "uniqueID", uniqueID.String())
		return ErrBackendNotFound
	}
	delete(rr.serversMap, uniqueID)
	cancelFunc, cancelExists := rr.serversCancelMap[uniqueID]
	if cancelExists {
		cancelFunc()
		delete(rr.serversCancelMap, uniqueID)
	}
	for i, s := range rr.servers {
		if s == server {
			rr.servers = append(rr.servers[:i], rr.servers[i+1:]...)
			break
		}
	}
	slog.Info("Removed backend server", "uniqueID", uniqueID.String())
	return nil
}
