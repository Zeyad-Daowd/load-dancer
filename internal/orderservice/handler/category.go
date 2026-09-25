package handler

import (
	"encoding/json"
	"net/http"
)

type AddCategoryRequest struct {
	Name string `json:"name"`
}

func (h *Handler) AddCategory(w http.ResponseWriter, r *http.Request) {
	var req AddCategoryRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}
	category, err := h.service.AddCategory(r.Context(), req.Name)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, category)
}

type AddProductCategoryRequest struct {
	CategoryID int32 `json:"category_id"`
}

func (h *Handler) AddProductCategory(w http.ResponseWriter, r *http.Request) {
	productID, err := h.getInt32FromPath(r, "productID")
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	var req AddProductCategoryRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.handleError(w, ErrInvalidRequest)
		return
	}

	categoryID := req.CategoryID

	resp, err := h.service.AddProductCategory(r.Context(), productID, categoryID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}
