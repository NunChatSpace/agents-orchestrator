# Plan: Plan Session — Discussion-First Flow

## Context

Replace the current single-shot "type goal → Generate Plan" with a persistent,
discussion-first flow. The Plan tab becomes a chat between the user and the
selected worker agent. Full conversation history is stored in DB and resumable.

Approved by user after discussion. Streaming transport: **SSE** (Server-Sent Events).

---

## User Flow

1. User opens Plan tab → sees list of past plan sessions (pending / completed)
2. User clicks **"Let's discuss plan"** → selects a worker → new `plan_session` created (status: `pending`)
3. Worker agent sends an opening question automatically
4. User and agent exchange messages until user is satisfied
5. User clicks **"Generate Plan"** → agent produces refined prompt from full discussion context
6. User reviews generated prompt → clicks **Confirm** → status → `completed`, job created
7. If user does not confirm → session stays `pending`, can be resumed later
8. User can **Discard** a session at any time → hidden from list

---

## Plan Session States

| State | Meaning |
|---|---|
| `pending` | Active or paused — user can return and continue |
| `completed` | User confirmed the plan; generated prompt stored |
| `discarded` | User discarded; hidden from list (soft delete) |

---

## DB Schema

### New migration file: `000N_plan_sessions.up.sql`

```sql
CREATE TABLE plan_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id UUID NOT NULL REFERENCES workers(id),
    title VARCHAR(255) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    resume_id TEXT,
    generated_prompt TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_plan_sessions_deleted_at ON plan_sessions(deleted_at);
CREATE INDEX idx_plan_sessions_worker_id ON plan_sessions(worker_id);

CREATE TABLE plan_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_session_id UUID NOT NULL REFERENCES plan_sessions(id),
    role VARCHAR(50) NOT NULL, -- user | agent
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_plan_messages_session_id ON plan_messages(plan_session_id);
```

---

## API Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/plan-sessions` | List sessions (status != discarded) |
| `POST` | `/api/v1/plan-sessions` | Create new session `{ worker_id }` |
| `GET` | `/api/v1/plan-sessions/{id}` | Get session + messages |
| `PATCH` | `/api/v1/plan-sessions/{id}` | Edit title |
| `POST` | `/api/v1/plan-sessions/{id}/message` | Send user message → SSE stream of agent reply |
| `POST` | `/api/v1/plan-sessions/{id}/generate` | Final "generate plan" turn → SSE stream → stores generated_prompt |
| `POST` | `/api/v1/plan-sessions/{id}/complete` | Confirm plan; create job |
| `POST` | `/api/v1/plan-sessions/{id}/discard` | Soft-delete session |

### Streaming (`/message` and `/generate`)

- Response: `Content-Type: text/event-stream`
- Streams agent reply chunks as `data: <chunk>\n\n`
- On stream end: full reply saved to `plan_messages` (role: `agent`)
- `resume_id` updated on `plan_sessions` after each agent turn

---

## Backend Files

| File | Change |
|---|---|
| `migrations/000N_plan_sessions.up.sql` | New migration |
| `internal/models/plan_session.go` | `PlanSession` + `PlanMessage` structs |
| `internal/repository/plan_session_repository.go` | CRUD + list + messages |
| `internal/services/plan_session_service.go` | Create, send message (SSE), generate, complete, discard |
| `internal/controllers/plan_session_controller.go` | HTTP handlers |
| `internal/services/dispatcher_service.go` | Add `RunPlanStream` method (SSE-capable variant of RunPlan) |
| `main.go` | Wire DI, mount new routes |

### Key service logic

- On session create: immediately call worker CLI with opening instruction → stream back agent's first question
- On each `/message`: append user message to `plan_messages`, call CLI with `resume_id` and user message, stream reply, save reply
- On `/generate`: send final instruction *"Based on our discussion, write a clear detailed task prompt. Output ONLY the prompt text."* → stream reply → save as `generated_prompt`
- On `/complete`: set `status = completed`; caller (frontend) creates job separately via existing job API

---

## Frontend Files

| File | Change |
|---|---|
| `src/lib/apis/planSessions.ts` | New API client (list, create, get, message SSE, generate SSE, complete, discard) |
| `src/types/index.ts` | Add `PlanSession`, `PlanMessage` types |
| `src/routes/(app)/+page.svelte` | Rewrite Plan tab: session list view + session detail (chat) view |

### Plan tab UI states

**List view (default)**
- Cards: title, worker name, status badge, updated date
- "Let's discuss plan" button → worker selector dropdown → creates session → enters detail view

**Detail view**
- Chat feed (agent messages left, user messages right — NEXUS bubble style)
- Composer at bottom (textarea + Send button)
- "Generate Plan" button (active after ≥1 exchange)
- "Discard" button
- Back arrow → return to list view (session stays pending)

**Streaming UX**
- While agent is replying: show typing indicator or streaming text inline
- "Generate Plan" button shows elapsed seconds while running (same pattern as current plan button)

---

## Assumptions

1. `resume_id` mechanism is the same as jobs — stored on `plan_sessions`, passed to CLI on each turn
2. Worker agent starts the conversation — opening message is triggered automatically on session create
3. Title defaults to first user message content (truncated), editable via PATCH
4. "Generate Plan" uses same CLI pattern as current `RunPlan` but streaming
5. After `/complete`, the frontend creates the job via existing `POST /api/v1/jobs` + `POST /api/v1/jobs/{id}/submit`
6. Discarded sessions are hidden from list (soft delete via `deleted_at`), not permanently removed
7. Multiple pending sessions allowed simultaneously; each has isolated `resume_id` context

---

## Docs to Update

- `spec_v1_oagent_wagent_webapp.md` — rewrite §17.1.2 for discussion-first Plan mode; add plan session states
- `ARCHITECTURE.md` — add plan session endpoints table and new frontend files

---

## Out of Scope

- Sharing plan sessions between users
- Plan session templates
- Attaching files to plan sessions
