package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
)

type AuthController struct {
	authSvc services.AuthService
}

func NewAuthController(authSvc services.AuthService) *AuthController {
	return &AuthController{authSvc: authSvc}
}

func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	var req domains.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	if req.Username == "" || req.Password == "" {
		views.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "username and password required", rid)
		return
	}

	sessionID, user, err := c.authSvc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		views.Error(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid username or password", rid)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
	views.JSON(w, http.StatusOK, domains.UserResponse{
		UserID:   user.UserID.String(),
		Username: user.Username,
	}, rid)
}

func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	cookie, err := r.Cookie("session_id")
	if err == nil {
		_ = c.authSvc.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:    "session_id",
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
	views.JSON(w, http.StatusOK, nil, rid)
}

func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())
	if user == nil {
		views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated", rid)
		return
	}
	views.JSON(w, http.StatusOK, domains.UserResponse{
		UserID:   user.UserID.String(),
		Username: user.Username,
	}, rid)
}

func reqID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}
