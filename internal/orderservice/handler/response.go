package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

var ErrInvalidRequest = fmt.Errorf("invalid request")

var ErrInvalidPath = fmt.Errorf("invalid path")
var ErrMissingIdempotencyKey = fmt.Errorf("missing idempotency key")

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidProductInventory):
		writeError(w, http.StatusBadRequest, "invalid product inventory, must be non-negative")
	case errors.Is(err, service.ErrIncreaseStockNotPositive):
		writeError(w, http.StatusBadRequest, "increase stock must be positive")
	case errors.Is(err, service.ErrInvalidSortValue):
		writeError(w, http.StatusBadRequest, "invalid sort value for getting products")
	case errors.Is(err, service.ErrInvalidItemQuantity):
		writeError(w, http.StatusBadRequest, "invalid item quantity, must be positive")
	case errors.Is(err, service.ErrCannotCheckoutEmptyOrder):
		writeError(w, http.StatusBadRequest, "cannot checkout empty order")
	case errors.Is(err, service.ErrCannotEditNotStartedOrder):
		writeError(w, http.StatusConflict, "cannot edit order item, order is not in started state")
	case errors.Is(err, service.ErrCannotAddProductCategory):
		writeError(w, http.StatusBadRequest, "cannot add product category")
	case errors.Is(err, service.ErrInvalidCategoryName):
		writeError(w, http.StatusBadRequest, "invalid category name")
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrMissingIdempotencyKey):
		writeError(w, http.StatusBadRequest, "missing idempotency key")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, service.ErrInvalidCreateUserRequest):
		writeError(w, http.StatusBadRequest, "invalid create user request")
	case errors.Is(err, service.ErrCannoutCheckoutOrder):
		writeError(w, http.StatusConflict, "cannot checkout order that is not in started state")
	case errors.Is(err, service.ErrInsufficientStock):
		writeError(w, http.StatusConflict, "insufficient stock for product")
	default:
		h.logger.Error("unhandled error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
