package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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
func (h *Handler) extractGetProductsParams(r *http.Request) (*service.GetProductsFilter, error) {
	query := r.URL.Query()
	page := query.Get("page")
	if page == "" {
		page = "1"
	}

	pageInt, err := strconv.Atoi(page)
	if err != nil {
		return nil, err
	}
	limit := query.Get("limit")
	if limit == "" {
		limit = "0"
	}
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		return nil, err
	}
	var minPricePtr *int32 = nil
	minPrice := query.Get("min_price")
	if minPrice != "" {
		minPriceInt64, err := strconv.ParseInt(minPrice, 10, 32)
		if err != nil {
			return nil, err
		}
		minPriceInt := int32(minPriceInt64)
		minPricePtr = &minPriceInt
	}
	maxPrice := query.Get("max_price")
	var maxPricePtr *int32 = nil
	if maxPrice != "" {
		maxPriceInt64, err := strconv.ParseInt(maxPrice, 10, 32)
		if err != nil {
			return nil, err
		}
		maxPriceInt := int32(maxPriceInt64)
		maxPricePtr = &maxPriceInt
	}
	categoryID := query.Get("categoryID")
	var categoryPtr *int32 = nil
	if categoryID != "" {
		categoryIDInt64, err := strconv.ParseInt(categoryID, 10, 32)
		if err != nil {
			return nil, err
		}
		categoryIDInt := int32(categoryIDInt64)
		categoryPtr = &categoryIDInt
	}
	sort := query.Get("sort")
	filter := service.GetProductsFilter{
		CategoryID:    categoryPtr,
		MinPriceCents: minPricePtr,
		MaxPriceCents: maxPricePtr,
		Sort:          sort,
		Page:          pageInt,
		Limit:         limitInt,
	}
	return &filter, nil
}
func (h *Handler) GetProducts(w http.ResponseWriter, r *http.Request) {
	filter, err := h.extractGetProductsParams(r)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	products, err := h.service.GetProducts(r.Context(), *filter)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (h *Handler) GetStock(w http.ResponseWriter, r *http.Request) {
	productId, err := h.getInt32FromPath(r, "id")
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
	productID, err := h.getInt32FromPath(r, "id")
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
