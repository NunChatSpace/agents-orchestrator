# Phase C — PreviewSession Service + NginxConfigService

## Goal
Implement the business logic layer: PreviewSessionService and the adapted NginxConfigService.

---

## STEP 1 — Domain DTOs: `backend/internal/domains/preview_session.go`

```go
package domains

import "time"

type PreviewRoleInput struct {
	Role     string `json:"role"`
	WorkerID string `json:"worker_id"`
}

type CreatePreviewSessionRequest struct {
	Roles []PreviewRoleInput `json:"roles"`
}

type PreviewSessionRoleResponse struct {
	ID         string  `json:"id"`
	WorkerID   string  `json:"worker_id"`
	Role       string  `json:"role"`
	Port       *int    `json:"port,omitempty"`
	Status     string  `json:"status"`
	PreviewURL *string `json:"preview_url,omitempty"`
	Error      *string `json:"error,omitempty"`
}

type PreviewSessionResponse struct {
	ID        string                       `json:"id"`
	Status    string                       `json:"status"`
	Error     *string                      `json:"error,omitempty"`
	Roles     []PreviewSessionRoleResponse `json:"roles"`
	CreatedAt time.Time                    `json:"created_at"`
	StoppedAt *time.Time                   `json:"stopped_at,omitempty"`
}

type PreviewReportRequest struct {
	SessionID    string `json:"session_id"`
	Status       string `json:"status"`         // "running" | "failed"
	Port         int    `json:"port,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
```

Also add to `backend/internal/domains/worker.go`:
```go
PreviewCommand *string `json:"preview_command,omitempty"`
```

---

## STEP 2 — Service: `backend/internal/services/preview_session_service.go`

### Interface

```go
type PreviewSessionService interface {
	Create(ctx context.Context, userID uuid.UUID, req domains.CreatePreviewSessionRequest) (*models.PreviewSession, error)
	Get(ctx context.Context, id uuid.UUID) (*models.PreviewSession, []*models.PreviewSessionRole, error)
	List(ctx context.Context, userID uuid.UUID) ([]*models.PreviewSession, error)
	Stop(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	ReportRole(ctx context.Context, workerID uuid.UUID, req domains.PreviewReportRequest) error
}
```

### Implementation struct

```go
type previewSessionService struct {
	repo       repository.PreviewSessionRepository
	workerRepo repository.WorkerRepository
	dispatcher DispatcherService
	nginx      NginxConfigService
}
```

### Create logic

1. Validate: `len(req.Roles) >= 1`; each `WorkerID` must be a valid UUID and worker must exist.
2. Create `PreviewSession` (status=starting).
3. Build a map of role→previewURL upfront for all roles: `http://shiphide.{worker.Name}.preview`.
4. For each role:
   a. `AllocatePort()` → port.
   b. Create `PreviewSessionRole` (status=starting, port, preview_url).
   c. Build env string:
      - Always: `PORT={port}`
      - If role == `"backend"` and a `"frontend"` role exists in session: append `\nCORS_ALLOWED_ORIGINS={frontend_preview_url}`
      - If role == `"frontend"` and a `"backend"` role exists in session: append `\nPUBLIC_API_BASE_URL={backend_preview_url}`
   d. Build instruction string:
      ```
      Start preview server.
      Run your preview_command with these environment variables set:
      {env_string}

      When the server is ready, call:
      POST /api/v1/workers/{worker_id}/preview-reports
      Body: {"session_id":"{session_id}","status":"running","port":{port}}

      If startup fails, call:
      POST /api/v1/workers/{worker_id}/preview-reports
      Body: {"session_id":"{session_id}","status":"failed","error_message":"<describe error>"}
      ```
   e. Call `dispatcher.SendToWorker(ctx, workerID, instruction)`.
5. Return session.

### Stop logic

1. Fetch session; return error if not found.
2. Fetch roles for session.
3. For each role with status `starting` or `running`:
   - Call `dispatcher.SendToWorker`: `"Stop the preview server running on port {port}. Kill the process gracefully."`
   - `UpdateRoleStatus(id, stopped, nil, nil, nil)`
4. `UpdateStatus(sessionID, stopped, nil, &now)`
5. `nginx.Reload(ctx)`

### Delete logic

1. Fetch session; return error if not found.
2. If status is not `stopped` or `failed`, return error: "session must be stopped or failed before deletion".
3. Hard delete session (cascade deletes roles).

### ReportRole logic

1. Parse `sessionID` UUID from `req.SessionID`.
2. `GetRoleByWorkerAndSession(workerID, sessionID)` → role.
3. Update port pointer (if provided) and error.
4. `UpdateRoleStatus(role.ID, status, &port, nil, &errMsg)`.
5. If `status == "running"`: call `nginx.Reload(ctx)`.
6. Re-fetch all roles for session:
   - All `running` → `UpdateStatus(sessionID, running, nil, nil)`
   - Any `failed` → `UpdateStatus(sessionID, failed, &errMsg, nil)`

---

## STEP 3 — NginxConfigService: adapt `backend/internal/services/nginx_config_service.go`

Read the existing file first. Rewrite it to implement this interface:

```go
type NginxConfigService interface {
	Reload(ctx context.Context) error
}
```

Implementation:

```go
type nginxConfigService struct {
	sessionRepo repository.PreviewSessionRepository
	workerRepo  repository.WorkerRepository
	confPath    string  // from env NGINX_PREVIEW_CONF, default "infra/nginx/preview-upstreams.conf"
}
```

`Reload(ctx)`:
1. Query all `preview_session_roles` where `status = 'running'` (add a helper method `ListRunningRoles(ctx) ([]*models.PreviewSessionRole, error)` to PreviewSessionRepository).
2. For each running role, look up the worker by `worker_id` to get `worker.Name`.
3. Build nginx config string — one upstream + server block per role:
   ```nginx
   upstream preview_{worker_name} {
       server host.docker.internal:{port};
   }
   server {
       listen 80;
       server_name shiphide.{worker_name}.preview;
       location / {
           proxy_pass http://preview_{worker_name};
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
       }
   }
   ```
4. Prepend header comment: `# Generated by NginxConfigService — do not edit manually\n`.
5. Write to `confPath` (create or overwrite).
6. Run: `docker exec proxy nginx -s reload` via `exec.CommandContext`.
7. Return any error from write or exec.

Add `ListRunningRoles` to the `PreviewSessionRepository` interface and implement it:
```go
ListRunningRoles(ctx context.Context) ([]*models.PreviewSessionRole, error)
// SELECT * FROM preview_session_roles WHERE status = 'running'
```

---

## Acceptance Criteria

1. `go build ./...` compiles without errors.
2. `Create` with 2 roles (frontend + backend) correctly injects `CORS_ALLOWED_ORIGINS` and `PUBLIC_API_BASE_URL`.
3. `Create` with 1 role (solo) does not inject CORS env vars.
4. `ReportRole` with `status=running` triggers `nginx.Reload`.
5. `NginxConfigService.Reload` generates one upstream+server block per running role.
6. `Stop` sets session status to `stopped` and calls `nginx.Reload`.
