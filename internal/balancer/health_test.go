package balancer

import (
	"net/http"
	"testing"
	"time"
)

func TestCheckServerHealth(t *testing.T) {
	backend := createMockBackendServer("OK")
	defer backend.Close()

	server, err := CreateBackendServer(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	failures := 0

	// Successful check keeps the server healthy.
	failures = checkServerHealth(client, server, failures)

	if !server.IsHealthy() {
		t.Error("expected server to be healthy after successful check")
	}

	if failures != 0 {
		t.Errorf("expected failures to be 0, got %d", failures)
	}

	// Make the backend unavailable.
	backend.Close()

	// First failure.
	failures = checkServerHealth(client, server, failures)

	if !server.IsHealthy() {
		t.Error("expected server to remain healthy after first failure")
	}

	if failures != 1 {
		t.Errorf("expected failures to be 1, got %d", failures)
	}

	// Second failure.
	failures = checkServerHealth(client, server, failures)

	if !server.IsHealthy() {
		t.Error("expected server to remain healthy after second failure")
	}

	if failures != 2 {
		t.Errorf("expected failures to be 2, got %d", failures)
	}

	// Third consecutive failure.
	failures = checkServerHealth(client, server, failures)

	if server.IsHealthy() {
		t.Error("expected server to be unhealthy after third consecutive failure")
	}

	if failures != 3 {
		t.Errorf("expected failures to be 3, got %d", failures)
	}
}
func TestCheckServerHealthRecovery(t *testing.T) {
	backend := createMockBackendServer("OK")
	defer backend.Close()

	server, err := CreateBackendServer(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Simulate two previous failures.
	server.SetHealth(false)
	failures := 3
	// Backend is actually healthy again.
	failures = checkServerHealth(client, server, failures)

	if !server.IsHealthy() {
		t.Error("expected server to become healthy after successful check")
	}

	if failures != 0 {
		t.Errorf("expected failures to reset to 0, got %d", failures)
	}
}
