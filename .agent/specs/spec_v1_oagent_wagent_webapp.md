# spec_v1.md — OAgent / WAgent Job-Oriented Web App (v1)

## 1. Goal
Build a simple web app for personal use to manage AI worker agents through one orchestrator agent.

The system should let the user:
- create a job
- choose a target worker group
- send prompts to that job
- continue the conversation in the same job
- let OAgent assign the job to a worker in the selected group
- keep the worker conversation resumable via `resume_id`

This is a **job-centric** design.

---

## 2. Core Model

### 2.1 Main Rule
**1 job = 1 session**

A job is both:
- the work item
- the conversation container
- the unit that owns `resume_id`

The user continues work by sending follow-up messages into the same job.

If the user wants to start a different topic, the user must create a new job.

---

## 3. Actors

### 3.1 User
Creates jobs, sends follow-up messages, answers worker questions, cancels jobs, closes jobs.

### 3.2 OAgent
Responsible for:
- holding the central job queue
- assigning jobs to workers
- forwarding user messages to workers
- forwarding worker replies back to the user
- storing job/session state
- storing `resume_id`

### 3.3 WAgent
A worker bound to a specific workspace and underlying CLI agent.

Examples:
- `fi-backend1 (Codex)`
- `fi-backend2 (Codex)`
- `fi-frontend1 (Codex)`
- `fi-frontend2 (Codex)`
- `ib-kha (Claude)`

WAgent does **not** manage queue logic.

---

## 4. Worker Groups

### 4.1 Defined Groups
Groups are not hardcoded in the frontend. `TargetGroup` is typed as `string`. Groups are derived at runtime from the `group_name` field of registered workers.

Current examples: `fi-backend`, `fi-frontend`, `ib-kha`.

### 4.2 Group Mapping
- `fi-backend` → `fi-backend1`, `fi-backend2`
- `fi-frontend` → `fi-frontend1`, `fi-frontend2`
- `ib-kha` → `ib-kha`

---

## 5. Assignment Rules

### 5.1 Input Requirement
User must choose `target_group` when creating a job.

If `target_group` is missing, OAgent must ask the user.

### 5.2 Worker Selection
When a queued job is ready to be assigned:
1. If `manual_worker_override` exists, use that worker.
2. Otherwise, pick an `idle` worker in the target group.
3. If multiple workers are `idle`, choose **least recently used**.
4. If no worker is `idle`, keep the job in queue.

### 5.3 Queue Ownership
Queue belongs only to OAgent.

Workers do not know queue state.

---

## 6. Job Queue Model

### 6.1 Queue Type
There is one OAgent-managed job queue.

Each queue item belongs to exactly one target group.

### 6.2 Queue Behavior
- queued jobs wait until a worker in the selected group becomes available
- worker selection happens only when assignment is possible
- queue behavior is FIFO within OAgent's scheduling flow
- worker itself does not manage queued items

---

## 7. Job Lifecycle

### 7.1 States
- `draft`
- `queued`
- `assigned`
- `busy`
- `pending_user`
- `done`
- `failed`
- `cancelled`

### 7.2 State Meaning

#### `draft`
User is still composing or required fields are not complete.

#### `queued`
Job is created and waiting for assignment.

#### `assigned`
OAgent selected a worker for this job.

#### `busy`
Worker is actively processing the job.

#### `pending_user`
Worker is waiting for user input, permission, or a choice.

#### `done`
Current round is complete and summary is available.

#### `failed`
Current round failed.

#### `cancelled`
Job was cancelled by the user.

### 7.3 Typical Flow
`draft -> queued -> assigned -> busy -> done`

Possible side paths:
- `busy -> pending_user -> busy`
- `busy -> failed`
- `assigned -> cancelled`
- `busy -> cancelled`

---

## 8. Worker Runtime Status

### 8.1 Status Values
- `busy`
- `pending_user`
- `idle`
- `offline`

### 8.2 Meaning

#### `busy`
Worker is processing.

#### `pending_user`
Worker needs an answer from the user.

#### `idle`
Worker finished the current round, sent summary, stored `resume_id`, and is ready for more work.

#### `offline`
Worker service is unreachable.

### 8.3 Important Rule
`offline` means unreachable service only.

It must **not** mean "finished".

Finished state is represented by `idle`.

---

## 9. Conversation Continuity

### 9.1 Resume Rule
Each job owns one `resume_id`.

When the user sends follow-up into the same job:
- OAgent forwards the message to the same worker
- worker continues with the same `resume_id`

### 9.2 New Topic Rule
If the user wants a new topic, create a new job.

Do not reuse the old job for unrelated work.

---

## 10. Message History Model

Each job stores `messages[]`.

### 10.1 Message Fields
- `message_id`
- `job_id`
- `role` = `user | oagent | worker`
- `kind` = `instruction | question | answer | summary | system`
- `content`
- `created_at`

### 10.2 Notes
- `resume_id` is stored on the job/session level, not per message
- every follow-up message is appended into the same job history

---

## 11. Job Schema

### 11.1 Minimum Fields
- `job_id`
- `title`
- `prompt`
- `target_group`
- `assigned_worker_id?`
- `manual_worker_override?`
- `status`
- `resume_id?`
- `messages[]`
- `created_at`
- `updated_at`

---

## 12. Create Job Action

### 12.1 Form Fields
- `target_group` **required**
- `prompt` **required**
- `title` optional
- `manual_worker_override` optional

### 12.2 Rules
- title defaults from the first prompt if not provided
- no priority field in v1
- no file/image attachment in v1

---

## 13. Follow-up Action

A follow-up means sending a new user message into an existing job.

Rules:
- follow-up stays in the same job
- follow-up uses the same `resume_id`
- follow-up keeps the same `target_group`
- follow-up keeps the same assigned worker unless new assignment logic is explicitly triggered later

---

## 14. Pending User Action

If worker asks for permission or requires a decision:
- job status becomes `pending_user`
- user can reply in the same job
- once user replies, job goes back to `busy`

Examples:
- approve command execution
- choose one option
- answer clarification requested by worker

---

## 15. Cancel and Close Rules

### 15.1 Cancel Job
User may cancel when status is:
- `assigned`
- `busy`
- `pending_user`

After successful cancel:
- job status becomes `cancelled`

### 15.2 Close Job
User may close when status is:
- `done`
- `failed`
- `cancelled`

Close means:
- no more follow-up in this job
- user must create a new job to continue with a new topic

---

## 16. Failure / Timeout Rules (v1)

### 16.1 No Auto Retry
There is no automatic retry in v1.

Retry must be initiated by the user.

### 16.0 Plan Generation Failures (RunPlan)

`RunPlan` is a synchronous CLI call with a 2-minute timeout. Failures surface as human-readable messages:

- Timeout: `"plan generation timed out after 2 minutes"`
- CLI stderr present: `"agent CLI failed: <stderr text>"`
- Non-zero exit, no stderr: `"agent CLI exited with code N"`

These are returned as `422 PLAN_ERROR` from `POST /workers/{id}/plan` and displayed in the Plan tab error block.

### 16.2 Failure Reasons
Store a normalized `failure_reason` when possible, for example:
- `ack_timeout`
- `run_timeout`
- `worker_error`
- `cancelled_by_user`

### 16.3 Recovery
If a job failed:
- job remains visible
- user may inspect messages
- user may create a new job for retry or continuation

---

## 17. UI Model

### 17.1 Layout

The app uses a persistent top bar for global navigation with no sidebar.

**Persistent top bar** (`routes/(app)/+layout.svelte`):

- Logo (`◈ NEXUS`) — links to `/`
- Workspace dropdown — one entry per unique `group_name` derived from `allWorkers`; updates `selectedGroup` store; persists across navigation
- Nav links: **Agents** (→ `/`, active when path is `/` or starts with `/agents`) | **Plans** (→ `/plans`, active when path starts with `/plans`) | **Office** (→ `/office`, active when path starts with `/office`)

**Pages:**

- `/` — Agents page (agent grid, filtered by selected workspace)
- `/agents/{id}/jobs` — Agent job list
- `/agents/{id}/settings` — Agent settings + health check
- `/plans` — Plans page (discussion-first job creation)
- `/office` — Office page (2D map interaction surface)
- `/jobs/{id}` — Job chat feed

### 17.1.1 Agents Page (`/`)

The home page shows agent cards filtered by the selected workspace group.

**Agent card:**

- Header: name, group, status pill
- Body: workspace path, CLI command, last active time
- Clicking the card header or body navigates to `/agents/{id}/jobs` (primary action)
- Footer buttons:
  - **Jobs** → `/agents/{id}/jobs`
  - **Settings** → `/agents/{id}/settings`
  - **VSCode** → `vscode://file{hostWorkspace}` where `hostWorkspace` is the worker's container path with the `/workspaces` prefix replaced by `WORKSPACES_PATH` (a Vite public env var set in `.env`, same value as the Docker bind-mount path). Falls back to the raw container path if the var is unset.

### 17.1.1a Agent Sub-Pages (`/agents/{id}/…`)

A shared layout (`routes/(app)/agents/[worker_id]/+layout.svelte`) renders the agent header (name, status, group/workspace, delete button) and a **Jobs | Settings** sub-nav bar. Child pages:

- **Jobs** (`/agents/{id}/jobs`) — job list filtered to `assigned_worker_id === id`, sorted by `updated_at DESC`. Clicking a row navigates to `/jobs/{job_id}`. **New Job** button opens a modal to dispatch a new job directly to this agent.
- **Settings** (`/agents/{id}/settings`) — displays CLI command, last active time, workspace path, git repo URL. Includes **Run Health Check** button that calls `POST /api/v1/workers/{id}/ping`. Includes **Office Position** fields (`map_x`, `map_y`) saved via `PATCH /api/v1/workers/{id}`.

### 17.1.2 Plans Page (`/plans`) — Discussion-First

The Plans page is a discussion-first job creation flow backed by `plan_sessions`.

**Plan session states:** `pending` | `completed` | `discarded`

**Discussion prompt framing:** Every `/message` turn prepends a hidden scoping instruction before sending to the agent: _"You are in a task scoping conversation. Your role is to ask focused clarifying questions… Keep responses short and conversational — 2-4 sentences max. Do not execute any code or make file changes. If the user's goal seems too broad or vague, say so briefly and suggest a narrower starting point."_ The `/generate` turn uses its own fixed instruction and is unaffected.

**Plans page — list view (default):**

- Shows all non-discarded plan sessions for the user (title, status badge, relative time).
- Worker selector dropdown (filtered by selected workspace group) + **"Let's discuss plan"** button to create a new session.
- Sessions are sorted by `updated_at DESC`.

**Plans page — session detail view (chat):**

1. User and the selected worker agent exchange messages in a chat feed.
2. Agent messages are streamed via SSE (`POST /api/v1/plan-sessions/{id}/message`).
3. When the user is satisfied, clicks **Generate Plan** (enabled after ≥ 2 messages).
4. Backend sends the final instruction to the agent: `"Based on our discussion, write a clear, detailed task prompt. Output ONLY the prompt text."` — streamed via SSE (`POST /api/v1/plan-sessions/{id}/generate`).
5. **Generate Plan** button shows elapsed seconds while the agent is working (`Generating… 14s`).
6. Generated prompt fills an editable textarea. User edits if needed, then clicks **Confirm & Create Job**.
7. `POST /api/v1/plan-sessions/{id}/complete` → session status → `completed`. Then `POST /api/v1/jobs` + `POST /api/v1/jobs/{id}/submit` → navigate to `/jobs/{id}`.
8. **Discard** button soft-deletes the session (status → `discarded`, hidden from list).

**Resuming a pending session:** User clicks a pending session in the list → chat feed reloads from DB → user can continue where they left off. The worker agent receives the full prior context via `resume_id`.

**Session title:** Auto-set from the first user message (truncated to 80 chars), editable via `PATCH /api/v1/plan-sessions/{id}`.

**Streaming:** Each agent turn is streamed via SSE. The frontend receives `event: thinking` events (intermediate steps, displayed as a pulsing indicator) and a final `event: done` event with the complete reply.

**On error:** An "Error" label appears above the error message inside the chat view.

### 17.1.3 Office Page (`/office`) — 2D Interaction Map

The Office page renders a top-down tile map via HTML5 Canvas with no game engine dependency.

Core behavior:

- Worker agents appear as NPCs at desk coordinates resolved in this order:
  1. explicit worker `map_x`/`map_y` when either is non-zero
  2. pre-configured desk mapping
  3. group-based fallback placement by registration order (`created_at`)
- Worker visual state updates live from the existing WebSocket worker events (`allWorkers` store)
- User avatar movement:
  - keyboard: WASD / Arrow keys
  - mobile: on-screen D-pad
  - speed: 4 tiles/second
  - collisions: walls + desk tiles
- Proximity interaction:
  - prompt appears when within 2 tiles of nearest worker desk
  - `E` or click worker opens side panel
- Interaction panel includes:
  - worker header (name, group, status, workspace)
  - active job (if present)
  - last 3 jobs for that worker (derived from existing `/api/v1/jobs` list; no new backend endpoint)
  - actions: **New Job**, **View All Jobs**, **Settings**
- New Job flow from panel reuses existing modal/form with:
  - prefilled `target_group`
  - prefilled `manual_worker_override`
  - submit path remains `POST /api/v1/jobs` + `POST /api/v1/jobs/{id}/submit`

### 17.2 Design System — NEXUS Theme

The frontend uses a cyber-premium dark design system referred to as NEXUS.

#### Colors

| CSS Token | Value | Usage |
| --- | --- | --- |
| `--nx-bg` | `#05050c` | Page background |
| `--nx-surface` | `#0e0e20` | Panel / card base |
| `--nx-violet` | `#7c3aed` | Primary brand |
| `--nx-vb` | `#8b5cf6` | Active states, glows |
| `--nx-lav` | `#a78bfa` | Secondary accents |
| `--nx-soft` | `#c4b5fd` | Highlight text |
| `--nx-border` | `rgba(139,92,246,0.18)` | Default border |
| `--nx-borderb` | `rgba(139,92,246,0.40)` | Focused/active border |

Tokens are defined in `frontend/src/app.css` under `:root` and mirrored in `tailwind.config.js` under `theme.extend.colors.nx`.

#### Typography

- Body: **Inter** (weights 300–600)
- Display / headings / job titles: **Space Grotesk** (weights 400–700)
- Loaded via Google Fonts in `app.html`

#### Visual Patterns

- Subtle 40×40 px crosshatch grid overlay on `body::before` at 2.5% opacity
- Glassmorphism cards: `backdrop-filter: blur(18px)` + semi-transparent dark background + 1 px violet border + top gradient line via `::before`
- Active sidebar item: violet left-bar glow (2.5 px, `box-shadow: 0 0 8px #8b5cf6`)
- Buttons: primary = violet gradient + `box-shadow` glow, secondary = ghost with violet border, danger = red-tinted ghost

#### Reusable CSS Classes (defined in `app.css` `@layer components`)

| Class | Purpose |
| --- | --- |
| `.nx-card` | Glassmorphism panel with top gradient line |
| `.nx-input` | Dark glass input / textarea / select |
| `.nx-label` | Uppercase, muted section label |

### 17.3 Sidebar

**Removed.** Navigation is handled by the persistent top bar (§17.1). Job history is accessible via the agent jobs page (`/agents/{id}/jobs`).

### 17.4 Main Pane

Show:

- full message history for one job
- composer for follow-up / reply / cancel / close
- top metadata bar

Message bubble appearance:

- `user` role → violet gradient bubble, right-aligned
- `oagent` / `worker` role → dark glass bubble with lavender text, left-aligned
- `system` kind → muted italic, left-aligned
- `instruction` kind → prominent full-width block with a labelled header ("Instruction sent to agent"), shown before other feed messages; not a bubble

All bubble content and instruction blocks render Markdown (parsed with `marked`, sanitized with `DOMPurify`). Supports bold, italic, code spans, fenced code blocks, lists, blockquotes, and tables.

### 17.5 Top Bar

Show:

- `title` (Space Grotesk, truncated)
- `target_group` + `assigned_worker_name` (muted meta line)
- `status` badge
- Cancel / Close action buttons when applicable

Style: sticky, `backdrop-filter: blur(20px)`, violet bottom border

---

## 18. Composer Behavior

### 18.1 When Job is `busy`
Composer may stay visible, but input is conceptually a follow-up to the same job.

### 18.2 When Job is `pending_user`
Composer must stay active and clearly indicate that the worker is waiting for user input.

### 18.3 When Job is `offline`
Composer should be disabled and recovery action should be offered.

### 18.4 v1 Input Type
Text only.

No files. No images.

---

## 19. Worker Integration Model

### 19.1 Runtime
Each WAgent runs as a dedicated service/process.

Example conceptual model:
- one worker service per worker
- fixed workspace per worker
- fixed underlying CLI agent per worker

### 19.2 Worker Service Responsibility
- receive instruction from OAgent
- invoke CLI agent in its workspace
- continue previous conversation using `resume_id`
- send back status + message + summary
- return updated `resume_id`

### 19.3 Queue Rule
Worker service does not know queue state.

Queue exists only in OAgent.

---

## 20. Reply Contract v1

Worker service must send back at least:

- `job_id`
- `worker_id`
- `status` = `busy | pending_user | idle | offline`
- `message`
- `resume_id?`
- `updated_at`

This is enough for OAgent to update the UI and job state.

---

## 21. Non-Goals for v1

Do not include in v1:
- Discord integration
- multi-worker collaboration on one job
- capability matching
- auto smart routing from natural language
- file uploads
- image uploads
- priority scheduling
- automatic retry
- worker-managed queue
- model-generated job titles

---

## 22. Acceptance Criteria

### 22.1 Job Creation
- user can create a job with `target_group` and `prompt`
- job appears in sidebar

### 22.2 Assignment
- OAgent assigns queued job to an idle worker in the target group
- if multiple idle workers exist, least recently used is chosen

### 22.3 Continuation
- user can send follow-up messages in the same job
- system continues through the same `resume_id`

### 22.4 Pending User
- worker can request user input
- job enters `pending_user`
- user reply sends the job back to `busy`

### 22.5 Completion
- worker can finish a round
- job becomes `done`
- summary is visible in message history

### 22.6 Cancellation
- user can cancel a job when status is `assigned`, `busy`, or `pending_user`

### 22.7 Close
- user can close job when status is `done`, `failed`, or `cancelled`

### 22.8 Visibility
- sidebar shows jobs with title, group, worker, status, and updated time
- main pane shows full history of selected job

### 22.9 Office Rendering
- `/office` renders a tile-based 2D office map with worker NPCs
- worker status visuals update in real-time without page reload

### 22.10 Office Interaction
- user can move avatar with keyboard and mobile D-pad
- collision blocks movement through wall and desk tiles
- proximity prompt appears within 2 tiles and `E`/click opens interaction panel

### 22.11 Office Dispatch
- New Job action from the interaction panel opens prefilled modal
- submit creates + dispatches job successfully to selected worker

---

## 23. Open Questions for v2
Not part of v1, but likely future topics:
- should one group have weighted worker selection instead of LRU?
- should a job be reassigned if a worker becomes offline?
- should we support attachments?
- should we support per-job templates?
- should we support worker-specific structured actions?
