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

func (h *Handler) getInt64FromPath(r *http.Request, paramName string) (int64, error) {
	val := r.PathValue(paramName)
	valInt, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		h.logger.Error("invalid path parameter", "param", paramName, "value", val, "error", err)
		return 0, ErrInvalidPath
	}
	return valInt, nil
}

func (h *Handler) getInt32FromPath(r *http.Request, paramName string) (int32, error) {
	valInt64, err := h.getInt64FromPath(r, paramName)
	if err != nil {
		return 0, err
	}
	return int32(valInt64), nil
}
