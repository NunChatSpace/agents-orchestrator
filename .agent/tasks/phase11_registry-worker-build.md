# Phase 11 — Local OCI Registry + Worker Build Trigger

## Goal

Add a local Docker registry to the compose stack and wire the backend to receive per-agent image build triggers and build report callbacks.

After this phase:
- A local OCI registry runs in the compose stack.
- The backend can receive a "trigger build" request for a specific worker.
- Workers can report their build result (success/failure + image reference + digest).
- Build history per worker is queryable via API.

No frontend changes. No deployment plan logic yet.

---

## Spec Reference

`.agent/specs/workspace-preview-runtime-spec-v1.md` §7.1, §8.1, §9.1, §9.2, §11

---

## Affected Files

| File | Change |
|---|---|
| `docker-compose.yml` | Add `registry` service |
| `backend/migrations/012_worker_builds.sql` | New `worker_builds` table |
| `backend/internal/models/worker_build.go` | New model + status constants |
| `backend/internal/repository/worker_build_repo.go` | New repository interface + implementation |
| `backend/internal/services/worker_build_service.go` | New service interface + implementation |
| `backend/internal/domains/worker_build.go` | Request/Response DTOs |
| `backend/internal/controllers/worker_build_controller.go` | New controller |
| `backend/internal/app/routes.go` | Register new routes |
| `backend/main.go` | Wire new dependencies into DI container |
| `.agent/ARCHITECTURE.md` | Document new endpoints and registry service |

---

## docker-compose.yml Changes

Add `registry` service:

```yaml
registry:
  image: registry:2
  ports:
    - "5001:5000"
  volumes:
    - registrydata:/var/lib/registry

volumes:
  pgdata:
  registrydata:
```

Workers inside Docker reach the registry at `registry:5000`.
The host (and any tool running on host) reaches it at `localhost:5001`.

---

## Migration: `backend/migrations/012_worker_builds.sql`

```sql
CREATE TABLE worker_builds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id       UUID NOT NULL REFERENCES workers(worker_id),
    stack_id        TEXT NOT NULL,
    role            TEXT NOT NULL,
    build_mode      TEXT NOT NULL DEFAULT 'fresh',    -- 'fresh' | 'latest'
    status          TEXT NOT NULL DEFAULT 'queued',   -- 'queued' | 'building' | 'ready' | 'failed'
    image_reference TEXT,
    image_digest    TEXT,
    error_message   TEXT,
    triggered_by    UUID REFERENCES users(id),        -- NULL = system-triggered
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_worker_builds_worker_id ON worker_builds(worker_id);
CREATE INDEX idx_worker_builds_status    ON worker_builds(status);
```

---

## Model: `backend/internal/models/worker_build.go`

```go
type WorkerBuildStatus string

const (
    WorkerBuildStatusQueued   WorkerBuildStatus = "queued"
    WorkerBuildStatusBuilding WorkerBuildStatus = "building"
    WorkerBuildStatusReady    WorkerBuildStatus = "ready"
    WorkerBuildStatusFailed   WorkerBuildStatus = "failed"
)

type WorkerBuild struct {
    ID             uuid.UUID         `db:"id"`
    WorkerID       uuid.UUID         `db:"worker_id"`
    StackID        string            `db:"stack_id"`
    Role           string            `db:"role"`
    BuildMode      string            `db:"build_mode"`
    Status         WorkerBuildStatus `db:"status"`
    ImageReference *string           `db:"image_reference"`
    ImageDigest    *string           `db:"image_digest"`
    ErrorMessage   *string           `db:"error_message"`
    TriggeredBy    *uuid.UUID        `db:"triggered_by"`
    StartedAt      *time.Time        `db:"started_at"`
    CompletedAt    *time.Time        `db:"completed_at"`
    CreatedAt      time.Time         `db:"created_at"`
}
```

---

## Repository: `backend/internal/repository/worker_build_repo.go`

Interface:

```go
type WorkerBuildRepository interface {
    Create(ctx context.Context, build *models.WorkerBuild) error
    GetByID(ctx context.Context, id uuid.UUID) (*models.WorkerBuild, error)
    ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status models.WorkerBuildStatus, imageRef, digest, errMsg *string, completedAt *time.Time) error
    LatestReadyForWorker(ctx context.Context, workerID uuid.UUID) (*models.WorkerBuild, error)
}
```

---

## Service: `backend/internal/services/worker_build_service.go`

Interface:

```go
type WorkerBuildService interface {
    TriggerBuild(ctx context.Context, workerID uuid.UUID, stackID, role, mode string, triggeredBy uuid.UUID) (*models.WorkerBuild, error)
    ReportBuild(ctx context.Context, workerID uuid.UUID, req domains.WorkerBuildReportRequest) (*models.WorkerBuild, error)
    ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error)
}
```

`TriggerBuild` rules:
1. Validate `mode` is `"fresh"` or `"latest"`.
2. If `mode == "latest"`: check for an existing `ready` build for this worker. If found, return it directly (no new record, no dispatch).
3. If `mode == "fresh"` or no ready build exists: create a new `WorkerBuild` with `status=queued`.
4. Dispatch build instruction to the worker using `DispatcherService` (creates a synthetic build job or sends a message to the active job; implementation detail for the dispatcher).
5. Return the new `WorkerBuild` record.

`ReportBuild` rules:
1. Look up the `WorkerBuild` by `worker_build_id` from the request.
2. Validate the reporting worker owns that build record.
3. If `status == "ready"`: require non-empty `image_reference` and `image_digest`. Update build to `ready`.
4. If `status == "failed"`: store `error_message`. Update build to `failed`.
5. Return updated record.

---

## Domains: `backend/internal/domains/worker_build.go`

```go
type TriggerWorkerBuildRequest struct {
    StackID string `json:"stack_id"`
    Role    string `json:"role"`
    Mode    string `json:"mode"` // "fresh" | "latest"
}

type WorkerBuildReportRequest struct {
    WorkerBuildID  string            `json:"worker_build_id"`
    Status         string            `json:"status"`           // "ready" | "failed"
    ImageReference string            `json:"image_reference,omitempty"`
    ImageDigest    string            `json:"image_digest,omitempty"`
    Metadata       map[string]string `json:"metadata,omitempty"`
    ErrorMessage   string            `json:"error_message,omitempty"`
}

type WorkerBuildResponse struct {
    ID             string     `json:"id"`
    WorkerID       string     `json:"worker_id"`
    StackID        string     `json:"stack_id"`
    Role           string     `json:"role"`
    BuildMode      string     `json:"build_mode"`
    Status         string     `json:"status"`
    ImageReference *string    `json:"image_reference,omitempty"`
    ImageDigest    *string    `json:"image_digest,omitempty"`
    ErrorMessage   *string    `json:"error_message,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

---

## Controller: `backend/internal/controllers/worker_build_controller.go`

```go
type WorkerBuildController struct {
    Service services.WorkerBuildService
}
```

Handlers:

- `TriggerBuild` — `POST /api/v1/workers/{worker_id}/builds`
  - Session-authenticated
  - Parse `TriggerWorkerBuildRequest`
  - Validate `stack_id`, `role`, `mode`
  - Call `service.TriggerBuild`
  - Return `WorkerBuildResponse` 201

- `List` — `GET /api/v1/workers/{worker_id}/builds`
  - Session-authenticated
  - Call `service.ListByWorker`
  - Return `[]WorkerBuildResponse` 200

- `ReportBuild` — `POST /api/v1/workers/{worker_id}/build-reports`
  - Worker-key-authenticated (same middleware as existing `build-reports` endpoint)
  - Parse `WorkerBuildReportRequest`
  - Call `service.ReportBuild`
  - Return updated `WorkerBuildResponse` 200

---

## Routes: `backend/internal/app/routes.go`

Add under session-protected routes:

```go
workerBuildCtrl := controllers.NewWorkerBuildController(workerBuildService)
v1.HandleFunc("/workers/{worker_id}/builds",        workerBuildCtrl.TriggerBuild).Methods("POST")
v1.HandleFunc("/workers/{worker_id}/builds",        workerBuildCtrl.List).Methods("GET")
```

Add under worker-key-protected routes:

```go
v1Worker.HandleFunc("/workers/{worker_id}/build-reports", workerBuildCtrl.ReportBuild).Methods("POST")
```

---

## Dispatch Mechanism (Assumption)

The `DispatcherService` sends the build instruction by creating a new job for the worker with a synthesized build instruction prompt. The instruction tells the worker to:

1. Run `docker build` from its workspace using the appropriate `Dockerfile`.
2. Tag the image as `registry:5000/{worker_name}/{role}:{YYYYMMDD-HHmmss}`.
3. Push to `registry:5000`.
4. Report back to `POST /api/v1/workers/{worker_id}/build-reports` with the image reference, digest, and the `worker_build_id`.

The exact instruction format is defined by the worker's `instruction_job` field or a build-specific override. This remains an open implementation point for phase 11.

---

## ARCHITECTURE.md Updates Required

- Add `registry` service to the Compose topology section.
- Add new API endpoints to the "API Endpoints — Workers" table.
- Document `WorkerBuild` model in the "Project-Specific Notes" section.

---

## Acceptance Criteria

1. `docker compose up` starts the registry container; `curl localhost:5001/v2/` returns `{}`.
2. `POST /api/v1/workers/{id}/builds` with `mode=fresh` creates a `worker_build` record with `status=queued`.
3. `POST /api/v1/workers/{id}/builds` with `mode=latest` when a `ready` build exists returns that existing build without creating a new record.
4. `GET /api/v1/workers/{id}/builds` returns build history ordered by `created_at DESC`.
5. `POST /api/v1/workers/{id}/build-reports` (worker-key auth) with `status=ready` updates the build to `ready` and stores `image_reference` and `image_digest`.
6. `POST /api/v1/workers/{id}/build-reports` with `status=failed` stores `error_message` and sets status to `failed`.
7. A worker cannot report a build that belongs to a different worker.
