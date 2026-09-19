package control

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	balancer "github.com/zeyad-daowd/load-dancer/internal/balancer"
	middleware "github.com/zeyad-daowd/load-dancer/internal/middleware"
)

type BackendController struct {
	balancer          *balancer.RoundRobin
	Srv               *http.Server
	healthCheckPeriod time.Duration
	ctx               context.Context
}

type RegisterBackendRequest struct {
	URL      string    `json:"url"`
	UniqueID uuid.UUID `json:"uniqueID"`
}

func NewBackendController(ctx context.Context, balancer *balancer.RoundRobin, addr string, healthCheckPeriod time.Duration, secret string) *BackendController {
	mux := http.NewServeMux()

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,   // time to read just the request headers
		ReadTimeout:       10 * time.Second,  // time to read the full request
		WriteTimeout:      10 * time.Second,  // time to write the response
		IdleTimeout:       120 * time.Second, // how long a keep-alive connection may sit idle
	}
	controller := &BackendController{
		balancer:          balancer,
		healthCheckPeriod: healthCheckPeriod,
		Srv:               srv,
		ctx:               ctx,
	}
	mux.Handle("GET /backends", middleware.ValidateAuthorizationHeader(secret)(controller.getBackendsHandler()))
	mux.Handle("POST /backends/register", middleware.ValidateAuthorizationHeader(secret)(controller.addBackendHandler()))
	mux.Handle("DELETE /backends/{uniqueID}", middleware.ValidateAuthorizationHeader(secret)(controller.deleteBackendHandler()))

	return controller
}

func (bc *BackendController) getBackendsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		backends := bc.balancer.GetServers()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err := json.NewEncoder(w).Encode(backends)
		if err != nil {
			// If encoding fails, return an internal server error
			slog.Error("Error encoding backends response", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (bc *BackendController) addBackendHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterBackendRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = bc.balancer.AddServer(bc.ctx, req.URL, req.UniqueID, bc.healthCheckPeriod)
		if err != nil && errors.Is(err, balancer.ErrExistingBackend) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func (bc *BackendController) deleteBackendHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("uniqueID")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "Invalid UUID", http.StatusBadRequest)
			return
		}
		err = bc.balancer.RemoveServer(id)
		if err != nil && errors.Is(err, balancer.ErrBackendNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if err != nil {
			slog.Error("Error removing backend server", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
