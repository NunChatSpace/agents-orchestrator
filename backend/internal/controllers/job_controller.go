package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/middleware"
	"github.com/chatchawan/agent-orchestrator/internal/services"
	"github.com/chatchawan/agent-orchestrator/internal/views"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type JobController struct {
	jobSvc services.JobService
}

func NewJobController(jobSvc services.JobService) *JobController {
	return &JobController{jobSvc: jobSvc}
}

func (c *JobController) List(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	filters := domains.JobFilters{
		Limit:  parseIntQuery(r, "limit", 50),
		Offset: parseIntQuery(r, "offset", 0),
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filters.Status = &s
	}
	if g := r.URL.Query().Get("target_group"); g != "" {
		filters.TargetGroup = &g
	}

	jobs, total, err := c.jobSvc.ListJobs(r.Context(), user.UserID, filters)
	if err != nil {
		views.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), rid)
		return
	}
	views.JSONWithTotal(w, http.StatusOK, jobs, total, rid)
}

func (c *JobController) Create(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	user := middleware.UserFromContext(r.Context())

	var req domains.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	job, err := c.jobSvc.CreateJob(r.Context(), user.UserID, req)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusCreated, job, rid)
}

func (c *JobController) Get(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	job, err := c.jobSvc.GetJob(r.Context(), jobID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found", rid)
		return
	}
	views.JSON(w, http.StatusOK, job, rid)
}

func (c *JobController) Patch(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	var req domains.PatchJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid JSON", rid)
		return
	}

	job, err := c.jobSvc.PatchJob(r.Context(), jobID, req)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrInvalidTransition {
			status = http.StatusConflict
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, job, rid)
}

func (c *JobController) Submit(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	var req struct {
		WorkerID string `json:"worker_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WorkerID == "" {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "worker_id is required", rid)
		return
	}
	workerID, err := uuid.Parse(req.WorkerID)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid worker_id", rid)
		return
	}

	job, err := c.jobSvc.SubmitJob(r.Context(), jobID, workerID)
	if err != nil {
		status := http.StatusConflict
		if err == services.ErrJobNotFound {
			status = http.StatusNotFound
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, job, rid)
}

func (c *JobController) Cancel(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	job, err := c.jobSvc.CancelJob(r.Context(), jobID)
	if err != nil {
		status := http.StatusConflict
		if err == services.ErrJobNotFound {
			status = http.StatusNotFound
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, job, rid)
}

func (c *JobController) Finish(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	job, err := c.jobSvc.FinishJob(r.Context(), jobID)
	if err != nil {
		status := http.StatusConflict
		if err == services.ErrJobNotFound {
			status = http.StatusNotFound
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, job, rid)
}

func (c *JobController) Delete(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job_id", rid)
		return
	}

	if err := c.jobSvc.CloseJob(r.Context(), jobID); err != nil {
		status := http.StatusConflict
		if err == services.ErrJobNotFound {
			status = http.StatusNotFound
		}
		views.Error(w, status, "ERROR", err.Error(), rid)
		return
	}
	views.JSON(w, http.StatusOK, nil, rid)
}

func (c *JobController) GetPatch(w http.ResponseWriter, r *http.Request) {
	rid := reqID(r)
	jobID, err := parseJobID(r)
	if err != nil {
		views.Error(w, http.StatusBadRequest, "INVALID_JOB_ID", "invalid job id", rid)
		return
	}
	data, err := c.jobSvc.GetPatch(r.Context(), jobID)
	if err != nil {
		views.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), rid)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+jobID.String()+`.patch"`)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// helpers
func parseJobID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(mux.Vars(r)["job_id"])
}

func parseIntQuery(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
