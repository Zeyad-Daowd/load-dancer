package handler

import (
	"log/slog"

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
