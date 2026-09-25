package orderservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *db.Queries) {
	t.Helper()

	ctx := context.Background()
	if os.Getenv("TEST_DBSTRING") == "" {
		t.Fatal("TEST_DBSTRING environment variable is not set")
	}
	cfg, err := pgxpool.ParseConfig(os.Getenv("TEST_DBSTRING"))
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
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
func createUser(t *testing.T, service *OrderService, ctx context.Context, userName string) *User {
	user, err := service.CreateUser(ctx, &CreateUserRequest{
		Username: userName,
		Password: "password",
		Role:     "user",
	})
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	return user

}
func createOrder(t *testing.T, service *OrderService, ctx context.Context, userID int32, idempotencyKey string) *db.Order {
	order, err := service.CreateOrder(ctx, &CreateOrderRequest{
		UserID:         userID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		t.Fatalf("CreateOrder failed: %v", err)
	}
	return order
}

func createProduct(t *testing.T, service *OrderService, ctx context.Context, name string, priceCents int, description string, inventory int) *Product {
	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        name,
		PriceCents:  priceCents,
		Description: description,
		Inventory:   inventory,
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}
	return product
}

func createOrderItem(t *testing.T, service *OrderService, ctx context.Context, productID int32, orderID int64, quantity int32) *db.OrderItem {
	item, err := service.AddOrderItem(ctx, &AddOrderItemRequest{
		ProductID: productID,
		OrderID:   orderID,
		Quantity:  quantity,
	})
	if err != nil {
		t.Fatalf("AddOrderItem failed: %v", err)
	}
	return item
}
func TestCheckout(t *testing.T) {
	pool, queries := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)
	ctx := context.Background()

	service := NewOrderService(queries, pool)
	user := createUser(t, service, ctx, "test-user")
	order := createOrder(t, service, ctx, user.ID, "idempotency")

	product := createProduct(t, service, ctx, "test-product", 1000, "A test product", 10)

	createOrderItem(t, service, ctx, product.ID, order.ID, 5)

	order, err := service.CheckoutOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("CheckoutOrder failed: %v", err)
	}

	if order.Status != "pending" {
		t.Errorf("Expected order status to be 'pending' after checkout, but got '%s'", order.Status)
	}

	stock, err := service.GetStock(ctx, product.ID)
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if stock != 5 {
		t.Errorf("Expected stock to be 5 after checkout, but got %d", stock)
	}
}

func TestConcurrentCheckouts(t *testing.T) {
	pool, queries := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)
	ctx := context.Background()

	service := NewOrderService(queries, pool)
	user := createUser(t, service, ctx, "test-user")
	ordersSize := 500
	orders := make([]*db.Order, ordersSize)
	for i := 0; i < ordersSize; i++ {
		orders[i] = createOrder(t, service, ctx, user.ID, fmt.Sprintf("idempotency-%d", i))
	}

	product := createProduct(t, service, ctx, "test-product", 1000, "A test product", ordersSize/5)
	for i := 0; i < ordersSize; i++ {
		createOrderItem(t, service, ctx, product.ID, orders[i].ID, 1)
	}

	successes := 0
	insufficientStock := 0
	otherErrors := 0
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	startTIme := time.Now()
	for i := 0; i < ordersSize; i++ {
		wg.Add(1)
		go func(orderID int64) {
			defer wg.Done()
			_, err := service.CheckoutOrder(ctx, orderID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successes++
			} else if errors.Is(err, ErrInsufficientStock) {
				insufficientStock++
			} else {
				t.Logf("Unexpected error during checkout: %v", err)
				otherErrors++
			}
		}(orders[i].ID)
	}
	wg.Wait()
	t.Logf("Successful checkouts: %d", successes)
	t.Logf("Insufficient stock errors: %d", insufficientStock)
	t.Logf("Other errors: %d", otherErrors)
	t.Logf("total time taken for %d orders is %d ms", ordersSize, time.Since(startTIme).Milliseconds())
	if successes != ordersSize/5 {
		t.Errorf("Expected %d successful checkouts, but got %d", ordersSize/5, successes)
	}
	if insufficientStock != ordersSize-ordersSize/5 {
		t.Errorf("Expected %d insufficient stock errors, but got %d", ordersSize-ordersSize/5, insufficientStock)
	}
	if otherErrors != 0 {
		t.Errorf("Expected 0 other errors, but got %d", otherErrors)
	}
}
