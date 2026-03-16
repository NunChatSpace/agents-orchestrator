# Phase D — PreviewSession Controller + Routes + DI Wiring

## Goal
Wire the new preview system into HTTP routes and the DI container. Also allow `preview_command` to be updated via the worker PATCH endpoint.

---

## STEP 1 — Controller: `backend/internal/controllers/preview_session_controller.go`

Read an existing controller (e.g. `job_controller.go` or `worker_controller.go`) for the exact pattern used in this project. Follow it exactly.

```go
type PreviewSessionController struct {
	Service services.PreviewSessionService
}

func NewPreviewSessionController(svc services.PreviewSessionService) *PreviewSessionController {
	return &PreviewSessionController{Service: svc}
}
```

### Handlers

**Create** — `POST /api/v1/preview-sessions` (session auth)
- Parse `CreatePreviewSessionRequest` from body.
- Validate `len(req.Roles) >= 1`; return 400 if empty.
- Call `service.Create(ctx, userID, req)`.
- On success: fetch roles, return `PreviewSessionResponse` with 201.

**List** — `GET /api/v1/preview-sessions` (session auth)
- Call `service.List(ctx, userID)`.
- For each session, fetch roles.
- Return `[]PreviewSessionResponse` 200.

**Get** — `GET /api/v1/preview-sessions/{id}` (session auth)
- Parse `{id}` as UUID.
- Call `service.Get(ctx, id)`.
- Return `PreviewSessionResponse` 200.

**Stop** — `POST /api/v1/preview-sessions/{id}/stop` (session auth)
- Parse `{id}` as UUID.
- Call `service.Stop(ctx, id)`.
- Return 200 `{ "ok": true }`.

**Delete** — `DELETE /api/v1/preview-sessions/{id}` (session auth)
- Parse `{id}` as UUID.
- Call `service.Delete(ctx, id)`.
- Return 204 on success; 400 if session is not stopped/failed.

**ReportRole** — `POST /api/v1/workers/{worker_id}/preview-reports` (worker-key auth)
- Parse `{worker_id}` as UUID.
- Parse `PreviewReportRequest` from body.
- Validate `req.Status` is `"running"` or `"failed"`.
- Call `service.ReportRole(ctx, workerID, req)`.
- Return 200 `{ "ok": true }`.

Helper to build `PreviewSessionResponse`:
```go
func toPreviewSessionResponse(session *models.PreviewSession, roles []*models.PreviewSessionRole) domains.PreviewSessionResponse {
    roleResponses := make([]domains.PreviewSessionRoleResponse, len(roles))
    for i, r := range roles {
        roleResponses[i] = domains.PreviewSessionRoleResponse{
            ID:         r.ID.String(),
            WorkerID:   r.WorkerID.String(),
            Role:       r.Role,
            Port:       r.Port,
            Status:     string(r.Status),
            PreviewURL: r.PreviewURL,
            Error:      r.ErrorMessage,
        }
    }
    return domains.PreviewSessionResponse{
        ID:        session.ID.String(),
        Status:    string(session.Status),
        Error:     session.Error,
        Roles:     roleResponses,
        CreatedAt: session.CreatedAt,
        StoppedAt: session.StoppedAt,
    }
}
```

---

## STEP 2 — Routes: `backend/internal/app/routes.go`

Read the current routes.go. Add under **session-protected** routes:

```go
previewCtrl := controllers.NewPreviewSessionController(container.PreviewSessionService)
v1.HandleFunc("/preview-sessions",            previewCtrl.Create).Methods("POST")
v1.HandleFunc("/preview-sessions",            previewCtrl.List).Methods("GET")
v1.HandleFunc("/preview-sessions/{id}",       previewCtrl.Get).Methods("GET")
v1.HandleFunc("/preview-sessions/{id}/stop",  previewCtrl.Stop).Methods("POST")
v1.HandleFunc("/preview-sessions/{id}",       previewCtrl.Delete).Methods("DELETE")
```

Add under **worker-key-protected** routes (same auth middleware group as existing worker-authenticated endpoints):

```go
v1Worker.HandleFunc("/workers/{worker_id}/preview-reports", previewCtrl.ReportRole).Methods("POST")
```

---

## STEP 3 — DI Container: `backend/internal/app/container.go`

Read the current container.go. Add:

```go
// Repository
PreviewSessionRepo := repository.NewPreviewSessionRepository(db)

// NginxConfigService (needs session repo + worker repo for name lookup)
NginxConfigSvc := services.NewNginxConfigService(PreviewSessionRepo, WorkerRepo)

// PreviewSessionService
PreviewSessionSvc := services.NewPreviewSessionService(
    PreviewSessionRepo,
    WorkerRepo,
    DispatcherSvc,   // existing dispatcher
    NginxConfigSvc,
)

// Controller
PreviewSessionCtrl := controllers.NewPreviewSessionController(PreviewSessionSvc)
```

Expose `PreviewSessionService` and `PreviewSessionCtrl` on the container struct so routes.go can access them.

---

## STEP 4 — Worker PATCH: allow `preview_command` update

### `backend/internal/controllers/worker_controller.go`

Read current PATCH handler. In the update request struct (or inline parse), add:
```go
PreviewCommand *string `json:"preview_command"`
```
If present and non-nil, include it in the update call.

### `backend/internal/repository/worker_repo.go`

In the UPDATE query for workers, add `preview_command = :preview_command` to the SET clause (only if the field is being updated — follow existing pattern for partial updates).

In SELECT queries that return full worker records, add `preview_command` to the column list.

---

## Acceptance Criteria

1. `go build ./...` compiles without errors.
2. `POST /api/v1/preview-sessions` returns 201 with a session and roles.
3. `POST /api/v1/workers/{id}/preview-reports` is reachable with worker-key auth.
4. `DELETE /api/v1/preview-sessions/{id}` returns 400 if session is still running.
5. `PATCH /api/v1/workers/{id}` with `{ "preview_command": "PORT={PORT} npm run dev" }` persists the value.
