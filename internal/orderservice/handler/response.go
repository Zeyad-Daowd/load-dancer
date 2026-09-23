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
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, "invalid request")
	case errors.Is(err, ErrMissingIdempotencyKey):
		writeError(w, http.StatusBadRequest, "missing idempotency key")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, service.ErrInvalidCreateUserRequest):
		writeError(w, http.StatusBadRequest, "invalid create user request")
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
