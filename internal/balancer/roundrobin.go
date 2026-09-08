package balancer

import (
	"log"
	"net/http"
)

type RoundRobin struct {
	servers []*BackendServer
	current int
}

func (rr *RoundRobin) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Forwarding request to backend server:", rr.servers[rr.current].addr.String())
		rr.servers[rr.current].ServeHTTP(w, r)
		rr.current = (rr.current + 1) % len(rr.servers)
	}
}

func NewRoundRobin(servers []*BackendServer) *RoundRobin {
	return &RoundRobin{
		servers: servers,
		current: 0,
	}
}
