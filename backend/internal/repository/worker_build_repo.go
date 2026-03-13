package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type WorkerBuildRepository interface {
	Create(ctx context.Context, build *models.WorkerBuild) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkerBuild, error)
	ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.WorkerBuildStatus, imageRef, digest, errMsg *string, completedAt *time.Time) error
	AppendBuildLog(ctx context.Context, id uuid.UUID, chunk string) error
	LatestReadyForWorkerRole(ctx context.Context, workerID uuid.UUID, stackID, role string) (*models.WorkerBuild, error)
}

type workerBuildRepo struct {
	db *sqlx.DB
}

func NewWorkerBuildRepo(db *sqlx.DB) WorkerBuildRepository {
	return &workerBuildRepo{db: db}
}

func (r *workerBuildRepo) Create(ctx context.Context, build *models.WorkerBuild) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO worker_builds
		  (id, worker_id, stack_id, role, build_mode, status, callback_token, image_reference, image_digest, error_message, triggered_by, started_at, completed_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		build.ID, build.WorkerID, build.StackID, build.Role, build.BuildMode, build.Status,
		build.CallbackToken,
		build.ImageReference, build.ImageDigest, build.ErrorMessage, build.TriggeredBy,
		build.StartedAt, build.CompletedAt, build.CreatedAt,
	)
	return err
}

func (r *workerBuildRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkerBuild, error) {
	var build models.WorkerBuild
	err := r.db.GetContext(ctx, &build, `
		SELECT id,
		       worker_id,
		       stack_id,
		       role,
		       build_mode,
		       status,
		       callback_token,
		       image_reference,
		       image_digest,
		       error_message,
		       build_log,
		       triggered_by,
		       started_at,
		       completed_at,
		       created_at
		FROM worker_builds
		WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return &build, nil
}

func (r *workerBuildRepo) ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error) {
	var builds []*models.WorkerBuild
	err := r.db.SelectContext(ctx, &builds, `
		SELECT id,
		       worker_id,
		       stack_id,
		       role,
		       build_mode,
		       status,
		       callback_token,
		       image_reference,
		       image_digest,
		       error_message,
		       build_log,
		       triggered_by,
		       started_at,
		       completed_at,
		       created_at
		FROM worker_builds
		WHERE worker_id = $1
		ORDER BY created_at DESC`, workerID)
	return builds, err
}

func (r *workerBuildRepo) AppendBuildLog(ctx context.Context, id uuid.UUID, chunk string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE worker_builds
		SET build_log = COALESCE(build_log, '') || $2
		WHERE id = $1`, id, chunk)
	return err
}

func (r *workerBuildRepo) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status models.WorkerBuildStatus,
	imageRef, digest, errMsg *string,
	completedAt *time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE worker_builds
		SET status = $2,
		    image_reference = $3,
		    image_digest = $4,
		    error_message = $5,
		    started_at = CASE
		        WHEN $2 = 'building' AND started_at IS NULL THEN NOW()
		        ELSE started_at
		    END,
		    completed_at = CASE
		        WHEN $6::TIMESTAMPTZ IS NOT NULL THEN $6::TIMESTAMPTZ
		        ELSE completed_at
		    END
		WHERE id = $1`,
		id, status, imageRef, digest, errMsg, completedAt,
	)
	return err
}

func (r *workerBuildRepo) LatestReadyForWorkerRole(ctx context.Context, workerID uuid.UUID, stackID, role string) (*models.WorkerBuild, error) {
	var build models.WorkerBuild
	err := r.db.GetContext(ctx, &build, `
		SELECT id,
		       worker_id,
		       stack_id,
		       role,
		       build_mode,
		       status,
		       callback_token,
		       image_reference,
		       image_digest,
		       error_message,
		       build_log,
		       triggered_by,
		       started_at,
		       completed_at,
		       created_at
		FROM worker_builds
		WHERE worker_id = $1
		  AND stack_id = $2
		  AND role = $3
		  AND status = 'ready'
		ORDER BY created_at DESC
		LIMIT 1`, workerID, stackID, role)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &build, nil
}

var _ WorkerBuildRepository = (*workerBuildRepo)(nil)
