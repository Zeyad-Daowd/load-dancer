package handler

import (
	"encoding/json"
	"net/http"

	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

type AddProductRequest struct {
	Name        string `json:"name"`
	PriceCents  int    `json:"price_cents"`
	Description string `json:"description"`
	Inventory   int    `json:"inventory"`
}

func (h *Handler) AddProduct(w http.ResponseWriter, r *http.Request) {
	var req AddProductRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	product, err := h.service.AddProduct(r.Context(), &service.AddProductRequest{
		Name:        req.Name,
		PriceCents:  req.PriceCents,
		Description: req.Description,
		Inventory:   req.Inventory,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, product)
}

func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.GetProducts(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	productId, err := h.getIntFromPath(r, "id")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	products, err := h.service.GetStock(r.Context(), productId)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

type IncreaseProductStockRequest struct {
	StockCount int32 `json:"stock_count"`
}

func (h *Handler) IncreaseProductStock(w http.ResponseWriter, r *http.Request) {
	var req IncreaseProductStockRequest
	productID, err := h.getIntFromPath(r, "id")
	if err != nil {
		h.handleError(w, err)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	updatedStock, err := h.service.IncreaseProductStock(r.Context(), productID, req.StockCount)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updatedStock)
}
