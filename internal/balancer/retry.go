package balancer

import (
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"time"
)

type RetryBalancerTransport struct {
	BaseTransport http.RoundTripper
	Balancer      *RoundRobin
	MaxAttempts   int
}

func (t *RetryBalancerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for attempt := 0; attempt < t.MaxAttempts; attempt++ {
		if attempt > 0 {
			nextIdx := t.Balancer.selectServer()
			if nextIdx == -1 {
				return nil, fmt.Errorf("retry failed: no healthy backend servers available")
			}

			nextServer := t.Balancer.servers[nextIdx]

			// route to next server
			req.URL.Scheme = nextServer.addr.Scheme // http or https
			req.URL.Host = nextServer.addr.Host     // update the host in the request URL
			req.Host = nextServer.addr.Host         // update the Host header in the request

			slog.Info("Retrying request on alternative backend",
				"url", nextServer.addr.String(),
				"attempt", attempt,
			)
		}

		resp, err = t.BaseTransport.RoundTrip(req)

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
