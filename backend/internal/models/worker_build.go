package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkerBuildMode string

const (
	WorkerBuildModeFresh  WorkerBuildMode = "fresh"
	WorkerBuildModeLatest WorkerBuildMode = "latest"
)

func IsWorkerBuildMode(value string) bool {
	switch WorkerBuildMode(value) {
	case WorkerBuildModeFresh, WorkerBuildModeLatest:
		return true
	default:
		return false
	}
}

type WorkerBuildStatus string

const (
	WorkerBuildStatusQueued   WorkerBuildStatus = "queued"
	WorkerBuildStatusBuilding WorkerBuildStatus = "building"
	WorkerBuildStatusReady    WorkerBuildStatus = "ready"
	WorkerBuildStatusFailed   WorkerBuildStatus = "failed"
)

func IsWorkerBuildStatus(value string) bool {
	switch WorkerBuildStatus(value) {
	case WorkerBuildStatusQueued,
		WorkerBuildStatusBuilding,
		WorkerBuildStatusReady,
		WorkerBuildStatusFailed:
		return true
	default:
		return false
	}
}

type WorkerBuild struct {
	ID             uuid.UUID         `db:"id"`
	WorkerID       uuid.UUID         `db:"worker_id"`
	StackID        string            `db:"stack_id"`
	Role           string            `db:"role"`
	BuildMode      WorkerBuildMode   `db:"build_mode"`
	Status         WorkerBuildStatus `db:"status"`
	CallbackToken  uuid.UUID         `db:"callback_token"`
	ImageReference *string           `db:"image_reference"`
	ImageDigest    *string           `db:"image_digest"`
	ErrorMessage   *string           `db:"error_message"`
	BuildLog       *string           `db:"build_log"`
	TriggeredBy    *uuid.UUID        `db:"triggered_by"`
	StartedAt      *time.Time        `db:"started_at"`
	CompletedAt    *time.Time        `db:"completed_at"`
	CreatedAt      time.Time         `db:"created_at"`
}
