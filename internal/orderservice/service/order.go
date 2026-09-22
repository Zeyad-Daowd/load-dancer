package orderservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/zeyad-daowd/load-dancer/internal/orderservice/db/generated"
)

type OrderService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

type CreateOrderRequest struct {
	UserID         int32
	IdempotencyKey string
}

func NewOrderService(queries *db.Queries, pool *pgxpool.Pool) *OrderService {
	return &OrderService{
		queries: queries,
		pool:    pool,
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

type AddProductOrderItemRequest struct {
	ProductID int32
	OrderID   int64
	Quantity  int32
}

func (s *OrderService) AddOrderItem(ctx context.Context, req *AddProductOrderItemRequest) (*db.OrderItem, error) {
	if req.Quantity <= 0 {
		slog.Info("quantity must be positive", "quantity", req.Quantity)
		return nil, fmt.Errorf("quantity must be positive")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	txQueries := s.queries.WithTx(tx)

	orderStatus, err := txQueries.GetOrderStatus(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if orderStatus != "started" {
		slog.Info("cannot edit an order that is not in started state", "order_id", req.OrderID, "status", orderStatus)
		return nil, fmt.Errorf("cannot edit an order that is not in started state")
	}
	createdOrder, err := txQueries.AddProductOrderItem(ctx, db.AddProductOrderItemParams{
		ProductID: req.ProductID,
		OrderID:   req.OrderID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return nil, err
	}
	err = txQueries.UpdateOrderTotal(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &createdOrder, nil
}

type EditProductOrderQuantityRequest struct {
	ProductID int32
	OrderID   int64
	Quantity  int32
}

func (s *OrderService) EditOrderItemQuantity(ctx context.Context, req *EditProductOrderQuantityRequest) (*db.OrderItem, error) {
	if req.Quantity < 0 {
		slog.Error("quantity cannot be negative")
		return nil, fmt.Errorf("quantity cannot be negative")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	txQueries := s.queries.WithTx(tx)

	orderStatus, err := txQueries.GetOrderStatus(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	if orderStatus != "started" {
		slog.Info("cannot edit an order that is not in started state", "order_id", req.OrderID, "status", orderStatus)
		return nil, fmt.Errorf("cannot edit an order that is not in started state")
	}

	if req.Quantity == 0 {
		slog.Info("Deleting order item since quantity is zero")
		editedOrder, err := txQueries.RemoveProductQuantityOrderItem(ctx, db.RemoveProductQuantityOrderItemParams{
			ProductID: req.ProductID,
			OrderID:   req.OrderID,
		})
		if err != nil {
			return nil, err
		}
		err = txQueries.UpdateOrderTotal(ctx, req.OrderID)
		if err != nil {
			return nil, err
		}
		err = tx.Commit(ctx)
		if err != nil {
			return nil, err
		}
		return &editedOrder, nil

	}
	editedOrder, err := txQueries.EditProductQuantityOrderItem(ctx, db.EditProductQuantityOrderItemParams{
		ProductID: req.ProductID,
		OrderID:   req.OrderID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return nil, err
	}
	err = txQueries.UpdateOrderTotal(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}
	return &editedOrder, nil
}

func (s *OrderService) CheckoutOrder(ctx context.Context, orderID int64) (*db.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	txQueries := s.queries.WithTx(tx)

	orderStatus, err := txQueries.GetOrderStatus(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if orderStatus != "started" {
		slog.Info("cannot checkout an order that is not in started state", "order_id", orderID, "status", orderStatus)
		return nil, fmt.Errorf("cannot checkout an order that is not in started state")
	}
	orderItems, err := txQueries.GetCheckoutItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if len(orderItems) == 0 {
		return nil, fmt.Errorf("cannot checkout an empty order")
	}
	for _, item := range orderItems {
		if item.RequestedQuantity > item.AvailableStock {
			slog.Info("insufficient stock for product", "product_id", item.ProductID)
			return nil, fmt.Errorf("insufficient stock for product %d", item.ProductID)
		}
		_, err := txQueries.DecreaseInventoryStock(ctx, db.DecreaseInventoryStockParams{
			ProductID:  item.ProductID,
			StockCount: item.RequestedQuantity,
		})
		if err != nil {
			return nil, err
		}
	}
	order, err := txQueries.UpdateOrderStatus(ctx, orderID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	return &order, nil

}

type OrderItem struct {
	ID         int64
	Status     string
	CreatedAt  time.Time
	TotalCents int32
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID int32) ([]OrderItem, error) {
	res, err := s.queries.GetUserOrders(ctx, userID)
	if err != nil {
		return nil, err
	}
	orders := []OrderItem{}
	var createdAt time.Time
	for _, order := range res {
		err := order.CreatedAt.Scan(&createdAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, OrderItem{
			ID:         order.ID,
			Status:     order.Status,
			CreatedAt:  createdAt,
			TotalCents: order.TotalCents,
		})
	}
	return orders, nil
}

type OrderItemDetails struct {
	ID             int64
	ProductID      int32
	OrderID        int64
	Quantity       int32
	UnitPriceCents int32
}

func (s *OrderService) GetOrderItems(ctx context.Context, orderID int64) ([]OrderItemDetails, error) {
	res, err := s.queries.GetOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	orders := []OrderItemDetails{}
	for _, order := range res {
		orders = append(orders, OrderItemDetails{
			ID:             order.ID,
			ProductID:      order.ProductID,
			OrderID:        order.OrderID,
			Quantity:       order.Quantity,
			UnitPriceCents: order.UnitPriceCents,
		})
	}
	return orders, nil
}
