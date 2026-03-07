package repository

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type messageRepo struct {
	db *sqlx.DB
}

func NewMessageRepo(db *sqlx.DB) MessageRepository {
	return &messageRepo{db: db}
}

func (r *messageRepo) Create(ctx context.Context, msg *models.Message) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO messages (message_id, job_id, role, kind, content)
		VALUES (:message_id, :job_id, :role, :kind, :content)`, msg)
	return err
}

func (r *messageRepo) ListByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.Message, error) {
	var msgs []*models.Message
	err := r.db.SelectContext(ctx, &msgs,
		`SELECT * FROM messages WHERE job_id=$1 ORDER BY created_at ASC, message_id ASC`, jobID)
	return msgs, err
}
