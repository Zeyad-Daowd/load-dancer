package balancer

import (
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/zeyad-daowd/load-dancer/internal/middleware"
)

var ErrNoHealthyBackends = errors.New("no healthy backend servers available")

var baseTransport = http.Transport{
	DialContext: (&net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 10 * time.Second,
	}).DialContext,
	ResponseHeaderTimeout: 5 * time.Second,
	IdleConnTimeout:       10 * time.Second,
	MaxIdleConns:          500,
	MaxIdleConnsPerHost:   500,
}

type RetryBalancerTransport struct {
	BaseTransport http.RoundTripper
	Balancer      *RoundRobin
	MaxAttempts   int
}

func (t *RetryBalancerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < t.MaxAttempts; attempt++ {

		nextServer := t.Balancer.selectServer()
		if nextServer == nil {
			// No healthy backend servers available
			slog.Error("No healthy backend servers available", "requestId", middleware.GetRequestId(req.Context()))
			return nil, ErrNoHealthyBackends
		}

		// route to next server
		req.URL.Scheme = nextServer.addr.Scheme // http or https
		req.URL.Host = nextServer.addr.Host     // update the host in the request URL
		req.Host = nextServer.addr.Host         // update the Host header in the request

		if attempt > 0 {
			slog.Info("Retrying request on alternative backend",
				"url", nextServer.addr.String(),
				"attempt", attempt,
			)
		} else {
			slog.Info("Forwarding request to backend server", "url", nextServer.addr.String(), "requestId", middleware.GetRequestId(req.Context()))
		}

		resp, err = t.BaseTransport.RoundTrip(req)
		// success means the request was successfully sent and a response was received with status code < 500
		if err == nil && (resp != nil && resp.StatusCode < 500) {
			nextServer.circuitBreaker.RecordSuccess()
		} else {
			nextServer.circuitBreaker.RecordFailure()
		}

		if !t.shouldRetry(err, req) {
			return resp, err
		}

		if attempt < t.MaxAttempts-1 {
			time.Sleep(backoffWithJitter(attempt))
		}
	}

	return resp, err
}
func (t *RetryBalancerTransport) shouldRetry(err error, req *http.Request) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodOptions && req.Method != http.MethodTrace {
		return false
	}
	if err != nil {
		return isRetryableError(err)
	}
	return false
}

func backoffWithJitter(attempt int) time.Duration {
	base := 100 * time.Millisecond
	maxBackoff := base * (1 << attempt)
	if maxBackoff > 2*time.Second {
		maxBackoff = 2 * time.Second
	}
	duration := time.Duration(rand.Int63n(int64(maxBackoff)))
	return duration
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			slog.Debug("timeout error")
			return true
		}
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		slog.Debug("network error")
		return true
	}
	return false
}
