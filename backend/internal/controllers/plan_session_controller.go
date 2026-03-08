package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type PlanSessionController struct {
	planSvc services.PlanSessionService
}

func NewPlanSessionController(planSvc services.PlanSessionService) *PlanSessionController {
	return &PlanSessionController{planSvc: planSvc}
}

// List handles GET /plan-sessions
func (c *PlanSessionController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())
	sessions, err := c.planSvc.List(r.Context(), user.UserID)
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, sessions, rid)
}

// Create handles POST /plan-sessions
func (c *PlanSessionController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())
	var req domains.CreatePlanSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WorkerID == "" {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "worker_id is required", rid)
		return
	}
	workerID, err := uuid.Parse(req.WorkerID)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	session, err := c.planSvc.Create(r.Context(), user.UserID, workerID)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, session, rid)
}

// Get handles GET /plan-sessions/{id}
func (c *PlanSessionController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}
	session, err := c.planSvc.Get(r.Context(), id)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, session, rid)
}

// UpdateTitle handles PATCH /plan-sessions/{id}
func (c *PlanSessionController) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}
	var req domains.UpdatePlanSessionTitleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "title is required", rid)
		return
	}
	session, err := c.planSvc.UpdateTitle(r.Context(), id, req.Title)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, session, rid)
}

// SendMessage handles POST /plan-sessions/{id}/message — SSE streaming response.
func (c *PlanSessionController) SendMessage(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}
	var req domains.SendPlanMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Content == "" {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "content is required", rid)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := c.planSvc.SendMessage(r.Context(), w, id, req.Content); err != nil {
		// Write error as SSE event so the frontend can handle it.
		encoded, _ := json.Marshal(err.Error())
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(encoded))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	// Signal completion.
	fmt.Fprintf(w, "event: done\ndata: {}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// Generate handles POST /plan-sessions/{id}/generate — SSE streaming response.
func (c *PlanSessionController) Generate(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := c.planSvc.Generate(r.Context(), w, id); err != nil {
		encoded, _ := json.Marshal(err.Error())
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(encoded))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}

	// Fetch updated session to return generated_prompt in done event.
	session, _ := c.planSvc.Get(r.Context(), id)
	var prompt string
	if session != nil && session.GeneratedPrompt != nil {
		prompt = *session.GeneratedPrompt
	}
	encoded, _ := json.Marshal(prompt)
	fmt.Fprintf(w, "event: done\ndata: %s\n\n", string(encoded))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// Complete handles POST /plan-sessions/{id}/complete
func (c *PlanSessionController) Complete(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}
	session, err := c.planSvc.Complete(r.Context(), id)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, session, rid)
}

// Discard handles POST /plan-sessions/{id}/discard
func (c *PlanSessionController) Discard(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id", rid)
		return
	}
	if err := c.planSvc.Discard(r.Context(), id); err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusNoContent, nil, rid)
}
