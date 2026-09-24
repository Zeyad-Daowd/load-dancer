package orderservice

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	ctx := context.Background()
	if os.Getenv("TEST_DBSTRING") == "" {
		t.Fatal("TEST_DBSTRING environment variable is not set")
	}
	pool, err := pgxpool.New(ctx, os.Getenv("TEST_DBSTRING"))
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}

	return pool, db.New(pool)
}
func cleanupTestDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(
		context.Background(),
		`TRUNCATE
			order_items,
			orders,
			inventory,
			products,
			categories,
			product_categories,
			users
			RESTART IDENTITY CASCADE`,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateOrder(t *testing.T) {
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
		t.Fatal(err)
	}
	ord, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if ord.UserID != user.ID || ord.IdempotencyKey != "test-key" {
		t.Errorf("Unexpected order returned: %+v", ord)
	}

}

func TestCreateOrderIdempotency(t *testing.T) {
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
		t.Fatal(err)
	}
	ord, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if ord.UserID != user.ID || ord.IdempotencyKey != "test-key" {
		t.Errorf("Unexpected order returned: %+v", ord)
	}

	ordRepeat, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})

	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if ordRepeat.ID != ord.ID || ordRepeat.UserID != ord.UserID || ordRepeat.IdempotencyKey != ord.IdempotencyKey {
		t.Errorf("Expected the same order to be returned for the same idempotency key, but got different orders: %+v and %+v", ord, ordRepeat)
	}

	otherUser, err := service.CreateUser(ctx, &CreateUserRequest{
		Username: "test-user-2",
		Password: "password",
		Role:     "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	ordOtherUser, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         otherUser.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	if ordOtherUser.ID == ord.ID {
		t.Errorf("Expected a different order to be returned for a different user with the same idempotency key, but got the same order: %+v and %+v", ord, ordOtherUser)
	}

}

func TestAddOrderItem(t *testing.T) {
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

	order, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        "test-product",
		PriceCents:  1000,
		Description: "A test product",
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	item, err := service.AddOrderItem(ctx, &AddOrderItemRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  2,
	})
	if err != nil {
		t.Fatalf("AddOrderItem failed: %v", err)
	}

	if item.ProductID != product.ID || item.OrderID != order.ID || item.Quantity != 2 {
		t.Errorf("Unexpected order item returned: %+v", item)
	}
}

func TestAddOrderItemNegativeQuantity(t *testing.T) {
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

	order, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        "test-product",
		PriceCents:  1000,
		Description: "A test product",
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	_, err = service.AddOrderItem(ctx, &AddOrderItemRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  -1,
	})
	if err == nil {
		t.Fatalf("Expected error when adding order item with negative quantity, but got none")
	}
}

func TestEditOrderItemQuantity(t *testing.T) {
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

	order, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        "test-product",
		PriceCents:  1000,
		Description: "A test product",
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	_, err = service.AddOrderItem(ctx, &AddOrderItemRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  5,
	})
	if err != nil {
		t.Fatalf("AddOrderItem failed: %v", err)
	}

	// Now edit the order item quantity
	editedItem, err := service.EditOrderItemQuantity(ctx, &EditOrderItemQuantityRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  10,
	})
	if err != nil {
		t.Fatalf("EditOrderItemQuantity failed: %v", err)
	}

	if editedItem.Quantity != 10 {
		t.Errorf("Expected quantity to be updated to 10, but got %d", editedItem.Quantity)
	}
}

func TestEditOrderItemQuantityNegative(t *testing.T) {
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

	order, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         user.ID,
		IdempotencyKey: "test-key",
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}

	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        "test-product",
		PriceCents:  1000,
		Description: "A test product",
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	_, err = service.AddOrderItem(ctx, &AddOrderItemRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  5,
	})
	if err != nil {
		t.Fatalf("AddOrderItem failed: %v", err)
	}

	// Now edit the order item quantity
	_, err = service.EditOrderItemQuantity(ctx, &EditOrderItemQuantityRequest{
		ProductID: product.ID,
		OrderID:   order.ID,
		Quantity:  -2,
	})
	if err == nil {
		t.Fatalf("Expected error when editing order item with negative quantity, but got none")
	}

}
