package repository

import (
	"context"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type userRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE user_id=$1 AND deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE username=$1 AND deleted_at IS NULL`, username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	_, err := r.db.NamedExecContext(ctx,
		`INSERT INTO users (user_id, username, password_hash)
		 VALUES (:user_id, :username, :password_hash)`, user)
	return err
}
