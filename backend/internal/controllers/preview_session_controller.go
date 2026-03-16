package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PreviewSessionController struct {
	svc services.PreviewSessionService
}

func NewPreviewSessionController(svc services.PreviewSessionService) *PreviewSessionController {
	return &PreviewSessionController{svc: svc}
}

// List returns all preview sessions for the current user.
func (c *PreviewSessionController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())
	sessions, err := c.svc.List(r.Context(), user.UserID)
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, sessions, rid)
}

// Create starts a new preview session.
func (c *PreviewSessionController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	var req domains.CreatePreviewSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	session, err := c.svc.Create(r.Context(), user.UserID, req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, session, rid)
}

// Get returns a single preview session by ID.
func (c *PreviewSessionController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["session_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid session_id", rid)
		return
	}
	session, err := c.svc.Get(r.Context(), id)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, session, rid)
}

// Stop kills all preview processes in a session and marks it stopped.
func (c *PreviewSessionController) Stop(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["session_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid session_id", rid)
		return
	}
	if err := c.svc.Stop(r.Context(), id); err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{"status": "stopped"}, rid)
}

// Delete stops and removes a preview session.
func (c *PreviewSessionController) Delete(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["session_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid session_id", rid)
		return
	}
	if err := c.svc.Delete(r.Context(), id); err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRoleLog returns the captured stdout/stderr log for a preview role.
func (c *PreviewSessionController) GetRoleLog(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	vars := mux.Vars(r)
	roleID, err := uuid.Parse(vars["role_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid role_id", rid)
		return
	}
	output, err := c.svc.GetRoleLog(r.Context(), roleID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{"log": output}, rid)
}
