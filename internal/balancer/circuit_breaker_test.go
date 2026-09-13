package balancer

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedAllowsRequests(t *testing.T) {
	cb := NewCircuitBreaker(3, 1)
	if !cb.AllowRequest() {
		t.Error("expected a new circuit breaker to allow requests")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(3, 1)
	cb.RecordFailure()
	cb.RecordFailure()
	if !cb.AllowRequest() {
		t.Error("expected requests to still be allowed below the failure threshold")
	}
	cb.RecordFailure() // 3rd failure hits the threshold
	if cb.AllowRequest() {
		t.Error("expected the circuit to be open after reaching the failure threshold")
	}
}

func TestCircuitBreaker_HalfOpenAllowsExactlyOneTrial(t *testing.T) {
	cb := NewCircuitBreaker(1, 1) // trips after 1 failure, 1s timeout
	cb.RecordFailure()
	if cb.AllowRequest() {
		t.Fatal("expected the circuit to be open immediately after tripping")
	}
	time.Sleep(1100 * time.Millisecond)
	if !cb.AllowRequest() {
		t.Error("expected exactly one trial request once the timeout has elapsed")
	}
	if cb.AllowRequest() {
		t.Error("expected a second concurrent request to be rejected while the trial is in flight")
	}
}

func TestCircuitBreaker_ClosesOnSuccessfulTrial(t *testing.T) {
	cb := NewCircuitBreaker(1, 1)
	cb.RecordFailure()
	time.Sleep(1100 * time.Millisecond)
	cb.AllowRequest() // consumes the trial
	cb.RecordSuccess()
	if !cb.AllowRequest() {
		t.Error("expected the circuit to be closed and allowing requests after a successful trial")
	}
}

func TestCircuitBreaker_RecoversAfterFailedHalfOpenTrial(t *testing.T) {
	cb := NewCircuitBreaker(1, 1)
	cb.RecordFailure() // trips open
	time.Sleep(1100 * time.Millisecond)
	cb.AllowRequest()  // consumes the trial
	cb.RecordFailure() // trial fails, circuit reopens

	time.Sleep(1100 * time.Millisecond) // wait out the new open window
	if !cb.AllowRequest() {
		t.Error("expected a new half-open trial to be allowed after the circuit reopens and its timeout elapses again")
	}
}

func TestCircuitBreakerSkipsPersistentlyFailingBackend(t *testing.T) {
	backend1 := createFailingBackendServer(http.StatusInternalServerError) // code >= 500 triggers failure
	backend2 := createMockBackendServer("OK2")
	defer backend1.Close()
	defer backend2.Close()

	server1, err := CreateBackendServer(backend1.URL)
	if err != nil {
		t.Fatal(err)
	}
	server2, err := CreateBackendServer(backend2.URL)
	if err != nil {
		t.Fatal(err)
	}

	rr := NewRoundRobin([]*BackendServer{server1, server2})
	handler := rr.Handler()

	// Round robin alternates 1,2,1,2,... - 6 requests guarantees backend1
	// is hit 3 times, tripping its circuit breaker (threshold is 3).
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)
	}

	// backend1's circuit should now be open; every subsequent request
	// should land on backend2, even though round robin would otherwise
	// alternate back to backend1.
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		if got := w.Body.String(); got != "OK2" {
			t.Errorf("expected OK2 (backend1's circuit should be open), got %q", got)
		}
	}
}
