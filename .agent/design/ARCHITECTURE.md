# ARCHITECTURE.md

## Purpose
This document describes the project at a **diagram-first, high-level** level.

It explains:
- the main components
- how they connect
- where responsibilities live
- what is intentionally outside the system

This is **not** a code architecture document.

---

## 1. System Shape

```text
User
  -> Frontend Web App
  -> Backend / OAgent API
      -> Job Queue / State Store
      -> WAgent Runner 1 -> spawn subprocess -> WAgent1 CLI
      -> WAgent Runner 2 -> spawn subprocess -> WAgent2 CLI
      -> WAgent Runner N -> spawn subprocess -> WAgentN CLI
```

---

## 2. Main Interaction View

### 2.1 Job Creation
```text
User
  -> Frontend Web App
  -> Backend / OAgent API
  -> Job Queue / State Store
  -> WAgent Runner
  -> spawn subprocess
  -> WAgent CLI
```

### 2.2 Follow-Up In Existing Job
```text
User
  -> Frontend Web App
  -> Backend / OAgent API
  -> Existing Job / resume_id
  -> Same WAgent Runner
  -> spawn subprocess
  -> Same WAgent CLI session
```

### 2.3 Worker Needs User Input
```text
WAgent CLI
  -> WAgent Runner
  -> Backend / OAgent API
  -> Frontend Web App
  -> User
```

---

## 3. Components

### 3.1 User
The human operator.

Responsibilities:
- create jobs
- choose target group
- send instructions
- respond to pending questions
- cancel or close jobs

The user is the source of intent.

---

### 3.2 Frontend Web App
The main UI.

Responsibilities:
- render job list
- render selected job history
- show current status
- collect new job input
- collect follow-up input
- display pending questions
- allow cancel / close actions

The frontend is presentation only.
It should not own orchestration rules.

---

### 3.3 Backend / OAgent API
The control plane.

Responsibilities:
- receive requests from the frontend
- validate required fields
- create and update jobs
- own job state
- own worker assignment
- own queue logic
- store `resume_id`
- forward instructions to workers
- receive worker replies
- return state back to the frontend

This is the orchestrator.

It decides **who should work**, but does not do the work itself.

---

### 3.4 Job Queue / State Store
The system memory and scheduling layer.

Responsibilities:
- persist jobs
- persist message history
- persist assigned worker
- persist current state
- persist `resume_id`
- hold queued jobs until assignment is possible

This is owned by the backend / OAgent API.

Workers must not own queue state.

---

### 3.5 WAgent Runner
A wrapper service/process for one worker.

Conceptually:

```text
Backend / OAgent API
  -> WAgent Runner
  -> spawn subprocess
  -> CLI agent
```

Responsibilities:
- receive work from OAgent
- invoke the bound CLI tool
- run inside the correct workspace
- continue work using `resume_id`
- capture output
- return normalized worker response

The WAgent Runner is a bridge between the orchestrator and the CLI tool.

It is not the orchestrator.

---

### 3.6 CLI Agent
The actual execution tool.

Examples:
- Codex CLI
- Claude Code CLI

Responsibilities:
- inspect code
- modify files
- answer questions
- continue prior conversation
- produce summaries

The CLI agent is not directly exposed to the user.
It is always wrapped by a WAgent Runner.

---

## 4. Worker Group View

Current worker groups:

```text
fi-backend
  -> fi-backend1
  -> fi-backend2

fi-frontend
  -> fi-frontend1
  -> fi-frontend2

ib-kha
  -> ib-kha
```

Routing starts from the selected `target_group`.

The backend / OAgent API then chooses a concrete worker from that group.

---

## 5. Responsibility Boundaries

### Frontend
Owns:
- user interaction
- rendering
- local UI state

Does not own:
- queue logic
- worker selection
- job lifecycle rules

---

### Backend / OAgent API
Owns:
- job lifecycle
- queue
- worker assignment
- message forwarding
- `resume_id`
- state transitions

Does not own:
- repository edits
- direct code execution logic inside the repo

---

### WAgent Runner
Owns:
- worker-specific execution wrapper
- subprocess spawning
- mapping CLI output to normalized response

Does not own:
- queue
- routing
- cross-worker coordination
- product workflow rules

---

### CLI Agent
Owns:
- actual implementation execution

Does not own:
- system orchestration
- scheduling
- persistence
- product state model

---

## 6. Execution Model

### 6.1 Why `spawn subprocess`
The shell/process layer is an execution mechanism, not the architectural center.

The correct view is:

```text
WAgent Runner
  -> spawn subprocess
  -> CLI agent
```

Not:

```text
Backend
  -> shell
```

Reason:
- shell is only how the runner launches the tool
- lifecycle and normalization belong to the runner
- architecture should model responsibilities, not just OS calls

---

## 7. State View

### 7.1 Job State
High-level business state owned by Backend / OAgent API:

- `draft`
- `queued`
- `assigned`
- `busy`
- `pending_user`
- `done`
- `failed`
- `cancelled`

### 7.2 Worker Runtime Status
Execution signal returned by WAgent Runner:

- `busy`
- `pending_user`
- `idle`
- `offline`

Important:
- `offline` means unreachable
- `idle` means finished current round and ready

---

## 8. Continuity Model

Each job stores one `resume_id`.

Flow:

```text
Job
  -> assigned worker
  -> resume_id
  -> follow-up message
  -> same runner
  -> same CLI conversation path
```

This avoids keeping a permanently open interactive process while still preserving continuity.

---

## 9. Why Web App Instead of Discord

Discord was rejected because it weakens the architecture in key areas:

- session/job boundaries are weak
- queue visibility is poor
- state visibility is poor
- follow-up continuity is harder to model
- orchestration becomes harder to inspect

A dedicated web app makes jobs, queue, worker assignment, and state first-class.

---

## 10. v1 Constraints

This architecture intentionally excludes:
- file attachments
- image input
- multi-worker collaboration on one job
- natural-language smart routing without `target_group`
- automatic retry
- worker-owned queue
- autonomous background planning
- Discord integration

These are deferred by design.

---

## 11. Design Intent

This system is designed to be:

- explicit
- deterministic
- resumable
- inspectable
- queue-driven
- easy to reason about

It is **not** designed to be highly autonomous in v1.

The main goal is clear control flow:
- user intent enters from the frontend
- orchestration happens in the backend
- execution happens in worker runners
- actual code work happens in CLI agents
