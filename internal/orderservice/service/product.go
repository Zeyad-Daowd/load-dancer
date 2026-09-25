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

var ErrInvalidProductInventory = fmt.Errorf("invalid product inventory, must be non-negative")

func (s *OrderService) AddProduct(ctx context.Context, req *AddProductRequest) (*Product, error) {
	if req.Inventory < 0 {
		slog.Warn("cannot add negative inventory")
		return nil, ErrInvalidProductInventory
	}
	tx, err := s.pool.Begin(ctx)
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

var ErrIncreaseStockNotPositive = fmt.Errorf("cannot increase stock by non-positive value")

func (s *OrderService) IncreaseProductStock(ctx context.Context, productID int32, stockCount int32) (*StockCount, error) {
	if stockCount <= 0 {
		slog.Warn("cannot increase stock by non-positive value")
		return nil, ErrIncreaseStockNotPositive
	}
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

type ProductItem struct {
	ID          int32
	Name        string
	PriceCents  int32
	Description string
}

type GetProductsFilter struct {
	CategoryID    *int32
	MinPriceCents *int32
	MaxPriceCents *int32
	Sort          string
	Page          int
	Limit         int
}

var ErrInvalidSortValue = fmt.Errorf("invalid sort value")

const (
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
)

func nullableInt(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}
func (s *OrderService) GetProducts(ctx context.Context, filter GetProductsFilter) ([]ProductItem, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 10
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Sort != "" && filter.Sort != SortPriceAsc && filter.Sort != SortPriceDesc {
		return nil, ErrInvalidSortValue
	}
	var res []db.Product
	var err error
	switch filter.Sort {
	case SortPriceAsc:
		res, err = s.queries.GetProductsSortedByPriceAsc(ctx, db.GetProductsSortedByPriceAscParams{
			CategoryID:     nullableInt(filter.CategoryID),
			MinPrice:       nullableInt(filter.MinPriceCents),
			MaxPrice:       nullableInt(filter.MaxPriceCents),
			LimitProducts:  int32(filter.Limit),
			OffsetProducts: int32((filter.Page - 1) * filter.Limit),
		})
	case SortPriceDesc:
		res, err = s.queries.GetProductsSortedByPriceDesc(ctx, db.GetProductsSortedByPriceDescParams{
			CategoryID:     nullableInt(filter.CategoryID),
			MinPrice:       nullableInt(filter.MinPriceCents),
			MaxPrice:       nullableInt(filter.MaxPriceCents),
			LimitProducts:  int32(filter.Limit),
			OffsetProducts: int32((filter.Page - 1) * filter.Limit),
		})
	default:
		res, err = s.queries.GetProducts(ctx, db.GetProductsParams{
			CategoryID:     nullableInt(filter.CategoryID),
			MinPrice:       nullableInt(filter.MinPriceCents),
			MaxPrice:       nullableInt(filter.MaxPriceCents),
			LimitProducts:  int32(filter.Limit),
			OffsetProducts: int32((filter.Page - 1) * filter.Limit),
		})
	}
	if err != nil {
		return nil, err
	}
	products := []ProductItem{}
	for _, prod := range res {
		products = append(products, ProductItem{
			ID:          prod.ID,
			Name:        prod.Name,
			PriceCents:  prod.PriceCents,
			Description: prod.Description.String,
		})
	}
	return products, nil

}

func (s *OrderService) GetStock(ctx context.Context, productID int32) (int32, error) {
	stock, err := s.queries.GetStock(ctx, productID)
	if err != nil {
		return 0, err
	}
	return stock, nil
}
