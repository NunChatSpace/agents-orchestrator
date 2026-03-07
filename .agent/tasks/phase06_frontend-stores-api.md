# Task: Frontend Types, Stores, API Layer

## Phase
phase06

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- frontend/src/types/job.ts
- frontend/src/types/message.ts
- frontend/src/types/worker.ts
- frontend/src/types/api.ts
- frontend/src/lib/apis/client.ts
- frontend/src/lib/apis/auth.ts
- frontend/src/lib/apis/jobs.ts
- frontend/src/lib/apis/messages.ts
- frontend/src/lib/apis/workers.ts
- frontend/src/stores/auth.ts
- frontend/src/stores/jobs.ts
- frontend/src/stores/activeJob.ts
- frontend/src/stores/workers.ts
- frontend/src/stores/ws.ts

## Notes
- client.ts unwraps body.data automatically — all API functions return T directly (not { data: T })
- ws.ts auto-reconnects after 3s on close; dispatches job_updated/message_added/worker_updated to stores
- composerEnabled derived: job.status === 'pending_user'
- filteredJobs derived: sorted by updated_at DESC, filterable by status and target_group
