package orderservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
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
	if err == nil {
		return &createdOrder, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	createdOrder, err = s.queries.GetOrderByUserIdAndIdempotencyKey(ctx, db.GetOrderByUserIdAndIdempotencyKeyParams{
		UserID:         req.UserID,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return &createdOrder, nil
}

type AddOrderItemRequest struct {
	ProductID int32
	OrderID   int64
	Quantity  int32
}

func (s *OrderService) AddOrderItem(ctx context.Context, req *AddOrderItemRequest) (*db.OrderItem, error) {
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

type EditOrderItemQuantityRequest struct {
	ProductID int32
	OrderID   int64
	Quantity  int32
}

func (s *OrderService) EditOrderItemQuantity(ctx context.Context, req *EditOrderItemQuantityRequest) (*db.OrderItem, error) {
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

var ErrInsufficientStock = fmt.Errorf("insufficient stock for product")

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
			err := ErrInsufficientStock
			err = fmt.Errorf("%w: product_id %d", err, item.ProductID)
			return nil, err
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

type Order struct {
	ID         int64
	Status     string
	CreatedAt  time.Time
	TotalCents int32
}

func (s *OrderService) GetUserOrders(ctx context.Context, userID int32) ([]Order, error) {
	res, err := s.queries.GetUserOrders(ctx, userID)
	if err != nil {
		return nil, err
	}
	orders := []Order{}
	for _, order := range res {
		orders = append(orders, Order{
			ID:         order.ID,
			Status:     order.Status,
			CreatedAt:  order.CreatedAt.Time,
			TotalCents: order.TotalCents,
		})
	}
	return orders, nil
}

type OrderItemDetails struct {
	ProductID      int32
	ProductName    string
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
			ProductID:      order.ProductID,
			OrderID:        order.OrderID,
			Quantity:       order.Quantity,
			UnitPriceCents: order.UnitPriceCents,
			ProductName:    order.ProductName,
		})
	}
	return orders, nil
}

type OrderDetails struct {
	ID         int64
	Status     string
	CreatedAt  time.Time
	TotalCents int32
	Items      []OrderItemDetails
}

func (s *OrderService) GetOrder(ctx context.Context, orderID int64) (*OrderDetails, error) {
	res, err := s.queries.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("order not found")
	}
	order := OrderDetails{}
	order.ID = res[0].ID
	order.Status = res[0].Status
	order.TotalCents = res[0].TotalCents

	order.CreatedAt = res[0].CreatedAt.Time
	order.Items = []OrderItemDetails{}
	for _, item := range res {
		order.Items = append(order.Items, OrderItemDetails{
			ProductID:      item.ProductID,
			OrderID:        item.OrderID,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
			ProductName:    item.Name,
		})
	}
	return &order, nil
}
