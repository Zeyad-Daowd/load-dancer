package balancer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestNoBackends(t *testing.T) {
	rr := NewRoundRobin([]*BackendServer{})
	handler := rr.Handler()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestAddingServer(t *testing.T) {
	urlStr := "http://localhost:8080"
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uniqueID := uuid.New()
	healthCheckPeriod := 10 * time.Second
	err := rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}
	servers := rr.GetServers()

	if len(servers) != 1 {
		t.Fatalf("Expected 1 server, got %d", len(servers))
	}

	if servers[0].Id != uniqueID {
		t.Fatalf("Expected ID %s, got %s", uniqueID, servers[0].Id)
	}
}

func TestAddingDuplicateServer(t *testing.T) {
	urlStr := "http://localhost:8080"
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uniqueID := uuid.New()
	healthCheckPeriod := 10 * time.Second
	err := rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}
	err = rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != ErrExistingBackend {
		t.Fatalf("Expected ErrExistingBackend, got: %v", err)
	}
}

func TestReaddingRemovedServer(t *testing.T) {
	urlStr := "http://localhost:8080"
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uniqueID := uuid.New()
	healthCheckPeriod := 10 * time.Second
	err := rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}
	err = rr.RemoveServer(uniqueID)
	if err != nil {
		t.Fatalf("Failed to remove server: %v", err)
	}
	err = rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}
}

func TestRemovingServer(t *testing.T) {
	urlStr := "http://localhost:8080"
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uniqueID := uuid.New()

	err := rr.AddServer(ctx, urlStr, uniqueID, 10*time.Second)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	err = rr.RemoveServer(uniqueID)
	if err != nil {
		t.Fatalf("Failed to remove server: %v", err)
	}

	servers := rr.GetServers()

	if len(servers) != 0 {
		t.Fatalf("Expected 0 servers after removal, got %d", len(servers))
	}
}

func TestRemovingNonexistentServer(t *testing.T) {
	rr := NewRoundRobin([]*BackendServer{})

	err := rr.RemoveServer(uuid.New())

	if err != ErrBackendNotFound {
		t.Fatalf("Expected ErrBackendNotFound, got: %v", err)
	}
}

func TestAddedServerIsRoutable(t *testing.T) {
	response := "Response from backend 1"
	server := createMockBackendServer(response)
	defer server.Close()
	urlStr := server.URL
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uniqueID := uuid.New()
	healthCheckPeriod := 10 * time.Second
	err := rr.AddServer(ctx, urlStr, uniqueID, healthCheckPeriod)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	handler := rr.Handler()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	body := w.Body.String()
	expectedResponse := response
	if body != expectedResponse {
		t.Errorf("Expected response: %s, got: %s", expectedResponse, body)
	}
}

func TestRemovedServerIsNotRoutable(t *testing.T) {
	responses := []string{"Response from backend 1", "Response from backend 2"}
	urls := []string{}
	rr := NewRoundRobin([]*BackendServer{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	healthCheckPeriod := 10 * time.Second
	ids := []uuid.UUID{uuid.New(), uuid.New()}
	for i, response := range responses {
		server := createMockBackendServer(response)
		defer server.Close()
		urls = append(urls, server.URL)
		err := rr.AddServer(ctx, server.URL, ids[i], healthCheckPeriod)
		if err != nil {
			t.Fatalf("Failed to add server: %v", err)
		}
	}
	rr.RemoveServer(ids[0]) // Remove the first server
	handler := rr.Handler()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	body := w.Body.String()
	expectedResponse := responses[1]
	if body != expectedResponse {
		t.Errorf("Expected response: %s, got: %s", expectedResponse, body)
	}
}
