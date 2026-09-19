package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
)

func ValidateAuthorizationHeader(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Error("Missing Authorization header")
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			expectedHeader := "Bearer " + secret

			if len(authHeader) != len(expectedHeader) {
				slog.Error("Invalid Authorization header length")
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}

			if subtle.ConstantTimeCompare([]byte(authHeader), []byte(expectedHeader)) != 1 {
				slog.Error("Invalid Authorization header")
				http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
