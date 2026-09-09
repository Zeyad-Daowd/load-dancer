package balancer

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func createMockBackendServer(responseText string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(responseText))
	}))
}

func TestRoundRobinHandler(t *testing.T) {
	// Create mock backend servers
	responses := []string{"Response from backend 1", "Response from backend 2", "Response from backend 3"}

	servers := []*BackendServer{}
	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()
		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}
		servers = append(servers, server)
	}
	rr := NewRoundRobin(servers)
	handler := rr.Handler()
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		body := w.Body.String()
		expectedResponse := responses[i%len(responses)]
		if body != expectedResponse {
			t.Errorf("Expected response: %s, got: %s", expectedResponse, body)
		}
	}

}
func TestRoundRobinHandlerConcurrent(t *testing.T) {
	// Create mock backend servers
	responses := []string{"Response from backend 1", "Response from backend 2", "Response from backend 3"}

	servers := []*BackendServer{}
	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()
		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}
		servers = append(servers, server)
	}
	rr := NewRoundRobin(servers)
	handler := rr.Handler()
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	responsesCounter := make(map[string]int)
	for i := 0; i < 300; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()
			handler(w, req)
			body := w.Body.String()
			mu.Lock()
			responsesCounter[body]++
			defer mu.Unlock()
		}()
	}
	wg.Wait()
	for _, response := range responses {
		count, ok := responsesCounter[response]
		if !ok || count != 100 {
			t.Errorf("Expected response: %s, was received %d times instead of 100", response, count)
		}
	}

}
func TestRoundRobinHandlerOneServer(t *testing.T) {
	// Create mock backend servers
	responses := []string{"Response from backend 1"}

	servers := []*BackendServer{}
	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()
		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}
		servers = append(servers, server)
	}
	rr := NewRoundRobin(servers)
	handler := rr.Handler()
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		handler(w, req)
		body := w.Body.String()
		expectedResponse := responses[i%len(responses)]
		if body != expectedResponse {
			t.Errorf("Expected response: %s, got: %s", expectedResponse, body)
		}
	}
}
func TestRoundRobinHandlerNoServers(t *testing.T) {
	servers := []*BackendServer{}
	rr := NewRoundRobin(servers)
	handler := rr.Handler()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	err := w.Result().StatusCode
	if err != http.StatusServiceUnavailable {
		t.Errorf("Expected status code: %d, got: %d", http.StatusServiceUnavailable, err)
	}

}

func TestRoundRobinHandlerSkipsUnhealthyServer(t *testing.T) {
	responses := []string{
		"Response from backend 1",
		"Response from backend 2",
		"Response from backend 3",
	}

	servers := []*BackendServer{}

	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()

		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}

		servers = append(servers, server)
	}

	// Backend 2 is unhealthy.
	servers[1].SetHealth(false)

	rr := NewRoundRobin(servers)
	handler := rr.Handler()

	expectedResponses := []string{
		responses[0],
		responses[2],
		responses[0],
		responses[2],
		responses[0],
		responses[2],
	}

	for i, expected := range expectedResponses {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := w.Body.String(); got != expected {
			t.Errorf("request %d: expected %q, got %q", i, expected, got)
		}
	}
}

func TestRoundRobinHandlerAllServersUnhealthy(t *testing.T) {
	responses := []string{
		"Response from backend 1",
		"Response from backend 2",
		"Response from backend 3",
	}

	servers := []*BackendServer{}

	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()

		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}

		servers = append(servers, server)
	}

	// Mark every backend as unhealthy.
	for _, server := range servers {
		server.SetHealth(false)
	}

	rr := NewRoundRobin(servers)
	handler := rr.Handler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestRoundRobinHandlerRecoveredServer(t *testing.T) {
	responses := []string{
		"Response from backend 1",
		"Response from backend 2",
		"Response from backend 3",
	}

	servers := []*BackendServer{}

	for _, response := range responses {
		backend := createMockBackendServer(response)
		defer backend.Close()

		server, err := CreateBackendServer(backend.URL)
		if err != nil {
			t.Fatal(err)
		}

		servers = append(servers, server)
	}

	// Backend 2 starts unhealthy.
	servers[1].SetHealth(false)

	rr := NewRoundRobin(servers)
	handler := rr.Handler()

	// Backend 2 should be skipped.
	expectedResponses := []string{
		responses[0],
		responses[2],
		responses[0],
		responses[2],
	}

	for i, expected := range expectedResponses {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := w.Body.String(); got != expected {
			t.Errorf("before recovery, request %d: expected %q, got %q", i, expected, got)
		}
	}

	// Backend 2 recovers.
	servers[1].SetHealth(true)

	// Now all three should participate again.
	expectedResponses = []string{
		responses[0],
		responses[1],
		responses[2],
		responses[0],
		responses[1],
		responses[2],
	}

	for i, expected := range expectedResponses {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := w.Body.String(); got != expected {
			t.Errorf("after recovery, request %d: expected %q, got %q", i, expected, got)
		}
	}
}
