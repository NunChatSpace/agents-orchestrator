package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type DeploymentPlanController struct {
	service services.DeploymentPlanService
}

func NewDeploymentPlanController(service services.DeploymentPlanService) *DeploymentPlanController {
	return &DeploymentPlanController{service: service}
}

func (c *DeploymentPlanController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	var req domains.CreateDeploymentPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	detail, err := c.service.Create(r.Context(), user.UserID, req)
	if err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	views.JSON(w, http.StatusCreated, toDeploymentPlanResponse(detail), rid)
}

func (c *DeploymentPlanController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	details, err := c.service.List(r.Context(), user.UserID)
	if err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	response := make([]domains.DeploymentPlanResponse, len(details))
	for i, detail := range details {
		response[i] = toDeploymentPlanResponse(detail)
	}
	views.JSON(w, http.StatusOK, response, rid)
}

func (c *DeploymentPlanController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deployment plan id", rid)
		return
	}

	detail, err := c.service.Get(r.Context(), user.UserID, id)
	if err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	views.JSON(w, http.StatusOK, toDeploymentPlanResponse(detail), rid)
}

func (c *DeploymentPlanController) Stop(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deployment plan id", rid)
		return
	}

	detail, err := c.service.Stop(r.Context(), user.UserID, id)
	if err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	views.JSON(w, http.StatusOK, toDeploymentPlanResponse(detail), rid)
}

func (c *DeploymentPlanController) Retry(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deployment plan id", rid)
		return
	}

	detail, err := c.service.Retry(r.Context(), user.UserID, id)
	if err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	views.JSON(w, http.StatusOK, toDeploymentPlanResponse(detail), rid)
}

func (c *DeploymentPlanController) Delete(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid deployment plan id", rid)
		return
	}

	if err := c.service.Delete(r.Context(), user.UserID, id); err != nil {
		handleDeploymentPlanError(w, err, rid)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDeploymentPlanError(w http.ResponseWriter, err error, requestID string) {
	switch {
	case errors.Is(err, services.ErrDeploymentPlanNotFound):
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, services.ErrDeploymentPlanDuplicateName):
		views.Error(w, http.StatusConflict, "CONFLICT", err.Error(), requestID)
	case errors.Is(err, services.ErrDeploymentPlanValidation):
		views.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), requestID)
	case errors.Is(err, services.ErrDeploymentPlanInvalidState):
		views.Error(w, http.StatusUnprocessableEntity, "INVALID_STATE", err.Error(), requestID)
	default:
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), requestID)
	}
}

func toDeploymentPlanResponse(detail *services.DeploymentPlanDetail) domains.DeploymentPlanResponse {
	roles := make([]domains.DeploymentPlanRoleResponse, len(detail.Roles))
	for i, role := range detail.Roles {
		roles[i] = toDeploymentPlanRoleResponse(role)
	}

	return domains.DeploymentPlanResponse{
		ID:         detail.Plan.ID.String(),
		Name:       detail.Plan.Name,
		StackID:    detail.Plan.StackID,
		BuildMode:  string(detail.Plan.BuildMode),
		Status:     string(detail.Plan.Status),
		PreviewURL: detail.Plan.PreviewURL,
		Error:      detail.Plan.Error,
		Roles:      roles,
		CreatedAt:  detail.Plan.CreatedAt,
		UpdatedAt:  detail.Plan.UpdatedAt,
		StoppedAt:  detail.Plan.StoppedAt,
	}
}

func toDeploymentPlanRoleResponse(role *models.DeploymentPlanRole) domains.DeploymentPlanRoleResponse {
	var workerBuildID *string
	if role.WorkerBuildID != nil {
		value := role.WorkerBuildID.String()
		workerBuildID = &value
	}

	return domains.DeploymentPlanRoleResponse{
		ID:                role.ID.String(),
		Role:              role.Role,
		WorkerID:          role.WorkerID.String(),
		WorkerBuildID:     workerBuildID,
		ImageReference:    role.ImageReference,
		ImageDigest:       role.ImageDigest,
		ContainerName:     role.ContainerName,
		BuildStatus:       string(role.BuildStatus),
		DeployStatus:      string(role.DeployStatus),
		WorkerBuildStatus: role.WorkerBuildStatus,
	}
}
