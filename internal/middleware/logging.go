package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIdKey contextKey = "requestId"
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

func GetRequestId(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestId, ok := ctx.Value(requestIdKey).(string)
	if ok {
		return requestId
	}
	return ""
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		initialTime := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		requestContext := r.Context()
		uid := uuid.NewString()
		requestContext = context.WithValue(requestContext, requestIdKey, uid)
		r = r.WithContext(requestContext)
		next.ServeHTTP(rec, r)
		slog.Info("Request processed", "method", r.Method, "url", r.URL.String(), "duration", time.Since(initialTime), "status", rec.status, "requestId", uid)
	})
}
