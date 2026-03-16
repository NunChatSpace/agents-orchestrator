# Phase B — PreviewSession: Migration + Model + Repository

## Goal
Add the database schema and Go data layer for the new process-based preview system.

---

## STEP 1 — Migration: `backend/migrations/012_preview_sessions.sql`

```sql
-- Add preview_command to workers (shell command the agent runs to start the preview server)
ALTER TABLE workers ADD COLUMN IF NOT EXISTS preview_command TEXT;

-- One preview session groups one or more agents previewing together
CREATE TABLE preview_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID NOT NULL REFERENCES users(id),
    status       TEXT NOT NULL DEFAULT 'starting',  -- starting | running | stopped | failed
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at   TIMESTAMPTZ
);

CREATE INDEX idx_preview_sessions_created_by ON preview_sessions(created_by);
CREATE INDEX idx_preview_sessions_status     ON preview_sessions(status);

-- One row per agent role within a session
CREATE TABLE preview_session_roles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES preview_sessions(id) ON DELETE CASCADE,
    worker_id     UUID NOT NULL REFERENCES workers(worker_id),
    role          TEXT NOT NULL,        -- e.g. 'frontend' | 'backend' | 'app'
    port          INT,                  -- dynamically allocated from 8300+
    status        TEXT NOT NULL DEFAULT 'starting',  -- starting | running | stopped | failed
    error_message TEXT,
    preview_url   TEXT,                 -- http://shiphide.{worker_name}.preview
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_preview_session_roles_session_id ON preview_session_roles(session_id);
CREATE INDEX idx_preview_session_roles_worker_id  ON preview_session_roles(worker_id);
```

---

## STEP 2 — Model: `backend/internal/models/preview_session.go`

```go
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
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
}
```

Also add `PreviewCommand *string \`db:"preview_command"\`` to the existing `Worker` struct in `backend/internal/models/worker.go`.

---

## STEP 3 — Repository: `backend/internal/repository/preview_session_repo.go`

Define and implement the interface:

```go
type PreviewSessionRepository interface {
	Create(ctx context.Context, session *models.PreviewSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.PreviewSession, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PreviewSession, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.PreviewSessionStatus, errMsg *string, stoppedAt *time.Time) error

	CreateRole(ctx context.Context, role *models.PreviewSessionRole) error
	GetRolesBySession(ctx context.Context, sessionID uuid.UUID) ([]*models.PreviewSessionRole, error)
	GetRoleByWorkerAndSession(ctx context.Context, workerID uuid.UUID, sessionID uuid.UUID) (*models.PreviewSessionRole, error)
	UpdateRoleStatus(ctx context.Context, id uuid.UUID, status models.PreviewRoleStatus, port *int, previewURL *string, errMsg *string) error
	AllocatePort(ctx context.Context) (int, error)
}
```

`AllocatePort` implementation:
```go
// SELECT all non-null ports, find lowest port >= 8300 not already taken
func (r *previewSessionRepo) AllocatePort(ctx context.Context) (int, error) {
    var ports []int
    err := r.db.SelectContext(ctx, &ports,
        `SELECT port FROM preview_session_roles WHERE port IS NOT NULL`)
    if err != nil {
        return 0, err
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
```

Also update `backend/internal/repository/worker_repo.go`:
- Add `preview_command` to SELECT columns
- Add `preview_command` to UPDATE SET clause when present

Follow existing patterns in the repo (sqlx, named queries).

---

## Acceptance Criteria

1. `go build ./...` from `backend/` compiles without errors.
2. Migration SQL is valid (no syntax errors).
3. `PreviewSession` and `PreviewSessionRole` models match the migration schema exactly.
4. `AllocatePort` returns 8300 when no ports are in use.
5. Worker model has `PreviewCommand *string \`db:"preview_command"\``.
