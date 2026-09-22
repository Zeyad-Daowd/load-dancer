package orderservice

import (
	"context"

	"github.com/jackc/pgx/v5"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

type OrderService struct {
	queries    *db.Queries
	connection *pgx.Conn
}

type CreateOrderRequest struct {
	UserID         int32
	IdempotencyKey string
}

func NewOrderService(queries *db.Queries) *OrderService {
	return &OrderService{
		queries: queries,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*db.Order, error) {
	createdOrder, err := s.queries.AddOrder(ctx, db.AddOrderParams{
		UserID:         req.UserID,
		IdempotencyKey: req.IdempotencyKey,
		TotalCents:     0,
	})
	if err != nil {
		return nil, err
	}
	return &createdOrder, nil
}
