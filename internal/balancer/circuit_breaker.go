package balancer

import (
	"sync"
	"time"
)

type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

type CircuitBreaker struct {
	failureThreshold int
	failureCount     int
	timeout          int64
	state            CircuitBreakerState
	mutex            sync.Mutex
	halfOpenSent     bool
	openingTime      time.Time
}

func NewCircuitBreaker(failureThreshold int, timeout int64) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		state:            Closed,
		timeout:          timeout,
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	if cb.state == Open && time.Since(cb.openingTime).Seconds() >= float64(cb.timeout) {
		cb.state = HalfOpen
	}
	switch cb.state {
	case Open:
		return false
	case HalfOpen:
		if !cb.halfOpenSent {
			cb.halfOpenSent = true
			return true
		}
		return false
	default:
		return true
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.state = Closed
	cb.failureCount = 0
	cb.halfOpenSent = false
	cb.openingTime = time.Time{}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount++
	if cb.failureCount >= cb.failureThreshold {
		cb.state = Open
		cb.halfOpenSent = false
		cb.openingTime = time.Now()
	}
}
