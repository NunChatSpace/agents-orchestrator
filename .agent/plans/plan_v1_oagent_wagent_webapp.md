# Plan: OAgent/WAgent Job-Oriented Web App (v1)

## Context

Build a personal-use web app to orchestrate AI worker agents via a central orchestrator (OAgent).
Users create jobs, assign them to a worker group, and continue conversations in the same job via `resume_id`.
OAgent manages the job queue and dispatches to WAgents (HTTP services that internally run CLI shell commands).

Spec source: `.agent/specs/spec_v1_oagent_wagent_webapp.md`
Architecture source: `.agent/ARCHITECTURE.md`

Key decisions confirmed with user:
- WAgents = HTTP services that shell out to CLI agents internally
- Real-time UI = WebSockets
- Job creation = draft first, then explicit submit to queue
- Auth = single hardcoded user seeded from env vars at startup

---

## Task Tracking Convention

When a phase is complete, create a task file at:
`.agent/tasks/{phase_id}_{short-slug}.md`

Example: `.agent/tasks/phase01_project-bootstrap.md`

---

## Project Structure to Create

```
agent-orchestrator/
├── docker-compose.yml
├── .env.example
├── backend/
│   ├── go.mod
│   ├── main.go
│   ├── migrations/
│   │   ├── 001_initial_schema.sql
│   │   └── 002_seed_workers.sql
│   └── internal/
│       ├── models/          # DB entities
│       ├── repository/      # Data access interfaces + implementations
│       ├── services/        # Business logic
│       ├── domains/         # Request/Response DTOs
│       ├── controllers/     # HTTP handlers (thin)
│       ├── views/           # JSON envelope helpers
│       ├── ws/              # WebSocket hub
│       ├── middleware/      # auth, worker-key, logging, CORS, recover
│       └── app/             # DI container, server, routes wiring
└── frontend/
    ├── package.json
    ├── svelte.config.js
    ├── tailwind.config.js
    └── src/
        ├── routes/
        ├── components/
        ├── lib/apis/
        ├── stores/
        └── types/
```

---

## Phase 1: Project Bootstrap

**Task file:** `.agent/tasks/phase01_project-bootstrap.md`

**Files to create:**
- `docker-compose.yml` — postgres + backend + frontend services
- `.env.example` — DATABASE_URL, SESSION_SECRET, WORKER_API_KEY, SEED_USERNAME, SEED_PASSWORD, PORT
- `backend/go.mod` — Go 1.23, deps: gorilla/mux, gorilla/websocket, sqlx, lib/pq, uber/dig, google/uuid, bcrypt, zerolog, goose
- `backend/main.go` — skeleton: load env, run migrations, seed user, build DI container, start HTTP+WS server
- `frontend/` — SvelteKit skeleton with TypeScript + Tailwind CSS

---

## Phase 2: Database Migrations

**Task file:** `.agent/tasks/phase02_database-migrations.md`

**File: `backend/migrations/001_initial_schema.sql`**

Tables:
- `users` — user_id (UUID PK), username, password_hash, created_at, deleted_at
- `sessions` — session_id (TEXT PK), user_id (FK), created_at, expires_at, deleted_at
- `worker_groups` — group_id (UUID PK), name (UNIQUE: fi-backend / fi-frontend / ib-kha), created_at
- `workers` — worker_id (UUID PK), group_id (FK), name (UNIQUE), callback_url, api_key_hash, status CHECK(busy/pending_user/idle/offline), last_active_at (LRU tracking), created_at, deleted_at
- `jobs` — job_id (UUID PK), user_id (FK), title, prompt, target_group, assigned_worker_id (FK nullable), manual_worker_override (FK nullable), status CHECK(draft/queued/assigned/busy/pending_user/done/failed/cancelled), resume_id (TEXT nullable), created_at, updated_at, deleted_at
- `messages` — message_id (UUID PK), job_id (FK), role CHECK(user/oagent/worker), kind CHECK(instruction/question/answer/summary/system), content, created_at

Key indexes: jobs(status), jobs(target_group), jobs(updated_at DESC), workers(status), workers(last_active_at), messages(job_id)

Trigger: `set_updated_at()` on jobs BEFORE UPDATE.

**File: `backend/migrations/002_seed_workers.sql`**

Insert worker_groups (fi-backend, fi-frontend, ib-kha) and 5 workers with placeholder callback_url/api_key_hash.

Note: `job.status = 'pending_user'` is included in the job status enum so the UI can check it directly.

---

## Phase 3: Backend — Models, Repositories, Services

**Task file:** `.agent/tasks/phase03_backend-models-repos-services.md`

### Models (`internal/models/`)
- `user.go`, `session.go`, `worker_group.go`, `worker.go`, `job.go`, `message.go`
- Define typed constants: `JobStatus`, `WorkerStatus`, `MessageRole`, `MessageKind`

### Repository interfaces (`internal/repository/interfaces.go`)

```
JobRepository:
  Create, GetByID, List(userID, filters), UpdateStatus, UpdateAssignment,
  UpdateResumeID, SoftDelete, DequeueOldest(targetGroup) -- FOR UPDATE SKIP LOCKED

WorkerRepository:
  GetByID, GetLRUIdleByGroup(groupName) -- ORDER BY last_active_at ASC NULLS FIRST FOR UPDATE SKIP LOCKED
  UpdateStatus, UpdateLastActiveAt, GetByAPIKeyHash, ListAll

MessageRepository:
  Create, ListByJobID

SessionRepository:
  Create, GetByID, Delete, DeleteExpired

UserRepository:
  GetByUsername, Create
```

### Services (`internal/services/`)
- `auth_service.go` — Login (bcrypt check, create session), Logout, ValidateSession
- `job_service.go` — CreateJob(draft), SubmitJob(draft→queued), GetJob, ListJobs, CancelJob, CloseJob, SendUserMessage
- `scheduler_service.go` — TryAssignJob(jobID), ProcessQueue(targetGroup)
- `dispatcher_service.go` — SendToWorker(worker, job), NotifyUserReply(worker, job, message), CancelWorkerJob(worker, jobID)
- `worker_service.go` — HandleWorkerReply(req), ListWorkers

---

## Phase 4: Backend — Controllers & Routes

**Task file:** `.agent/tasks/phase04_backend-controllers-routes.md`

### Full API surface (base: `/api/v1`)

**Auth** (no session required):
- `POST /auth/login` → `{ username, password }` → Set-Cookie + user data
- `POST /auth/logout` → Clear-Cookie
- `GET /auth/me` → current user

**Jobs** (session required):
- `GET /jobs` → query: status, target_group, limit, offset
- `POST /jobs` → `{ title, prompt, target_group, manual_worker_override? }` → status=draft
- `GET /jobs/{job_id}`
- `PATCH /jobs/{job_id}` → edit draft fields (only when status=draft)
- `POST /jobs/{job_id}/submit` → draft → queued, triggers scheduler
- `POST /jobs/{job_id}/cancel` → assigned/busy/pending_user → cancelled
- `DELETE /jobs/{job_id}` → soft delete (only done/failed/cancelled)

**Messages** (session required):
- `GET /jobs/{job_id}/messages`
- `POST /jobs/{job_id}/messages` → `{ content }` (only when status=pending_user)

**Workers** (session required):
- `GET /workers` → list all workers with status

**Worker callback** (X-Worker-Key auth, NOT session):
- `POST /workers/reply` → `{ job_id, worker_id, status, message, resume_id?, updated_at }`

**WebSocket** (session required):
- `GET /ws` → upgrades to WebSocket; client subscribes to job updates

### Domains (`internal/domains/`)
- `auth.go` — LoginRequest, UserResponse
- `job.go` — CreateJobRequest, PatchJobRequest, JobResponse (includes message_count, assigned_worker_name)
- `message.go` — SendMessageRequest, MessageResponse
- `worker.go` — WorkerReplyRequest, WorkerResponse, DispatchRequest

### Views (`internal/views/response.go`)
- `JSON(w, status, data, requestID)` — wraps in `{ data, meta: { request_id } }`
- `Error(w, status, code, message, requestID)`

### Middleware (`internal/middleware/`)
- `auth.go` — reads session cookie, validates, injects user into context
- `worker_key.go` — reads X-Worker-Key header, SHA-256 hash lookup
- `logging.go` — zerolog request logger
- `cors.go` — allow frontend origin, credentials: true
- `recover.go` — panic → 500

---

## Phase 5: WebSocket Hub

**Task file:** `.agent/tasks/phase05_websocket-hub.md`

**File: `internal/ws/hub.go`**

```
Hub:
  clients map[userID][]Conn
  broadcast(userID, event)
  register(conn)
  unregister(conn)

Events pushed to frontend:
  { type: "job_updated", data: JobResponse }
  { type: "message_added", data: MessageResponse }
  { type: "worker_updated", data: WorkerResponse }
```

Services call `hub.Broadcast(userID, event)` after state changes.

**File: `internal/controllers/ws_controller.go`**
- Upgrades HTTP to WebSocket using gorilla/websocket
- Registers connection with hub
- Reads pings (keep-alive); writes events from hub channel

---

## Phase 6: Frontend — Types, Stores, API Layer

**Task file:** `.agent/tasks/phase06_frontend-types-stores-api.md`

### Types (`src/types/`)
- `job.ts` — Job, JobStatus, TargetGroup, CreateJobPayload, PatchJobPayload
- `message.ts` — Message, MessageRole, MessageKind
- `worker.ts` — Worker, WorkerStatus
- `api.ts` — ApiResponse\<T\>, ApiErrorResponse

### API Layer (`src/lib/apis/`)
- `client.ts` — base fetch wrapper, credentials: include, throws ApiError on non-ok
- `auth.ts` — login, logout, getMe
- `jobs.ts` — listJobs, createJob, getJob, patchJob, submitJob, cancelJob, deleteJob
- `messages.ts` — listMessages, sendMessage
- `workers.ts` — listWorkers

### Stores (`src/stores/`)
- `auth.ts` — user writable, isSignedIn derived
- `jobs.ts` — allJobs writable, statusFilter, groupFilter, filteredJobs derived (sorted by updated_at desc)
- `activeJob.ts` — activeJobId, activeJob, activeMessages, composerEnabled derived (status === pending_user)
- `workers.ts` — allWorkers writable
- `ws.ts` — WebSocket singleton; on message: patches jobs/messages/workers stores

---

## Phase 7: Frontend — Components & Routes

**Task file:** `.agent/tasks/phase07_frontend-components-routes.md`

### Routes
```
src/routes/
├── +layout.svelte            # auth guard; redirect /login if no session
├── login/+page.svelte        # login form
└── (app)/
    ├── +layout.svelte        # 3-pane shell: Sidebar + Main
    ├── +page.svelte          # redirect to first job or empty state
    └── jobs/
        ├── new/+page.svelte  # NewJobForm
        └── [job_id]/+page.svelte  # job detail: TopBar + MessageFeed + Composer
```

### Components (Atomic Design)
```
atoms/     Badge, Button, Input, Textarea, Spinner, Avatar (U/O/W)
molecules/ JobListItem, MessageBubble, StatusBadge, FilterBar, WorkerChip
organisms/ Sidebar, TopBar, MessageFeed, Composer, NewJobForm
layout/    AppShell, AuthGuard
```

**Sidebar** — subscribes to filteredJobs store; FilterBar (status + group); "New Job" button

**TopBar** — job title, StatusBadge, WorkerChip; Cancel button (assigned/busy/pending_user); Close button (done/failed/cancelled)

**MessageFeed** — subscribes to activeMessages; auto-scroll; role styling: user=right/blue, oagent=left/purple, worker=left/green

**Composer** — disabled + tooltip when !composerEnabled; active/highlighted when status=pending_user

**NewJobForm** — title (optional), prompt (required), target_group (required), manual_worker_override (optional); "Save Draft" + "Save & Submit" buttons

---

## Phase 8: OAgent Scheduling Logic

**Task file:** `.agent/tasks/phase08_scheduler-logic.md`

### TryAssignJob(ctx, jobID)
1. Load job; verify status = queued
2. If manual_worker_override set, use that worker (if idle)
3. Else `WorkerRepo.GetLRUIdleByGroup(targetGroup)` — FOR UPDATE SKIP LOCKED
4. If worker found: DB transaction:
   - UPDATE jobs SET status='assigned', assigned_worker_id=workerID
   - UPDATE workers SET status='busy', last_active_at=NOW()
   - INSERT message (role=oagent, kind=instruction, content=job.prompt)
   - Call Dispatcher.SendToWorker(worker, job)
   - Broadcast job_updated + worker_updated
5. If no idle worker: leave queued

### ProcessQueue(ctx, targetGroup)
1. `JobRepo.DequeueOldest(targetGroup)` — FOR UPDATE SKIP LOCKED
2. If found: TryAssignJob(job.job_id)

### Worker reply → job status mapping
| Worker status | Job status   | Message kind |
|---------------|-------------|--------------|
| busy          | busy        | answer       |
| pending_user  | pending_user| question     |
| idle          | done        | answer       |
| offline       | failed      | system       |

After reply: if worker=idle → ProcessQueue(targetGroup)

### SendUserMessage flow
1. INSERT message (role=user, kind=answer)
2. UPDATE job status = busy
3. Dispatcher.NotifyUserReply(worker, job, message)
4. Broadcast via WS hub

---

## Phase 9: WAgent Integration Layer

**Task file:** `.agent/tasks/phase09_wagent-integration.md`

### OAgent → WAgent dispatch payload
```json
{
  "job_id": "uuid",
  "action": "new | continue | cancel",
  "resume_id": null,
  "instruction": "...",
  "oagent_callback_url": "http://backend:8080/api/v1/workers/reply",
  "auth_key": "<worker-specific raw key>"
}
```

### WAgent → OAgent reply payload
```json
{
  "job_id": "uuid",
  "worker_id": "uuid",
  "status": "idle",
  "message": "...",
  "resume_id": "updated-token",
  "updated_at": "2026-03-06T12:00:00Z"
}
```

### Worker auth
X-Worker-Key header → SHA-256 hash → lookup in workers.api_key_hash

### Cancel flow
CancelJob → Dispatcher.CancelWorkerJob (fire-and-forget) → UPDATE job=cancelled, worker=idle → ProcessQueue

---

## Phase 10: Wiring & Dockerization

**Task file:** `.agent/tasks/phase10_wiring-dockerization.md`

### DI Container (`internal/app/container.go`)
Uber dig: DB → repos → services (including WS hub) → controllers → router

### Middleware stack order
RequestID → Logger → Recover → CORS → (route-level: RequireSession or RequireWorkerKey)

### Startup sequence (main.go)
1. Load env, connect DB
2. Run goose migrations
3. Seed user if not exists (SEED_USERNAME / SEED_PASSWORD)
4. Build DI container
5. Start HTTP+WS server on PORT

### docker-compose.yml services
- `postgres:16-alpine` — port 5432, named volume pgdata
- `backend` — port 8080, depends on postgres
- `frontend` — port 5173, proxies `/api` to backend

---

## Critical Files (in implementation order)

1. `backend/migrations/001_initial_schema.sql` — foundation
2. `backend/internal/repository/interfaces.go` — contracts all layers depend on
3. `backend/internal/services/scheduler_service.go` — core LRU assignment logic
4. `backend/internal/ws/hub.go` — real-time event distribution
5. `backend/internal/app/container.go` — DI graph correctness
6. `frontend/src/stores/ws.ts` + `activeJob.ts` — drives all UI reactivity

---

## Verification Plan

1. `docker-compose up` — all services start cleanly
2. `curl POST /api/v1/auth/login` — returns session cookie
3. Create job via UI → sidebar shows draft
4. Submit job → status changes to queued/assigned via WS in real time
5. Simulate WAgent reply: `curl POST /api/v1/workers/reply -H "X-Worker-Key: ..."` — message appears, job status updates
6. Send user reply when pending_user — composer activates, message sent
7. Job reaches done — close button appears, soft delete works
8. Cancel flow — job cancelled, worker goes idle, next queued job assigned
9. Sidebar filters by status and target_group work
10. Page refresh — job list and messages restore from DB
