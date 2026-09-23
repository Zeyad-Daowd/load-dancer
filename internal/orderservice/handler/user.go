package handler

import (
	"encoding/json"
	"net/http"

	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.CreateUser(r.Context(), &service.CreateUserRequest{
		Username: req.Username,
		Password: req.Password,
		Role:     "user", // default role
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}
