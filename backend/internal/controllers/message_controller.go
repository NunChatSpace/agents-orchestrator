package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type MessageController struct {
	jobSvc services.JobService
}

func NewMessageController(jobSvc services.JobService) *MessageController {
	return &MessageController{jobSvc: jobSvc}
}

func (c *MessageController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := uuid.Parse(mux.Vars(r)["job_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	msgs, err := c.jobSvc.ListMessages(r.Context(), jobID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found", rid)
		return
	}
	views.JSON(w, http.StatusOK, msgs, rid)
}

func (c *MessageController) Send(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := uuid.Parse(mux.Vars(r)["job_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	var req domains.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	if req.Content == "" {
		views.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "content is required", rid)
		return
	}

	msg, err := c.jobSvc.SendUserMessage(r.Context(), jobID, req.Content)
	if err != nil {
		status := http.StatusConflict
		if err == services.ErrJobNotFound {
			status = http.StatusNotFound
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, msg, rid)
}
