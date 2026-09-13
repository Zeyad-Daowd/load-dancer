package balancer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func createMockBackendServerWithDelay(responseText string, delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.Write([]byte(responseText))
	}))
}

func createFailingBackendServer(status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}))
}
func TestRetry(t *testing.T) {
	backend1 := createMockBackendServer("OK1")
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
	// Make the first backend unavailable.
	backend1.Close()

	rr := NewRoundRobin([]*BackendServer{server1, server2})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	expected := "OK2"
	handler(w, req)

	if got := w.Body.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestMultipleRetries(t *testing.T) {
	backend1 := createMockBackendServer("OK1")
	backend2 := createMockBackendServer("OK2")
	backend3 := createMockBackendServer("OK3")
	defer backend1.Close()
	defer backend2.Close()
	defer backend3.Close()

	server1, err := CreateBackendServer(backend1.URL)
	if err != nil {
		t.Fatal(err)
	}

	server2, err := CreateBackendServer(backend2.URL)
	if err != nil {
		t.Fatal(err)
	}

	server3, err := CreateBackendServer(backend3.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Make the first 2 backends unavailable.
	backend1.Close()
	backend2.Close()

	rr := NewRoundRobin([]*BackendServer{server1, server2, server3})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	expected := "OK3"
	handler(w, req)

	if got := w.Body.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRetryStopsAfterMaxAttempts(t *testing.T) {
	backend1 := createMockBackendServer("OK1")
	backend2 := createMockBackendServer("OK2")
	backend3 := createMockBackendServer("OK3")
	backend4 := createMockBackendServer("OK4")
	defer backend1.Close()
	defer backend2.Close()
	defer backend3.Close()
	defer backend4.Close()

	server1, err := CreateBackendServer(backend1.URL)
	if err != nil {
		t.Fatal(err)
	}

	server2, err := CreateBackendServer(backend2.URL)
	if err != nil {
		t.Fatal(err)
	}

	server3, err := CreateBackendServer(backend3.URL)
	if err != nil {
		t.Fatal(err)
	}

	server4, err := CreateBackendServer(backend4.URL)
	if err != nil {
		t.Fatal(err)
	}
	// Make the first three backend unavailable.
	backend1.Close()
	backend2.Close()
	backend3.Close()

	rr := NewRoundRobin([]*BackendServer{server1, server2, server3, server4})

	// expected it to fail since 3 failed servers
	errorMessage := "backend unavailable"
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	handler(w, req)
	got := w.Body.String()
	if !strings.Contains(got, errorMessage) {
		t.Errorf("expected %q to be in message, got %q", errorMessage, got)
	}
	w = httptest.NewRecorder()
	expected := "OK4"
	handler(w, req)

	if got := w.Body.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestRetryTimeout(t *testing.T) {
	//slow backend1
	backend1 := createMockBackendServerWithDelay("OK1", 6*time.Second)
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

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	expected := "OK2"
	handler(w, req)

	if got := w.Body.String(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestNoRetryPost(t *testing.T) {
	backend1 := createMockBackendServer("OK1")
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
	backend1.Close() // Make the first backend unavailable.
	rr := NewRoundRobin([]*BackendServer{server1, server2})

	errorMessage := "backend unavailable"
	req := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	handler(w, req)
	got := w.Body.String()
	if !strings.Contains(got, errorMessage) {
		t.Errorf("expected %q to be in message, got %q", errorMessage, got)
	}
}

func TestNoRetryInternalServerErrors(t *testing.T) {
	backend1 := createFailingBackendServer(http.StatusInternalServerError)
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

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler := rr.Handler()
	handler(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

}
