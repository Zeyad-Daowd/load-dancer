package orderservice

import (
	"context"
	"testing"
)

func TestCreateUser(t *testing.T) {
	pool, queries := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)
	ctx := context.Background()

	service := NewOrderService(queries, pool)
	user, err := service.CreateUser(ctx, &CreateUserRequest{
		Username: "test-user",
		Password: "password",
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if user.Username != "test-user" || user.Role != "user" {
		t.Errorf("Unexpected user returned: %+v", user)
	}
}
