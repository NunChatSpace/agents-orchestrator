package models

import (
	"time"

	"github.com/google/uuid"
)

type PlanSessionStatus string

const (
	PlanSessionStatusPending   PlanSessionStatus = "pending"
	PlanSessionStatusCompleted PlanSessionStatus = "completed"
	PlanSessionStatusDiscarded PlanSessionStatus = "discarded"
)

type PlanSession struct {
	ID              uuid.UUID         `db:"id"`
	UserID          uuid.UUID         `db:"user_id"`
	WorkerID        uuid.UUID         `db:"worker_id"`
	Title           string            `db:"title"`
	Status          PlanSessionStatus `db:"status"`
	ResumeID        *string           `db:"resume_id"`
	GeneratedPrompt *string           `db:"generated_prompt"`
	CreatedAt       time.Time         `db:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at"`
	DeletedAt       *time.Time        `db:"deleted_at"`
}

type PlanMessage struct {
	ID            uuid.UUID `db:"id"`
	PlanSessionID uuid.UUID `db:"plan_session_id"`
	Role          string    `db:"role"` // user | agent
	Content       string    `db:"content"`
	CreatedAt     time.Time `db:"created_at"`
}
