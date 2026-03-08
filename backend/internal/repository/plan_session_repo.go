package repository

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type planSessionRepo struct {
	db *sqlx.DB
}

func NewPlanSessionRepo(db *sqlx.DB) PlanSessionRepository {
	return &planSessionRepo{db: db}
}

func (r *planSessionRepo) Create(ctx context.Context, session *models.PlanSession) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO plan_sessions (id, user_id, worker_id, title, status)
		VALUES (:id, :user_id, :worker_id, :title, :status)`, session)
	return err
}

func (r *planSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PlanSession, error) {
	var s models.PlanSession
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM plan_sessions WHERE id=$1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *planSessionRepo) List(ctx context.Context, userID uuid.UUID) ([]*models.PlanSession, error) {
	var sessions []*models.PlanSession
	err := r.db.SelectContext(ctx, &sessions, `
		SELECT * FROM plan_sessions
		WHERE user_id=$1 AND status != 'discarded' AND deleted_at IS NULL
		ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *planSessionRepo) UpdateTitle(ctx context.Context, id uuid.UUID, title string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plan_sessions SET title=$1 WHERE id=$2 AND deleted_at IS NULL`, title, id)
	return err
}

func (r *planSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PlanSessionStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plan_sessions SET status=$1 WHERE id=$2 AND deleted_at IS NULL`, status, id)
	return err
}

func (r *planSessionRepo) UpdateResumeID(ctx context.Context, id uuid.UUID, resumeID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plan_sessions SET resume_id=$1 WHERE id=$2 AND deleted_at IS NULL`, resumeID, id)
	return err
}

func (r *planSessionRepo) UpdateGeneratedPrompt(ctx context.Context, id uuid.UUID, prompt string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plan_sessions SET generated_prompt=$1 WHERE id=$2 AND deleted_at IS NULL`, prompt, id)
	return err
}

func (r *planSessionRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE plan_sessions SET deleted_at=NOW(), status='discarded' WHERE id=$1`, id)
	return err
}

func (r *planSessionRepo) AddMessage(ctx context.Context, msg *models.PlanMessage) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO plan_messages (id, plan_session_id, role, content)
		VALUES (:id, :plan_session_id, :role, :content)`, msg)
	return err
}

func (r *planSessionRepo) ListMessages(ctx context.Context, sessionID uuid.UUID) ([]*models.PlanMessage, error) {
	var msgs []*models.PlanMessage
	err := r.db.SelectContext(ctx, &msgs, `
		SELECT * FROM plan_messages
		WHERE plan_session_id=$1
		ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	return msgs, nil
}
