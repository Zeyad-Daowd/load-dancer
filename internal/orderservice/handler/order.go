package handler

import (
	"encoding/json"
	"net/http"

	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

type CreateOrderRequest struct {
	UserID int32 `json:"user_id"`
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.handleError(w, ErrMissingIdempotencyKey)
		return
	}
	order, err := h.service.CreateOrder(r.Context(), &service.CreateOrderRequest{
		UserID:         req.UserID,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

type AddOrderItemRequest struct {
	ProductID int32 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
}

func (h *Handler) AddOrderItem(w http.ResponseWriter, r *http.Request) {
	var req AddOrderItemRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	orderID, err := h.getInt64FromPath(r, "orderID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	orderItem, err := h.service.AddOrderItem(r.Context(), &service.AddOrderItemRequest{
		ProductID: req.ProductID,
		OrderID:   orderID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, orderItem)
}

type EditOrderItemRequest struct {
	Quantity int32 `json:"quantity"`
}

func (h *Handler) EditOrderItem(w http.ResponseWriter, r *http.Request) {
	var req EditOrderItemRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	orderID, err := h.getInt64FromPath(r, "orderID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	productID, err := h.getInt32FromPath(r, "productID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	orderItem, err := h.service.EditOrderItemQuantity(r.Context(), &service.EditProductOrderQuantityRequest{
		ProductID: productID,
		OrderID:   orderID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orderItem)
}

func (h *Handler) CheckoutOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := h.getInt64FromPath(r, "orderID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	order, err := h.service.CheckoutOrder(r.Context(), orderID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}

func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userId, err := h.getInt32FromPath(r, "userId")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	orders, err := h.service.GetUserOrders(r.Context(), userId)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, orders)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := h.getInt64FromPath(r, "orderID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	order, err := h.service.GetOrder(r.Context(), orderID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, order)
}
