package repository

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/jmoiron/sqlx"
)

type sessionRepo struct {
	db *sqlx.DB
}

func NewSessionRepo(db *sqlx.DB) SessionRepository {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, session *models.Session) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES (:session_id, :user_id, :expires_at)`, session)
	return err
}

func (r *sessionRepo) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	var s models.Session
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM sessions
		 WHERE session_id=$1 AND deleted_at IS NULL AND expires_at > NOW()`, sessionID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepo) Delete(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET deleted_at=NOW() WHERE session_id=$1`, sessionID)
	return err
}

func (r *sessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE sessions SET deleted_at=NOW()
		 WHERE expires_at < NOW() AND deleted_at IS NULL`)
	return err
}
