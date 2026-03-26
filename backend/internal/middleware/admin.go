package middleware

import (
	"net/http"

	"github.com/kojikokojiko/signalix/internal/ctxkey"
)

// RequireAdmin rejects requests from non-admin users with 403.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, _ := r.Context().Value(ctxkey.IsAdmin).(bool)
		if !isAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
