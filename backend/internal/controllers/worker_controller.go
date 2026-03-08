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

type WorkerController struct {
	jobSvc     services.JobService
	workerSvc  services.WorkerService
	dispatcher services.DispatcherService
}

func NewWorkerController(jobSvc services.JobService, workerSvc services.WorkerService, dispatcher services.DispatcherService) *WorkerController {
	return &WorkerController{jobSvc: jobSvc, workerSvc: workerSvc, dispatcher: dispatcher}
}

// Reply handles POST /workers/reply from a WAgent.
func (c *WorkerController) Reply(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	var req domains.WorkerReplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	if req.JobID == "" || req.WorkerID == "" || req.Status == "" {
		views.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "job_id, worker_id, and status are required", rid)
		return
	}

	if err := c.jobSvc.HandleWorkerReply(r.Context(), req); err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, nil, rid)
}

// List handles GET /workers — returns all workers with current status.
func (c *WorkerController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workers, err := c.workerSvc.ListWorkers(r.Context())
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, workers, rid)
}

// Create handles POST /workers — registers a new agent (clones repo, creates workspace).
func (c *WorkerController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	var req domains.CreateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	worker, err := c.workerSvc.CreateWorker(r.Context(), req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, worker, rid)
}

// Get handles GET /workers/{worker_id} — returns a single worker.
func (c *WorkerController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	worker, err := c.workerSvc.GetWorker(r.Context(), workerID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, worker, rid)
}

// Ping handles POST /workers/{worker_id}/ping — runs {cli_command} --version.
func (c *WorkerController) Ping(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	result, err := c.workerSvc.PingWorker(r.Context(), workerID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, result, rid)
}

// Delete handles DELETE /workers/{worker_id} — soft-deletes the worker.
func (c *WorkerController) Delete(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	if err := c.workerSvc.DeleteWorker(r.Context(), workerID); err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusNoContent, nil, rid)
}

// Update handles PATCH /workers/{worker_id} — updates worker settings (cli_command, map_x, map_y).
func (c *WorkerController) Update(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	var req domains.UpdateWorkerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	worker, err := c.workerSvc.UpdateWorker(r.Context(), workerID, req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, worker, rid)
}

// ListGroups handles GET /groups — returns all worker groups.
func (c *WorkerController) ListGroups(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	groups, err := c.workerSvc.ListGroups(r.Context())
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, groups, rid)
}

// CreateGroup handles POST /groups — creates a group by name (idempotent).
func (c *WorkerController) CreateGroup(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	group, err := c.workerSvc.CreateGroup(r.Context(), req.Name)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, group, rid)
}

// Plan handles POST /workers/{worker_id}/plan — runs the CLI in plan mode and returns a refined prompt.
func (c *WorkerController) Plan(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}
	var req struct {
		Task string `json:"task"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Task == "" {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "task is required", rid)
		return
	}
	prompt, err := c.dispatcher.RunPlan(r.Context(), workerID, req.Task)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "PLAN_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, map[string]string{"prompt": prompt}, rid)
}
