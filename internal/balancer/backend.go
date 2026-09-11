package balancer

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"
)

type BackendServer struct {
	addr   *url.URL
	proxy  *httputil.ReverseProxy
	health atomic.Bool
}

var transport = http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 10 * time.Second,
	}).DialContext,
	ResponseHeaderTimeout: 5 * time.Second,
	IdleConnTimeout:       10 * time.Second,
	MaxIdleConns:          100,
	MaxIdleConnsPerHost:   20,
}

func CreateBackendServer(urlStr string) (*BackendServer, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// reverse proxy to forward requests
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	proxy.Transport = &transport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Error proxying request to backend server %s: %v", parsedURL.String(), err)
		http.Error(w, "backend unavailable", http.StatusBadGateway)
	}
	server := &BackendServer{addr: parsedURL, proxy: proxy}
	server.health.Store(true)
	return server, nil
}

func (b *BackendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.proxy.ServeHTTP(w, r)
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
