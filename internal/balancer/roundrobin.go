package balancer

import (
	"log"
	"net/http"
	"sync"
)

type RoundRobin struct {
	servers []*BackendServer
	mutex   sync.Mutex
	current int
}

func (rr *RoundRobin) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(rr.servers) == 0 {
			http.Error(w, "No backend servers available", http.StatusServiceUnavailable)
			return
		}
		rr.mutex.Lock()
		currentServer := rr.current
		maxAttempts := len(rr.servers)
		attempts := 0
		for rr.servers[currentServer].IsHealthy() == false && attempts < maxAttempts {
			currentServer = (currentServer + 1) % len(rr.servers)
			attempts++
		}
		if attempts == maxAttempts {
			rr.mutex.Unlock()
			http.Error(w, "No backend servers available", http.StatusServiceUnavailable)
			return
		}
		rr.current = (currentServer + 1) % len(rr.servers)
		rr.mutex.Unlock()
		log.Println("Forwarding request to backend server:", rr.servers[currentServer].addr.String())
		rr.servers[currentServer].ServeHTTP(w, r)
	}
}

func NewRoundRobin(servers []*BackendServer) *RoundRobin {
	return &RoundRobin{
		servers: servers,
		current: 0,
		mutex:   sync.Mutex{},
	}
}
