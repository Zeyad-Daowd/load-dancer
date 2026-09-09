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
		rr.mutex.Lock()
		currentServer := rr.current
		rr.current = (rr.current + 1) % len(rr.servers)
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
