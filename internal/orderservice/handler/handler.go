package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	service "github.com/zeyad-daowd/load-dancer/internal/orderservice/service"
)

type Handler struct {
	logger  *slog.Logger
	service *service.OrderService
	address string
}

func NewOrderServiceHandler(logger *slog.Logger, service *service.OrderService, address string) *Handler {
	Handler := &Handler{
		logger:  logger,
		service: service,
		address: address,
	}
	return Handler
}

func (h *Handler) getIntFromPath(r *http.Request, paramName string) (int32, error) {
	id := r.PathValue(paramName)
	idInt, err := strconv.Atoi(id)
	if err != nil {
		h.logger.Error("invalid path parameter", "param", paramName, "value", id, "error", err)
		return 0, ErrInvalidPath
	}
	return int32(idInt), nil
}
