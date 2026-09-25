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

func (s *OrderService) AddProductCategory(ctx context.Context, productID int32, categoryID int32) error {
	_, err := s.queries.AddProductCategory(ctx, db.AddProductCategoryParams{
		ProductID:  productID,
		CategoryID: categoryID,
	})
	if err != nil {
		return err
	}
	return nil
}
