package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type WorkerBuildController struct {
	Service services.WorkerBuildService
}

func NewWorkerBuildController(service services.WorkerBuildService) *WorkerBuildController {
	return &WorkerBuildController{Service: service}
}

func (c *WorkerBuildController) TriggerBuild(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}

	var req domains.TriggerWorkerBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	if req.StackID == "" || req.Role == "" || req.Mode == "" {
		views.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "stack_id, role, and mode are required", rid)
		return
	}

	build, err := c.Service.TriggerBuild(r.Context(), workerID, req.StackID, req.Role, req.Mode, user.UserID)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}

	views.JSON(w, http.StatusCreated, toWorkerBuildResponse(build), rid)
}

func (c *WorkerBuildController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)

	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}

	builds, err := c.Service.ListByWorker(r.Context(), workerID)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}

	response := make([]domains.WorkerBuildResponse, len(builds))
	for i, build := range builds {
		response[i] = toWorkerBuildResponse(build)
	}
	views.JSON(w, http.StatusOK, response, rid)
}

func (c *WorkerBuildController) ReportBuild(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)

	workerID, err := uuid.Parse(mux.Vars(r)["worker_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}

	token := r.Header.Get("X-Build-Token")
	if token == "" {
		views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing X-Build-Token", rid)
		return
	}

	var req domains.WorkerBuildReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}
	req.CallbackToken = token

	build, err := c.Service.ReportBuild(r.Context(), workerID, req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}

	views.JSON(w, http.StatusOK, toWorkerBuildResponse(build), rid)
}

func (c *WorkerBuildController) GetBuild(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	id, err := uuid.Parse(mux.Vars(r)["build_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid build_id", rid)
		return
	}
	build, err := c.Service.GetByID(r.Context(), id)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", "worker build not found", rid)
		return
	}
	views.JSON(w, http.StatusOK, toWorkerBuildResponse(build), rid)
}

func toWorkerBuildResponse(build *models.WorkerBuild) domains.WorkerBuildResponse {
	return domains.WorkerBuildResponse{
		ID:             build.ID.String(),
		WorkerID:       build.WorkerID.String(),
		StackID:        build.StackID,
		Role:           build.Role,
		BuildMode:      string(build.BuildMode),
		Status:         string(build.Status),
		ImageReference: build.ImageReference,
		ImageDigest:    build.ImageDigest,
		ErrorMessage:   build.ErrorMessage,
		BuildLog:       build.BuildLog,
		CreatedAt:      build.CreatedAt,
		CompletedAt:    build.CompletedAt,
	}
}
