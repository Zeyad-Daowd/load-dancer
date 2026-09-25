package orderservice

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

func (s *OrderService) AddCategory(ctx context.Context, categoryName string) (int32, error) {
	if categoryName == "" {
		slog.Warn("cannot add category with empty name")
		return -1, fmt.Errorf("cannot add category with empty name")
	}
	category, err := s.queries.InsertCategory(ctx, categoryName)
	if err != nil {
		return -1, err
	}
	return category.ID, nil

}

type ProductCategoryResponse struct {
	ProductID  int32 `json:"product_id"`
	CategoryID int32 `json:"category_id"`
}

func (s *OrderService) AddProductCategory(ctx context.Context, productID int32, categoryID int32) (*ProductCategoryResponse, error) {
	resp, err := s.queries.AddProductCategory(ctx, db.AddProductCategoryParams{
		ProductID:  productID,
		CategoryID: categoryID,
	})
	if err != nil {
		return nil, err
	}
	return &ProductCategoryResponse{
		ProductID:  resp.ProductID,
		CategoryID: resp.CategoryID,
	}, nil
}
