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

// TODO: make an interface for the balancer to allow for different balancing strategies
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

func NewRoundRobin() *RoundRobin {
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
		servers: []*BackendServer{},
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
	URL       string    `json:"url"`
	Healthy   bool      `json:"healthy"`
	Available bool      `json:"available"`
	Id        uuid.UUID `json:"uniqueID"`
}

func (rr *RoundRobin) GetServers() []backendStatus {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	backends := make([]backendStatus, len(rr.servers))
	index := 0
	for id, server := range rr.serversMap {
		backends[index] = backendStatus{
			URL:       server.addr.String(),
			Healthy:   server.IsHealthy(),
			Available: server.IsAvailable(),
			Id:        id,
		}
		index++
	}
	return backends
}

var ErrExistingBackend = errors.New("backend server already exists")

func (rr *RoundRobin) AddServer(server *BackendServer, uniqueID uuid.UUID) error {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	_, exists := rr.serversMap[uniqueID]
	if exists {
		slog.Warn("Attempted to add a backend server that already exists", "url", server.addr.String(), "uniqueID", uniqueID.String())
		return ErrExistingBackend
	}
	rr.serversMap[uniqueID] = server
	rr.servers = append(rr.servers, server)
	slog.Info("Added new backend server", "url", server.addr.String())
	return nil
}

func (rr *RoundRobin) StartHealthChecks(ctx context.Context, healthCheckPeriod time.Duration, uniqueID uuid.UUID, server *BackendServer) error {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()
	_, exists := rr.serversMap[uniqueID]
	if !exists {
		slog.Warn("Attempted to start health checks for a backend server that does not exist", "url", server.addr.String(), "uniqueID", uniqueID.String())
		return ErrBackendNotFound
	}
	ctxChild, cancel := context.WithCancel(ctx)
	rr.serversCancelMap[uniqueID] = cancel
	go HealthCheck(ctxChild, []*BackendServer{server}, healthCheckPeriod)
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
