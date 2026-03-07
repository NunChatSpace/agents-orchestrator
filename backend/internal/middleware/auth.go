package middleware

import (
	"context"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireSession validates the session cookie and injects the user into context.
func RequireSession(authSvc services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing session cookie", requestID(r))
				return
			}
			user, err := authSvc.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired session", requestID(r))
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext extracts the authenticated user from context.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(UserContextKey).(*models.User)
	return u
}
