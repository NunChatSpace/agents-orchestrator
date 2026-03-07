package models

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	JobStatusDraft       JobStatus = "draft"
	JobStatusQueued      JobStatus = "queued"
	JobStatusAssigned    JobStatus = "assigned"
	JobStatusBusy        JobStatus = "busy"
	JobStatusPendingUser JobStatus = "pending_user"
	JobStatusDone        JobStatus = "done"
	JobStatusFailed      JobStatus = "failed"
	JobStatusCancelled   JobStatus = "cancelled"
)

type Job struct {
	JobID                uuid.UUID  `db:"job_id"`
	UserID               uuid.UUID  `db:"user_id"`
	Title                string     `db:"title"`
	Prompt               string     `db:"prompt"`
	TargetGroup          string     `db:"target_group"`
	AssignedWorkerID     *uuid.UUID `db:"assigned_worker_id"`
	ManualWorkerOverride *uuid.UUID `db:"manual_worker_override"`
	Status               JobStatus  `db:"status"`
	ResumeID             *string    `db:"resume_id"`
	FailureReason        *string    `db:"failure_reason"`
	CreatedAt            time.Time  `db:"created_at"`
	UpdatedAt            time.Time  `db:"updated_at"`
	DeletedAt            *time.Time `db:"deleted_at"`
}
