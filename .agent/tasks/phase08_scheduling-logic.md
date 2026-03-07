# Task: OAgent Scheduling Logic

## Phase
phase08

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- backend/internal/services/scheduler_service.go
- backend/internal/services/dispatcher_service.go
- backend/internal/services/job_service.go (HandleWorkerReply, SendUserMessage)

## Notes
- TryAssignJob: checks manual_worker_override first, then LRU GetLRUIdleByGroup
- Worker→Job status mapping: busy→busy (interim answer), pending_user→pending_user (question), idle→done (final), offline→failed
- ProcessQueue called after idle worker reply to pick up next queued job
- SendUserMessage: INSERT user message, UPDATE job to busy, call NotifyUserReply, broadcast WS
- Dispatcher POSTs DispatchRequest to worker.callback_url with action new/continue/cancel
- CancelWorkerJob is fire-and-forget (errors logged, not fatal)
