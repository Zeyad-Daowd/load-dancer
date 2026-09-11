package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	// Embed the original ResponseWriter (and its methods) to capture the status code
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initialTime := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("Request processed", "method", r.Method, "url", r.URL.String(), "duration", time.Since(initialTime), "status", rec.status)
	})
}
