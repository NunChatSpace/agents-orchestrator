package repository

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
)

// JobFilters holds optional filters for listing jobs.
type JobFilters struct {
	Status      *models.JobStatus
	TargetGroup *string
	Limit       int
	Offset      int
}

// JobRepository defines data access for jobs.
type JobRepository interface {
	Create(ctx context.Context, job *models.Job) error
	GetByID(ctx context.Context, jobID uuid.UUID) (*models.Job, error)
	List(ctx context.Context, userID uuid.UUID, filters JobFilters) ([]*models.Job, int, error)
	UpdateDraftFields(ctx context.Context, jobID uuid.UUID, title, prompt, targetGroup string) error
	UpdateStatus(ctx context.Context, jobID uuid.UUID, status models.JobStatus) error
	UpdateAssignment(ctx context.Context, jobID uuid.UUID, workerID uuid.UUID, status models.JobStatus) error
	UpdateResumeID(ctx context.Context, jobID uuid.UUID, resumeID string) error
	UpdateFailure(ctx context.Context, jobID uuid.UUID, reason string) error
	SoftDelete(ctx context.Context, jobID uuid.UUID) error
	// DequeueOldest returns the oldest queued job for targetGroup using FOR UPDATE SKIP LOCKED.
	// Must be called inside a transaction.
	DequeueOldest(ctx context.Context, targetGroup string) (*models.Job, error)
}

// WorkerRepository defines data access for workers.
type WorkerRepository interface {
	GetByID(ctx context.Context, workerID uuid.UUID) (*models.Worker, error)
	// GetLRUIdleByGroup returns the idle worker with the oldest last_active_at in the group.
	// Uses FOR UPDATE SKIP LOCKED — must be called inside a transaction.
	GetLRUIdleByGroup(ctx context.Context, groupName string) (*models.Worker, error)
	UpdateStatus(ctx context.Context, workerID uuid.UUID, status models.WorkerStatus) error
	UpdateLastActiveAt(ctx context.Context, workerID uuid.UUID) error
	GetByAPIKeyHash(ctx context.Context, hash string) (*models.Worker, error)
	ListAll(ctx context.Context) ([]*models.Worker, error)
	// GetOrCreateGroup finds an existing group by name or creates it.
	GetOrCreateGroup(ctx context.Context, groupName string) (uuid.UUID, error)
	// CreateWorker inserts a new worker record.
	CreateWorker(ctx context.Context, worker *models.Worker) error
	// UpdateCLICommand changes the cli_command for a worker.
	UpdateCLICommand(ctx context.Context, workerID uuid.UUID, cliCommand string) error
	// UpdateMapPosition updates the worker desk tile coordinates for Office view.
	UpdateMapPosition(ctx context.Context, workerID uuid.UUID, mapX, mapY int) error
	// DeleteWorker soft-deletes a worker by setting deleted_at.
	DeleteWorker(ctx context.Context, workerID uuid.UUID) error
}

// MessageRepository defines data access for messages.
type MessageRepository interface {
	Create(ctx context.Context, msg *models.Message) error
	ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.Message, error)
}

// SessionRepository defines data access for sessions.
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, sessionID string) (*models.Session, error)
	Delete(ctx context.Context, sessionID string) error
	DeleteExpired(ctx context.Context) error
}

// UserRepository defines data access for users.
type UserRepository interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Create(ctx context.Context, user *models.User) error
}

// PlanSessionRepository defines data access for plan sessions and their messages.
type PlanSessionRepository interface {
	Create(ctx context.Context, session *models.PlanSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PlanSession, error)
	List(ctx context.Context, userID uuid.UUID) ([]*models.PlanSession, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.PlanSessionStatus) error
	UpdateResumeID(ctx context.Context, id uuid.UUID, resumeID string) error
	UpdateGeneratedPrompt(ctx context.Context, id uuid.UUID, prompt string) error
	SoftDelete(ctx context.Context, id uuid.UUID) error

	AddMessage(ctx context.Context, msg *models.PlanMessage) error
	ListMessages(ctx context.Context, sessionID uuid.UUID) ([]*models.PlanMessage, error)
}
