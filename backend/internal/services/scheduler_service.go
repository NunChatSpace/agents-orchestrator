package services

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type schedulerService struct {
	jobRepo    repository.JobRepository
	workerRepo repository.WorkerRepository
	msgRepo    repository.MessageRepository
	dispatcher DispatcherService
	hub        WSBroadcaster
}

func NewSchedulerService(
	jobRepo repository.JobRepository,
	workerRepo repository.WorkerRepository,
	msgRepo repository.MessageRepository,
	dispatcher DispatcherService,
	hub WSBroadcaster,
) SchedulerService {
	return &schedulerService{
		jobRepo:    jobRepo,
		workerRepo: workerRepo,
		msgRepo:    msgRepo,
		dispatcher: dispatcher,
		hub:        hub,
	}
}

// TryAssignJob picks an idle worker for the given queued job and dispatches it.
func (s *schedulerService) TryAssignJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status != models.JobStatusQueued {
		return nil // already assigned or cancelled
	}

	var worker *models.Worker

	// manual_worker_override takes priority
	if job.ManualWorkerOverride != nil {
		w, err := s.workerRepo.GetByID(ctx, *job.ManualWorkerOverride)
		if err == nil && w.Status == models.WorkerStatusIdle {
			worker = w
		}
	}

	// LRU idle worker in target group
	if worker == nil {
		w, err := s.workerRepo.GetLRUIdleByGroup(ctx, job.TargetGroup)
		if err != nil {
			log.Debug().Str("group", job.TargetGroup).Msg("no idle worker available")
			return nil // stay queued
		}
		worker = w
	}

	// Assign
	if err := s.jobRepo.UpdateAssignment(ctx, jobID, worker.WorkerID, models.JobStatusAssigned); err != nil {
		return err
	}
	if err := s.workerRepo.UpdateStatus(ctx, worker.WorkerID, models.WorkerStatusBusy); err != nil {
		return err
	}
	if err := s.workerRepo.UpdateLastActiveAt(ctx, worker.WorkerID); err != nil {
		return err
	}

	// Instruction message
	_ = s.msgRepo.Create(ctx, &models.Message{
		MessageID: uuid.New(),
		JobID:     jobID,
		Role:      models.MessageRoleOAgent,
		Kind:      models.MessageKindInstruction,
		Content:   job.Prompt,
	})

	// Dispatch to WAgent
	if err := s.dispatcher.SendToWorker(ctx, worker.WorkerID, job); err != nil {
		log.Error().Err(err).Str("worker", worker.Name).Msg("dispatch failed")
		// Mark job as failed
		_ = s.jobRepo.UpdateFailure(ctx, jobID, "dispatch_error")
		_ = s.workerRepo.UpdateStatus(ctx, worker.WorkerID, models.WorkerStatusIdle)
	}

	s.hub.BroadcastJobUpdated(ctx, jobID)
	s.hub.BroadcastWorkerUpdated(ctx, worker.WorkerID)

	return nil
}

// ProcessQueue assigns the oldest queued job in targetGroup to an idle worker.
func (s *schedulerService) ProcessQueue(ctx context.Context, targetGroup string) error {
	job, err := s.jobRepo.DequeueOldest(ctx, targetGroup)
	if err != nil {
		return nil // no queued jobs
	}
	return s.TryAssignJob(ctx, job.JobID)
}
