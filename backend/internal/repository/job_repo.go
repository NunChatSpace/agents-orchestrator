package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type jobRepo struct {
	db *sqlx.DB
}

func NewJobRepo(db *sqlx.DB) JobRepository {
	return &jobRepo{db: db}
}

func (r *jobRepo) Create(ctx context.Context, job *models.Job) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO jobs (job_id, user_id, title, prompt, target_group,
		                  assigned_worker_id, manual_worker_override, status)
		VALUES (:job_id, :user_id, :title, :prompt, :target_group,
		        :assigned_worker_id, :manual_worker_override, :status)`, job)
	return err
}

func (r *jobRepo) GetByID(ctx context.Context, jobID uuid.UUID) (*models.Job, error) {
	var j models.Job
	err := r.db.GetContext(ctx, &j,
		`SELECT * FROM jobs WHERE job_id=$1 AND deleted_at IS NULL`, jobID)
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *jobRepo) List(ctx context.Context, userID uuid.UUID, filters JobFilters) ([]*models.Job, int, error) {
	args := []interface{}{userID}
	conds := []string{"user_id=$1", "deleted_at IS NULL"}
	i := 2

	if filters.Status != nil {
		conds = append(conds, fmt.Sprintf("status=$%d", i))
		args = append(args, *filters.Status)
		i++
	}
	if filters.TargetGroup != nil {
		conds = append(conds, fmt.Sprintf("target_group=$%d", i))
		args = append(args, *filters.TargetGroup)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	// Count
	var total int
	if err := r.db.GetContext(ctx, &total,
		"SELECT COUNT(*) FROM jobs "+where, args...); err != nil {
		return nil, 0, err
	}

	// Paginate
	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf("SELECT * FROM jobs %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d",
		where, i, i+1)
	args = append(args, limit, filters.Offset)

	var jobs []*models.Job
	if err := r.db.SelectContext(ctx, &jobs, query, args...); err != nil {
		return nil, 0, err
	}
	return jobs, total, nil
}

func (r *jobRepo) UpdateDraftFields(ctx context.Context, jobID uuid.UUID, title, prompt, targetGroup string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET title=$1, prompt=$2, target_group=$3 WHERE job_id=$4 AND status='draft'`,
		title, prompt, targetGroup, jobID)
	return err
}

func (r *jobRepo) UpdateStatus(ctx context.Context, jobID uuid.UUID, status models.JobStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET status=$1 WHERE job_id=$2`, status, jobID)
	return err
}

func (r *jobRepo) UpdateAssignment(ctx context.Context, jobID uuid.UUID, workerID uuid.UUID, status models.JobStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET assigned_worker_id=$1, status=$2 WHERE job_id=$3`,
		workerID, status, jobID)
	return err
}

func (r *jobRepo) UpdateResumeID(ctx context.Context, jobID uuid.UUID, resumeID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET resume_id=$1 WHERE job_id=$2`, resumeID, jobID)
	return err
}

func (r *jobRepo) UpdateFailure(ctx context.Context, jobID uuid.UUID, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET status='failed', failure_reason=$1 WHERE job_id=$2`, reason, jobID)
	return err
}

func (r *jobRepo) SoftDelete(ctx context.Context, jobID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE jobs SET deleted_at=NOW() WHERE job_id=$1`, jobID)
	return err
}

func (r *jobRepo) DequeueOldest(ctx context.Context, targetGroup string) (*models.Job, error) {
	var j models.Job
	err := r.db.GetContext(ctx, &j, `
		SELECT * FROM jobs
		WHERE target_group=$1
		  AND status='queued'
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, targetGroup)
	if err != nil {
		return nil, err
	}
	return &j, nil
}
