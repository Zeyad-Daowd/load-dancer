package orderservice

import (
	"context"
	"testing"
)

func TestAddProduct(t *testing.T) {
	pool, queries := setupTestDB(t)
	defer pool.Close()
	defer cleanupTestDB(t, pool)
	ctx := context.Background()

	service := NewOrderService(queries, pool)
	product, err := service.AddProduct(ctx, &AddProductRequest{
		Name:        "test-product",
		PriceCents:  1000,
		Description: "A test product",
		Inventory:   10,
	})
	if err != nil {
		t.Fatalf("AddProduct failed: %v", err)
	}

	if product.Name != "test-product" || product.PriceCents != 1000 || product.Description != "A test product" {
		t.Errorf("Unexpected product returned: %+v", product)
	}
	stock, err := service.GetStock(ctx, product.ID)
	if err != nil {
		t.Fatalf("GetStock failed: %v", err)
	}
	if stock != 10 {
		t.Errorf("Unexpected stock count returned: %d", stock)
	}
}
