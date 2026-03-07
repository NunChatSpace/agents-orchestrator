# Task: Backend Controllers & Routes

## Phase
phase04

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- backend/internal/domains/auth.go
- backend/internal/domains/job.go
- backend/internal/domains/message.go
- backend/internal/domains/worker.go
- backend/internal/views/response.go
- backend/internal/middleware/auth.go
- backend/internal/middleware/request_id.go
- backend/internal/middleware/worker_key.go
- backend/internal/middleware/logging.go
- backend/internal/middleware/cors.go
- backend/internal/middleware/recover.go
- backend/internal/controllers/auth_controller.go
- backend/internal/controllers/job_controller.go
- backend/internal/controllers/message_controller.go
- backend/internal/controllers/worker_controller.go
- backend/internal/app/container.go
- backend/internal/app/routes.go
- backend/internal/app/database.go
- backend/internal/app/seed.go

## Notes
- provideWSBroadcaster adapter bridges *ws.Hub → services.WSBroadcaster for dig DI
- reqID() in controllers reads X-Request-ID header injected by RequestID middleware
- Worker callback uses X-Worker-Key → SHA-256 → DB lookup (not session auth)
- Middleware stack: RequestID → Logger → Recover → CORS → route-level auth
