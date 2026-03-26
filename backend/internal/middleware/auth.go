package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kojikokojiko/signalix/internal/ctxkey"
	"github.com/kojikokojiko/signalix/internal/usecase"
)

// Authenticate validates the Bearer token and stores claims in context.
func Authenticate(uc *usecase.AuthUsecase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := uc.ParseAccessToken(tokenStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), ctxkey.UserID, claims.Subject)
			ctx = context.WithValue(ctx, ctxkey.Email, claims.Email)
			ctx = context.WithValue(ctx, ctxkey.IsAdmin, claims.IsAdmin)
			ctx = context.WithValue(ctx, ctxkey.JTI, claims.JTI)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message})
}
