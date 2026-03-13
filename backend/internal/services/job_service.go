package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrJobNotFound       = errors.New("job not found")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidTransition = errors.New("invalid status transition")
)

type JobService interface {
	CreateJob(ctx context.Context, userID uuid.UUID, req domains.CreateJobRequest) (*domains.JobResponse, error)
	GetJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error)
	ListJobs(ctx context.Context, userID uuid.UUID, filters domains.JobFilters) ([]*domains.JobResponse, int, error)
	PatchJob(ctx context.Context, jobID uuid.UUID, req domains.PatchJobRequest) (*domains.JobResponse, error)
	SubmitJob(ctx context.Context, jobID uuid.UUID, workerID uuid.UUID) (*domains.JobResponse, error)
	CancelJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error)
	FinishJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error)
	CloseJob(ctx context.Context, jobID uuid.UUID) error
	SendUserMessage(ctx context.Context, jobID uuid.UUID, content string) (*domains.MessageResponse, error)
	ListMessages(ctx context.Context, jobID uuid.UUID) ([]*domains.MessageResponse, error)
	HandleWorkerReply(ctx context.Context, req domains.WorkerReplyRequest) error
	HandleThinkingStep(ctx context.Context, jobID uuid.UUID, content string) error
	HandleFileChange(ctx context.Context, jobID uuid.UUID, content string) error
	GetPatch(ctx context.Context, jobID uuid.UUID) ([]byte, error)
	// ws.JobFetcher / ws.MessageFetcher implementations for hub broadcast payloads.
	GetJobJSON(ctx context.Context, jobID uuid.UUID) (any, error)
	GetMessageJSON(ctx context.Context, msg *models.Message) any
}

type jobService struct {
	jobRepo    repository.JobRepository
	msgRepo    repository.MessageRepository
	workerRepo repository.WorkerRepository
	scheduler  SchedulerService
	dispatcher DispatcherService
	hub        WSBroadcaster
}

func NewJobService(
	jobRepo repository.JobRepository,
	msgRepo repository.MessageRepository,
	workerRepo repository.WorkerRepository,
	scheduler SchedulerService,
	dispatcher DispatcherService,
	hub WSBroadcaster,
) JobService {
	return &jobService{
		jobRepo:    jobRepo,
		msgRepo:    msgRepo,
		workerRepo: workerRepo,
		scheduler:  scheduler,
		dispatcher: dispatcher,
		hub:        hub,
	}
}

func (s *jobService) CreateJob(ctx context.Context, userID uuid.UUID, req domains.CreateJobRequest) (*domains.JobResponse, error) {
	if req.Prompt == "" || req.TargetGroup == "" {
		return nil, errors.New("prompt and target_group are required")
	}
	title := req.Title
	if title == "" {
		title = truncate(req.Prompt, 60)
	}

	job := &models.Job{
		JobID:       uuid.New(),
		UserID:      userID,
		Title:       title,
		Prompt:      req.Prompt,
		TargetGroup: req.TargetGroup,
		Status:      models.JobStatusDraft,
	}
	if req.ManualWorkerOverride != nil {
		wid, err := uuid.Parse(*req.ManualWorkerOverride)
		if err == nil {
			job.ManualWorkerOverride = &wid
		}
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}
	return s.toResponse(ctx, job), nil
}

func (s *jobService) GetJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	return s.toResponse(ctx, job), nil
}

func (s *jobService) GetJobJSON(ctx context.Context, jobID uuid.UUID) (any, error) {
	return s.GetJob(ctx, jobID)
}

func (s *jobService) GetMessageJSON(_ context.Context, msg *models.Message) any {
	return &domains.MessageResponse{
		MessageID: msg.MessageID.String(),
		JobID:     msg.JobID.String(),
		Role:      string(msg.Role),
		Kind:      string(msg.Kind),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func (s *jobService) ListJobs(ctx context.Context, userID uuid.UUID, filters domains.JobFilters) ([]*domains.JobResponse, int, error) {
	repoFilters := repository.JobFilters{
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}
	if filters.Status != nil {
		s := models.JobStatus(*filters.Status)
		repoFilters.Status = &s
	}
	repoFilters.TargetGroup = filters.TargetGroup

	jobs, total, err := s.jobRepo.List(ctx, userID, repoFilters)
	if err != nil {
		return nil, 0, err
	}
	resp := make([]*domains.JobResponse, len(jobs))
	for i, j := range jobs {
		resp[i] = s.toResponse(ctx, j)
	}
	return resp, total, nil
}

func (s *jobService) PatchJob(ctx context.Context, jobID uuid.UUID, req domains.PatchJobRequest) (*domains.JobResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.Status != models.JobStatusDraft {
		return nil, ErrInvalidTransition
	}
	if req.Title != nil {
		job.Title = *req.Title
	}
	if req.Prompt != nil {
		job.Prompt = *req.Prompt
	}
	if req.TargetGroup != nil {
		job.TargetGroup = *req.TargetGroup
	}
	if err := s.jobRepo.UpdateDraftFields(ctx, jobID, job.Title, job.Prompt, job.TargetGroup); err != nil {
		return nil, err
	}
	return s.toResponse(ctx, job), nil
}

func (s *jobService) SubmitJob(ctx context.Context, jobID uuid.UUID, workerID uuid.UUID) (*domains.JobResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.Status != models.JobStatusDraft {
		return nil, ErrInvalidTransition
	}

	// Assign directly to the specified worker.
	if err := s.jobRepo.UpdateAssignment(ctx, jobID, workerID, models.JobStatusAssigned); err != nil {
		return nil, err
	}
	if err := s.workerRepo.UpdateStatus(ctx, workerID, models.WorkerStatusBusy); err != nil {
		return nil, err
	}
	_ = s.workerRepo.UpdateLastActiveAt(ctx, workerID)

	instructionMsg := &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleOAgent,
		Kind:      models.MessageKindInstruction,
		Content:   job.Prompt,
	}
	_ = s.msgRepo.Create(ctx, instructionMsg)
	s.hub.BroadcastMessageAdded(ctx, jobID, instructionMsg)

	job.AssignedWorkerID = &workerID
	job.Status = models.JobStatusAssigned

	// Dispatch asynchronously.
	go func() {
		if err := s.dispatcher.SendToWorker(context.Background(), workerID, job); err != nil {
			_ = s.jobRepo.UpdateFailure(context.Background(), jobID, "dispatch_error")
			_ = s.workerRepo.UpdateStatus(context.Background(), workerID, models.WorkerStatusIdle)
		}
		s.hub.BroadcastJobUpdated(context.Background(), jobID)
		s.hub.BroadcastWorkerUpdated(context.Background(), workerID)
	}()

	s.hub.BroadcastJobUpdated(ctx, jobID)
	s.hub.BroadcastWorkerUpdated(ctx, workerID)

	return s.toResponse(ctx, job), nil
}

func (s *jobService) CancelJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}

	allowed := map[models.JobStatus]bool{
		models.JobStatusAssigned:    true,
		models.JobStatusBusy:        true,
		models.JobStatusPendingUser: true,
	}
	if !allowed[job.Status] {
		return nil, ErrInvalidTransition
	}

	if job.AssignedWorkerID != nil {
		go s.dispatcher.CancelWorkerJob(context.Background(), *job.AssignedWorkerID, jobID)
	}

	if err := s.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusCancelled); err != nil {
		return nil, err
	}
	if job.AssignedWorkerID != nil {
		_ = s.workerRepo.UpdateStatus(ctx, *job.AssignedWorkerID, models.WorkerStatusIdle)
		_ = s.workerRepo.UpdateLastActiveAt(ctx, *job.AssignedWorkerID)
		go s.scheduler.ProcessQueue(context.Background(), job.TargetGroup)
	}

	_ = s.msgRepo.Create(ctx, &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleOAgent,
		Kind:      models.MessageKindSystem,
		Content:   "Job cancelled by user.",
	})

	s.hub.BroadcastJobUpdated(ctx, jobID)

	job.Status = models.JobStatusCancelled
	return s.toResponse(ctx, job), nil
}

// FinishJob marks an active job as done and releases the worker.
// Allowed from: assigned, busy, pending_user, done.
func (s *jobService) FinishJob(ctx context.Context, jobID uuid.UUID) (*domains.JobResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}

	allowed := map[models.JobStatus]bool{
		models.JobStatusAssigned:    true,
		models.JobStatusBusy:        true,
		models.JobStatusPendingUser: true,
		models.JobStatusDone:        true,
	}
	if !allowed[job.Status] {
		return nil, ErrInvalidTransition
	}

	if job.AssignedWorkerID != nil {
		go s.dispatcher.CancelWorkerJob(context.Background(), *job.AssignedWorkerID, jobID)
		_ = s.workerRepo.UpdateStatus(ctx, *job.AssignedWorkerID, models.WorkerStatusIdle)
		_ = s.workerRepo.UpdateLastActiveAt(ctx, *job.AssignedWorkerID)
		go s.scheduler.ProcessQueue(context.Background(), job.TargetGroup)
	}

	_ = s.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusDone)
	_ = s.msgRepo.Create(ctx, &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleOAgent,
		Kind:      models.MessageKindSystem,
		Content:   "Job marked as done by user.",
	})

	s.hub.BroadcastJobUpdated(ctx, jobID)
	if job.AssignedWorkerID != nil {
		s.hub.BroadcastWorkerUpdated(ctx, *job.AssignedWorkerID)
	}

	job.Status = models.JobStatusDone
	return s.toResponse(ctx, job), nil
}

func (s *jobService) CloseJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return ErrJobNotFound
	}
	closeable := map[models.JobStatus]bool{
		models.JobStatusDone:      true,
		models.JobStatusFailed:    true,
		models.JobStatusCancelled: true,
	}
	if !closeable[job.Status] {
		return ErrInvalidTransition
	}
	return s.jobRepo.SoftDelete(ctx, jobID)
}

func (s *jobService) ListMessages(ctx context.Context, jobID uuid.UUID) ([]*domains.MessageResponse, error) {
	msgs, err := s.msgRepo.ListByJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	resp := make([]*domains.MessageResponse, len(msgs))
	for i, m := range msgs {
		resp[i] = toMessageResponse(m)
	}
	return resp, nil
}

func (s *jobService) SendUserMessage(ctx context.Context, jobID uuid.UUID, content string) (*domains.MessageResponse, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.Status != models.JobStatusPendingUser && job.Status != models.JobStatusDone {
		return nil, ErrInvalidTransition
	}

	msg := &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleUser,
		Kind:      models.MessageKindAnswer,
		Content:   content,
	}
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	if err := s.jobRepo.UpdateStatus(ctx, jobID, models.JobStatusBusy); err != nil {
		return nil, err
	}

	if job.AssignedWorkerID != nil {
		resumeID := ""
		if job.ResumeID != nil {
			resumeID = *job.ResumeID
		}
		_ = s.workerRepo.UpdateStatus(ctx, *job.AssignedWorkerID, models.WorkerStatusBusy)
		go s.dispatcher.NotifyUserReply(context.Background(), *job.AssignedWorkerID, jobID, resumeID, content)
	}

	s.hub.BroadcastMessageAdded(ctx, jobID, msg)
	s.hub.BroadcastJobUpdated(ctx, jobID)

	return toMessageResponse(msg), nil
}

func (s *jobService) HandleWorkerReply(ctx context.Context, req domains.WorkerReplyRequest) error {
	jobID, err := uuid.Parse(req.JobID)
	if err != nil {
		return errors.New("invalid job_id")
	}
	workerID, err := uuid.Parse(req.WorkerID)
	if err != nil {
		return errors.New("invalid worker_id")
	}

	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return ErrJobNotFound
	}

	// Map worker status → job status + message kind
	var newJobStatus models.JobStatus
	var msgKind models.MessageKind
	var failureReason string

	switch req.Status {
	case "busy":
		newJobStatus = models.JobStatusBusy
		msgKind = models.MessageKindAnswer
	case "pending_user":
		newJobStatus = models.JobStatusPendingUser
		msgKind = models.MessageKindQuestion
	case "idle":
		newJobStatus = models.JobStatusPendingUser
		msgKind = models.MessageKindAnswer
	case "offline":
		newJobStatus = models.JobStatusFailed
		msgKind = models.MessageKindSystem
		failureReason = "worker_error"
	default:
		return errors.New("unknown worker status")
	}

	// Insert worker message
	msg := &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleWorker,
		Kind:      msgKind,
		Content:   req.Message,
	}
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return err
	}

	// Update job status
	if failureReason != "" {
		_ = s.jobRepo.UpdateFailure(ctx, jobID, failureReason)
	} else {
		_ = s.jobRepo.UpdateStatus(ctx, jobID, newJobStatus)
	}

	// Update resume_id
	if req.ResumeID != nil {
		_ = s.jobRepo.UpdateResumeID(ctx, jobID, *req.ResumeID)
	}

	// Update worker status.
	// CLI subprocess workers report "offline" on error but are still alive;
	// reset them to idle so they can accept new jobs immediately.
	workerStatus := models.WorkerStatus(req.Status)
	if workerStatus == models.WorkerStatusOffline {
		workerStatus = models.WorkerStatusIdle
	}
	_ = s.workerRepo.UpdateStatus(ctx, workerID, workerStatus)
	_ = s.workerRepo.UpdateLastActiveAt(ctx, workerID)

	// Broadcast
	s.hub.BroadcastMessageAdded(ctx, jobID, msg)
	s.hub.BroadcastJobUpdated(ctx, jobID)
	s.hub.BroadcastWorkerUpdated(ctx, workerID)

	// If worker is now idle, try to assign next queued job
	if workerStatus == models.WorkerStatusIdle {
		go s.scheduler.ProcessQueue(context.Background(), job.TargetGroup)
	}

	return nil
}

// toResponse converts a job model to a JobResponse DTO.
func (s *jobService) toResponse(ctx context.Context, job *models.Job) *domains.JobResponse {
	msgs, _ := s.msgRepo.ListByJobID(ctx, job.JobID)

	resp := &domains.JobResponse{
		JobID:         job.JobID.String(),
		Title:         job.Title,
		Prompt:        job.Prompt,
		TargetGroup:   job.TargetGroup,
		Status:        string(job.Status),
		ResumeID:      job.ResumeID,
		FailureReason: job.FailureReason,
		MessageCount:  len(msgs),
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
	}
	if job.AssignedWorkerID != nil {
		id := job.AssignedWorkerID.String()
		resp.AssignedWorkerID = &id
		// Fetch worker name
		if w, err := s.workerRepo.GetByID(ctx, *job.AssignedWorkerID); err == nil {
			resp.AssignedWorkerName = &w.Name
		}
	}
	if job.ManualWorkerOverride != nil {
		id := job.ManualWorkerOverride.String()
		resp.ManualWorkerOverride = &id
	}
	return resp
}

func toMessageResponse(msg *models.Message) *domains.MessageResponse {
	return &domains.MessageResponse{
		MessageID: msg.MessageID.String(),
		JobID:     msg.JobID.String(),
		Role:      string(msg.Role),
		Kind:      string(msg.Kind),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt,
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// HandleFileChange stores a file-change event emitted by the agent and broadcasts it via WS.
func (s *jobService) HandleFileChange(ctx context.Context, jobID uuid.UUID, content string) error {
	msg := &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleWorker,
		Kind:      models.MessageKindFileChange,
		Content:   content,
	}
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return err
	}
	s.hub.BroadcastMessageAdded(ctx, jobID, msg)
	return nil
}

// HandleThinkingStep stores a streaming thinking step and broadcasts it via WS.
func (s *jobService) HandleThinkingStep(ctx context.Context, jobID uuid.UUID, content string) error {
	msg := &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleWorker,
		Kind:      models.MessageKindThinking,
		Content:   content,
	}
	if err := s.msgRepo.Create(ctx, msg); err != nil {
		return err
	}
	s.hub.BroadcastMessageAdded(ctx, jobID, msg)
	return nil
}

// GetPatch reads the .patch file saved by captureGitDiff for the given job.
func (s *jobService) GetPatch(ctx context.Context, jobID uuid.UUID) ([]byte, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.AssignedWorkerID == nil {
		return nil, fmt.Errorf("job has no assigned worker")
	}
	worker, err := s.workerRepo.GetByID(ctx, *job.AssignedWorkerID)
	if err != nil {
		return nil, fmt.Errorf("worker not found")
	}
	patchPath := filepath.Join(worker.Workspace, ".agent", "patches", jobID.String()+".patch")
	data, err := os.ReadFile(patchPath)
	if err != nil {
		return nil, fmt.Errorf("patch not found")
	}
	return data, nil
}

// WSBroadcaster is the interface the job service uses to push events.
// Implemented by the WS hub in Phase 5.
type WSBroadcaster interface {
	BroadcastJobUpdated(ctx context.Context, jobID uuid.UUID)
	BroadcastMessageAdded(ctx context.Context, jobID uuid.UUID, msg *models.Message)
	BroadcastWorkerUpdated(ctx context.Context, workerID uuid.UUID)
}

// SchedulerService is defined in scheduler_service.go.
type SchedulerService interface {
	TryAssignJob(ctx context.Context, jobID uuid.UUID) error
	ProcessQueue(ctx context.Context, targetGroup string) error
}

// DispatcherService is defined in dispatcher_service.go.
type DispatcherService interface {
	SendToWorker(ctx context.Context, workerID uuid.UUID, job *models.Job) error
	NotifyUserReply(ctx context.Context, workerID uuid.UUID, jobID uuid.UUID, resumeID string, message string) error
	CancelWorkerJob(ctx context.Context, workerID uuid.UUID, jobID uuid.UUID) error
	SetResultHandler(h JobResultHandler)
	RunBuildInstruction(ctx context.Context, workerID uuid.UUID, buildID uuid.UUID, message string, logFn func(string)) error
	RunPlan(ctx context.Context, workerID uuid.UUID, task string) (string, error)
	// RunPlanStream streams the agent reply via SSE to w. Returns the full reply text and new resume_id.
	RunPlanStream(ctx context.Context, w http.ResponseWriter, workerID uuid.UUID, resumeID string, field models.WorkerInstructionField, message string) (reply string, newResumeID string, err error)
}
