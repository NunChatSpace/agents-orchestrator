package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkerStatus string

const (
	WorkerStatusBusy        WorkerStatus = "busy"
	WorkerStatusPendingUser WorkerStatus = "pending_user"
	WorkerStatusIdle        WorkerStatus = "idle"
	WorkerStatusOffline     WorkerStatus = "offline"
)

type Worker struct {
	WorkerID     uuid.UUID    `db:"worker_id"`
	GroupID      uuid.UUID    `db:"group_id"`
	// GroupName is populated by queries that JOIN with worker_groups.
	GroupName    string       `db:"group_name"`
	Name         string       `db:"name"`
	CallbackURL  string       `db:"callback_url"`
	APIKeyHash   string       `db:"api_key_hash"`
	Workspace    string       `db:"workspace"`
	CLICommand   string       `db:"cli_command"`
	GitRepoURL   string       `db:"git_repo_url"`
	Status       WorkerStatus `db:"status"`
	LastActiveAt *time.Time   `db:"last_active_at"`
	CreatedAt    time.Time    `db:"created_at"`
	DeletedAt    *time.Time   `db:"deleted_at"`
}
