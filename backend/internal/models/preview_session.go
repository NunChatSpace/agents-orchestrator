package models

import (
	"time"

	"github.com/google/uuid"
)

type PreviewSessionStatus string

const (
	PreviewSessionStatusStarting PreviewSessionStatus = "starting"
	PreviewSessionStatusRunning  PreviewSessionStatus = "running"
	PreviewSessionStatusStopped  PreviewSessionStatus = "stopped"
	PreviewSessionStatusFailed   PreviewSessionStatus = "failed"
)

type PreviewRoleStatus string

const (
	PreviewRoleStatusStarting PreviewRoleStatus = "starting"
	PreviewRoleStatusRunning  PreviewRoleStatus = "running"
	PreviewRoleStatusStopped  PreviewRoleStatus = "stopped"
	PreviewRoleStatusFailed   PreviewRoleStatus = "failed"
)

type PreviewSession struct {
	ID        uuid.UUID            `db:"id"`
	CreatedBy uuid.UUID            `db:"created_by"`
	Status    PreviewSessionStatus `db:"status"`
	Error     *string              `db:"error"`
	CreatedAt time.Time            `db:"created_at"`
	UpdatedAt time.Time            `db:"updated_at"`
	StoppedAt *time.Time           `db:"stopped_at"`
}

type PreviewSessionRole struct {
	ID           uuid.UUID         `db:"id"`
	SessionID    uuid.UUID         `db:"session_id"`
	WorkerID     uuid.UUID         `db:"worker_id"`
	Role         string            `db:"role"`
	Port         *int              `db:"port"`
	Status       PreviewRoleStatus `db:"status"`
	ErrorMessage *string           `db:"error_message"`
	PreviewURL   *string           `db:"preview_url"`
	LogOutput    string            `db:"log_output"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
}
