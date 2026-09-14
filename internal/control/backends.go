package control

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	balancer "github.com/zeyad-daowd/load-dancer/internal/balancer"
)

type BackendController struct {
	balancer          *balancer.RoundRobin
	Srv               *http.Server
	healthCheckPeriod time.Duration
}

func NewBackendController(balancer *balancer.RoundRobin, addr string, healthCheckPeriod time.Duration) *BackendController {
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
	}
	mux.Handle("GET /backends", controller.getBackendsHandler())
	mux.Handle("POST /backends/register", controller.addBackendHandler())

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

type RegisterBackendRequest struct {
	URL string `json:"url"`
}

func (bc *BackendController) addBackendHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterBackendRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = bc.balancer.AddServer(req.URL, bc.healthCheckPeriod)
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
