package handler

import (
	"net/http"
)

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Got health check to backend server", "url", h.address)
	w.WriteHeader(http.StatusOK)
}
