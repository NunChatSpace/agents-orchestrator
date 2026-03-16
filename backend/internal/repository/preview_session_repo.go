package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PreviewSessionRepository defines data access for preview sessions and their roles.
type PreviewSessionRepository interface {
	Create(ctx context.Context, session *models.PreviewSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PreviewSession, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PreviewSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.PreviewSessionStatus, errMsg *string, stoppedAt *time.Time) error

	CreateRole(ctx context.Context, role *models.PreviewSessionRole) error
	GetRolesBySession(ctx context.Context, sessionID uuid.UUID) ([]*models.PreviewSessionRole, error)
	GetRoleByWorkerAndSession(ctx context.Context, workerID uuid.UUID, sessionID uuid.UUID) (*models.PreviewSessionRole, error)
	UpdateRoleStatus(ctx context.Context, id uuid.UUID, status models.PreviewRoleStatus, port *int, previewURL *string, errMsg *string) error
	AppendRoleLog(ctx context.Context, id uuid.UUID, lines string) error
	GetRoleLog(ctx context.Context, id uuid.UUID) (string, error)
	ListRunningRoles(ctx context.Context) ([]*models.PreviewSessionRole, error)
	AllocatePort(ctx context.Context) (int, error)
}

type previewSessionRepo struct {
	db *sqlx.DB
}

func NewPreviewSessionRepo(db *sqlx.DB) PreviewSessionRepository {
	return &previewSessionRepo{db: db}
}

func (r *previewSessionRepo) Create(ctx context.Context, session *models.PreviewSession) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO preview_sessions (id, created_by, status, error, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		session.ID, session.CreatedBy, session.Status, session.Error,
		session.CreatedAt, session.UpdatedAt)
	return err
}

func (r *previewSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PreviewSession, error) {
	var s models.PreviewSession
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM preview_sessions WHERE id=$1`, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *previewSessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PreviewSession, error) {
	var sessions []*models.PreviewSession
	err := r.db.SelectContext(ctx, &sessions,
		`SELECT * FROM preview_sessions WHERE created_by=$1 ORDER BY created_at DESC`, userID)
	return sessions, err
}

func (r *previewSessionRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PreviewSessionStatus, errMsg *string, stoppedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE preview_sessions SET status=$1, error=$2, stopped_at=$3, updated_at=NOW() WHERE id=$4`,
		status, errMsg, stoppedAt, id)
	return err
}

func (r *previewSessionRepo) CreateRole(ctx context.Context, role *models.PreviewSessionRole) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO preview_session_roles (id, session_id, worker_id, role, port, status, error_message, preview_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		role.ID, role.SessionID, role.WorkerID, role.Role, role.Port,
		role.Status, role.ErrorMessage, role.PreviewURL, role.CreatedAt, role.UpdatedAt)
	return err
}

func (r *previewSessionRepo) GetRolesBySession(ctx context.Context, sessionID uuid.UUID) ([]*models.PreviewSessionRole, error) {
	var roles []*models.PreviewSessionRole
	err := r.db.SelectContext(ctx, &roles,
		`SELECT * FROM preview_session_roles WHERE session_id=$1 ORDER BY created_at ASC`, sessionID)
	return roles, err
}

func (r *previewSessionRepo) GetRoleByWorkerAndSession(ctx context.Context, workerID uuid.UUID, sessionID uuid.UUID) (*models.PreviewSessionRole, error) {
	var role models.PreviewSessionRole
	err := r.db.GetContext(ctx, &role,
		`SELECT * FROM preview_session_roles WHERE worker_id=$1 AND session_id=$2`, workerID, sessionID)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *previewSessionRepo) UpdateRoleStatus(ctx context.Context, id uuid.UUID, status models.PreviewRoleStatus, port *int, previewURL *string, errMsg *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE preview_session_roles SET status=$1, port=$2, preview_url=$3, error_message=$4, updated_at=NOW() WHERE id=$5`,
		status, port, previewURL, errMsg, id)
	return err
}

func (r *previewSessionRepo) GetRoleLog(ctx context.Context, id uuid.UUID) (string, error) {
	var output string
	err := r.db.QueryRowContext(ctx,
		`SELECT log_output FROM preview_session_roles WHERE id=$1`, id).Scan(&output)
	return output, err
}

func (r *previewSessionRepo) AppendRoleLog(ctx context.Context, id uuid.UUID, lines string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE preview_session_roles SET log_output = log_output || $1, updated_at=NOW() WHERE id=$2`,
		lines, id)
	return err
}

func (r *previewSessionRepo) ListRunningRoles(ctx context.Context) ([]*models.PreviewSessionRole, error) {
	var roles []*models.PreviewSessionRole
	err := r.db.SelectContext(ctx, &roles,
		`SELECT * FROM preview_session_roles WHERE status='running'`)
	return roles, err
}

func (r *previewSessionRepo) AllocatePort(ctx context.Context) (int, error) {
	var ports []int
	if err := r.db.SelectContext(ctx, &ports,
		`SELECT port FROM preview_session_roles WHERE port IS NOT NULL`); err != nil {
		return 0, fmt.Errorf("allocate port: %w", err)
	}
	taken := make(map[int]bool, len(ports))
	for _, p := range ports {
		taken[p] = true
	}
	for p := 8300; p < 9000; p++ {
		if !taken[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no available preview ports in range 8300-8999")
}
