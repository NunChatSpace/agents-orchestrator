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

type PreviewBundleController struct {
	previewSvc services.PreviewBundleService
}

func NewPreviewBundleController(previewSvc services.PreviewBundleService) *PreviewBundleController {
	return &PreviewBundleController{previewSvc: previewSvc}
}

func (c *PreviewBundleController) ListStacks(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	stacks, err := c.previewSvc.ListStacks(r.Context())
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, stacks, rid)
}

func (c *PreviewBundleController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())
	bundles, err := c.previewSvc.List(r.Context(), user.UserID)
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, bundles, rid)
}

func (c *PreviewBundleController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	var req domains.CreatePreviewBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	bundle, err := c.previewSvc.Create(r.Context(), user.UserID, req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, bundle, rid)
}

func (c *PreviewBundleController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	bundleID, err := uuid.Parse(mux.Vars(r)["bundle_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid bundle_id", rid)
		return
	}

	bundle, err := c.previewSvc.Get(r.Context(), user.UserID, bundleID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, bundle, rid)
}

func (c *PreviewBundleController) Destroy(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	bundleID, err := uuid.Parse(mux.Vars(r)["bundle_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid bundle_id", rid)
		return
	}

	bundle, err := c.previewSvc.Destroy(r.Context(), user.UserID, bundleID)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, bundle, rid)
}

func (c *PreviewBundleController) ReportBuild(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	worker := middleware.WorkerFromContext(r.Context())
	if worker == nil {
		views.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "worker authentication required", rid)
		return
	}

	bundleID, err := uuid.Parse(mux.Vars(r)["bundle_id"])
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid bundle_id", rid)
		return
	}

	var req domains.ReportPreviewBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	bundle, err := c.previewSvc.ReportBuild(r.Context(), worker.WorkerID, bundleID, req)
	if err != nil {
		views.Error(w, http.StatusUnprocessableEntity, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, bundle, rid)
}
