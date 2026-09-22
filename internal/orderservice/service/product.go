package orderservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

type AddProductRequest struct {
	Name        string
	PriceCents  int
	Description string
	Inventory   int
}

type Product struct {
	ID          int32
	Name        string
	PriceCents  int32
	Description string
	Inventory   int32
}

type StockCount struct {
	ProductID  int32
	StockCount int32
}

func (s *OrderService) AddProduct(ctx context.Context, req *AddProductRequest) (*Product, error) {
	if req.Inventory < 0 {
		slog.Warn("cannot add negative inventory")
		return nil, fmt.Errorf("cannot add negative inventory")
	}
	tx, err := s.connection.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	txQueries := s.queries.WithTx(tx)
	addedProduct, err := txQueries.AddProduct(ctx, db.AddProductParams{
		Name:        req.Name,
		PriceCents:  int32(req.PriceCents),
		Description: pgtype.Text{String: req.Description, Valid: true},
	})

	if err != nil {
		return nil, err
	}

	product := Product{
		ID:          addedProduct.ID,
		Name:        addedProduct.Name,
		PriceCents:  addedProduct.PriceCents,
		Description: addedProduct.Description.String,
		Inventory:   0,
	}
	inventory, err := txQueries.InsertInventory(ctx, db.InsertInventoryParams{
		ProductID:  addedProduct.ID,
		StockCount: int32(req.Inventory),
	})
	if err != nil {
		return nil, err
	}
	product.Inventory = inventory.StockCount
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (s *OrderService) IncreaseProductStock(ctx context.Context, productID int32, stockCount int32) (*StockCount, error) {
	updatedInventory, err := s.queries.IncreaseProductStock(ctx, db.IncreaseProductStockParams{
		ProductID:  productID,
		StockCount: stockCount,
	})

	if err != nil {
		return nil, err
	}

	return &StockCount{
		ProductID:  updatedInventory.ProductID,
		StockCount: updatedInventory.StockCount,
	}, nil
}
