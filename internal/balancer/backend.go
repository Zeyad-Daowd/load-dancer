package balancer

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type BackendServer struct {
	addr  *url.URL
	proxy *httputil.ReverseProxy
}

func CreateBackendServer(urlStr string) (*BackendServer, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// reverse proxy to forward requests
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	return &BackendServer{addr: parsedURL, proxy: proxy}, nil
}

func (b *BackendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.proxy.ServeHTTP(w, r)
}
