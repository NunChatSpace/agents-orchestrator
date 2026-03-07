package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	SessionID string     `db:"session_id"`
	UserID    uuid.UUID  `db:"user_id"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt time.Time  `db:"expires_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}
