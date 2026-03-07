# Task: Backend Models, Repositories, Services

## Phase
phase03

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- backend/internal/models/user.go
- backend/internal/models/session.go
- backend/internal/models/worker_group.go
- backend/internal/models/worker.go
- backend/internal/models/job.go
- backend/internal/models/message.go
- backend/internal/repository/interfaces.go
- backend/internal/repository/user_repo.go
- backend/internal/repository/session_repo.go
- backend/internal/repository/worker_repo.go
- backend/internal/repository/job_repo.go
- backend/internal/repository/message_repo.go
- backend/internal/services/auth_service.go
- backend/internal/services/job_service.go
- backend/internal/services/scheduler_service.go
- backend/internal/services/dispatcher_service.go

## Notes
- DequeueOldest uses FOR UPDATE SKIP LOCKED to prevent concurrent scheduler races
- GetLRUIdleByGroup orders by last_active_at ASC NULLS FIRST FOR UPDATE SKIP LOCKED
- WSBroadcaster interface defined in services package to avoid circular imports
- HandleWorkerReply maps: busy→busy (interim), pending_user→pending_user, idle→done, offline→failed
- truncate(prompt, 60) used as auto-title fallback
- ValidateSession calls userRepo.GetByID(ctx, session.UserID) — fixed from initial GetByUsername bug
