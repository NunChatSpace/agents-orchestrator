# Workspace Preview Runtime Spec v1

## 1. Purpose
Define a deterministic runtime contract for manual preview environments built from agent workspaces before approval.

Goals:
- The user can review a preview without requiring a Git commit or Git push first.
- Preview artifacts are built from the agent's current local workspace state, including uncommitted changes.
- Each agent builds its own role image independently and pushes to a shared local OCI registry.
- The user assembles images from one or more agents into a named **deployment plan** and applies it on demand.
- Multiple deployment plans may run simultaneously.
- The orchestrator can create, inspect, start, stop, and destroy deployment plans without repo-specific if/else logic.

Non-goals:
- Automatic preview creation on every task update.
- Pre-approval Git commits as a prerequisite for preview.
- Shared app runtime containers across deployment plans.
- Production-grade multi-tenant preview hosting.

---

## 2. High-Level Model

### Shared Infrastructure
Shared across all previews on one host:
- Postgres instance (platform, not for preview app data)
- Local OCI registry (`registry:2` container, `registry:5000` internal / `localhost:5001` on host)
- Reverse proxy (nginx)
- Optional MinIO for preview app storage

### Worker Build
Each agent independently builds its role image from its current workspace state and pushes it to the local OCI registry.
A `WorkerBuild` record tracks one build attempt per agent.

### Deployment Plan
A **DeploymentPlan** is a named, user-created unit that:
- selects one `WorkerBuild` per required stack role (one agent per role)
- has a user-chosen **slug name** used in the preview URL
- can be applied (deploy) and stopped independently
- multiple plans may run at the same time

### Build Modes
At deployment plan creation time, the user selects a build mode:
- **Fresh**: force each assigned agent to run a new build before deploying
- **Latest**: use each agent's most recent `ready` build; if none exists, trigger a fresh build

### Isolation Rule
Each deployment plan runs in isolated containers. Shared infra (registry, postgres, nginx) is allowed; shared app containers are not.

---

## 3. Core Terminology

### Agent / Worker
A worker that executes tasks and owns a local workspace checkout.

### Task / Job
A single implementation or review job assigned to one agent. The agent's build button is visible only when the agent has an active task (the workspace has active changes to preview).

### Workspace State
The local agent workspace contents at build time. May include uncommitted changes.

### WorkerBuild
A per-agent image build record. One build attempt = one record.
Not scoped to any specific task. An agent builds from its current workspace regardless of which task is active.

Key fields: `id`, `worker_id`, `stack_id`, `role`, `status`, `image_reference`, `image_digest`, `build_mode`, `error_message`, `created_at`, `completed_at`.

### DeploymentPlan
A named deployment assembled from worker builds, identified by a slug. The slug is used directly in the preview URL.

Key fields: `id`, `name` (slug), `stack_id`, `build_mode`, `status`, `preview_url`, `roles[]`.

### DeploymentPlanRole
One row per required stack role within a deployment plan.

Key fields: `plan_id`, `role`, `worker_id`, `worker_build_id`, `container_name`, `host_port`.

### Build Manifest
Retained from the v1 bundle system for audit. Each worker build report creates one manifest record.

### Stack Registry
A file-based registry in the orchestrator repo defining roles, worker groups, and deployment services per stack.

---

## 4. Design Principles

1. No pre-approval Git push
   - Agents must not be forced to commit or push code before preview.
   - Preview artifacts come from the current local workspace state.

2. Agent-centric build trigger
   - Build is initiated from the agent card, not the job detail page.
   - The Build button is visible only when the agent has an active job.

3. Fresh or Latest mode per deployment plan
   - Fresh: force new build for every role in the plan.
   - Latest: reuse each agent's most recent ready build. Trigger fresh if none exists.

4. Immutable deploy inputs
   - Deployment must use immutable image digests, not mutable tags.

5. File-based stack registry
   - Stack definitions are versioned with the orchestrator codebase.
   - Runtime stack resolution must not depend on ad hoc repo-specific logic.

6. Recoverable failures
   - One failed role build must not invalidate other roles or other deployment plans.

7. Multiple simultaneous deployments
   - Each deployment plan is independent. Multiple plans can be `running` at the same time.
   - Plans are distinguished by name (slug), not by agent identity.

8. Destroyability
   - Every deployment plan must be stoppable and removable.
   - Cleanup must leave no orphan containers or stale proxy routes.

---

## 5. Stack Registry Contract

The orchestrator owns the stack registry. Stored as files in the orchestrator repo, loaded at startup.

### Required Fields Per Stack
- `stack_id`
- `display_name`
- `deployment_template`
- `roles[]`

### Required Fields Per Role
- `role`
- `worker_group`
- `service_name`
- `healthcheck_path`
- `required_image_labels`

### Current Expected Roles
For the current previewable web app:
- `frontend`
- `backend`

### Rules
- `stack_id` must be unique.
- Role names must be unique within a stack.
- Worker routing is resolved by `worker_group`, not hardcoded worker IDs.
- The registry must be readable without requiring database state.
- The deployment plan UI presents one agent picker per role, filtered by that role's `worker_group`.

---

## 6. Required Runtime Contract Per Repo

Every repo that participates in a previewable stack must support image builds from local workspace state.

### Required Build Inputs
- repo-local container build definition (`Dockerfile` or documented equivalent)
- `.env.preview.example`
- `Makefile`
- optional `scripts/healthcheck.sh` if no health endpoint exists

### Required Make Targets
- `make preview-build`
- `make test`
- `make smoke`

### Required Artifact Expectations
- build must work from the current local workspace state
- build must not require a Git commit
- build output must be pushable to the local OCI registry at `registry:5000`
- build output must include role-specific labels required by the stack registry

### Required Health Endpoints
- Frontend: returns HTTP 200 on preview route root or `/healthz`
- Backend: returns HTTP 200 on `/healthz`

### Required Env Variables
#### Preview Identity
- `PLAN_ID`
- `STACK_ID`
- `ROLE`
- `AGENT_ID`

#### Routing
- `PREVIEW_BASE_DOMAIN`
- `PREVIEW_HOSTNAME`

#### Backend Runtime
- `DATABASE_URL`
- `DJANGO_SECRET_KEY`
- `DJANGO_ALLOWED_HOSTS`
- `DJANGO_CSRF_TRUSTED_ORIGINS`
- `MEDIA_BUCKET`
- `MEDIA_PREFIX`
- `S3_ENDPOINT`
- `S3_ACCESS_KEY_ID`
- `S3_SECRET_ACCESS_KEY`

#### Frontend Runtime
- `PUBLIC_API_BASE_URL`
- `PUBLIC_PLAN_ID`
- `PUBLIC_PREVIEW_URL`

---

## 7. Shared Infrastructure Specification

### 7.1 OCI Registry
One local OCI registry (`registry:2`) stores preview artifacts for all builds.

Registry address:
- Inside Docker network: `registry:5000`
- From host: `localhost:5001`
- Workers push to: `registry:5000/{worker_name}/{role}:{timestamp}`

Rules:
- every reported artifact must be addressable by immutable digest
- tags may exist for convenience, but deploy must resolve to digest
- image naming must include `worker_name` and `role` to avoid collisions

### 7.2 Postgres (Preview App)
One fresh Postgres database per deployment plan for the preview app backend.

Per-plan requirements:
- unique database name: `preview_{plan_slug}`
- unique database user
- unique password

Rules:
- orchestrator creates DB and credentials before backend container starts
- migrations run only against the plan database
- destroyed plans must have their DB dropped

### 7.3 MinIO (Optional)
If the preview app uses object storage, one MinIO instance may be shared.

Preferred isolation: one bucket per deployment plan (`preview-{plan_slug}`).

### 7.4 Reverse Proxy

#### Preview URL Format
Subdomain-based routing keyed by **deployment plan slug**:

```text
http://shiphide.{plan-name}.preview
```

Examples:
- `http://shiphide.v1.preview`
- `http://shiphide.feature-cart.preview`
- `http://shiphide.hotfix-1.preview`

`{plan-name}` is the user-supplied slug, validated as `[a-z0-9-]+`, max 40 chars.

#### Nginx Config Management
The nginx `proxy` container serves preview routes alongside the main app.

Strategy: the backend generates and writes `infra/nginx/preview-upstreams.conf` (included by `default.conf`) and signals nginx to reload after each deploy or stop.

Rules:
- nginx reload is triggered by `docker exec proxy nginx -s reload` from the backend process
- each running plan gets one upstream block and one server block in the generated file
- stopped or failed plans are removed from the generated file on cleanup
- the `*.preview` server block in `default.conf` delegates to upstream entries from the generated file

#### DNS Requirements
- Mac: `127.0.0.1 shiphide.{plan-name}.preview` in `/etc/hosts` per plan name, OR install host-based dnsmasq (`brew install dnsmasq`) for automatic wildcard `*.preview` resolution
- Windows (LAN access): `{mac-ip} shiphide.{plan-name}.preview` in `C:\Windows\System32\drivers\etc\hosts` per plan name
- Wildcard LAN DNS is not assumed available in v1

---

## 8. Lifecycle

### 8.1 WorkerBuild States
- `queued`: build triggered, not yet started by agent
- `building`: agent is actively building and pushing
- `ready`: build succeeded; `image_reference` and `image_digest` are populated
- `failed`: build failed; `error_message` is populated

Transition rules:
- trigger → `queued`
- agent picks up → `building`
- agent reports success → `ready`
- agent reports failure → `failed`
- a new build for the same worker supersedes old builds but does not delete them (append-only history)

### 8.2 DeploymentPlan States
- `pending`: plan created; waiting for builds to complete (Fresh mode)
- `deploying`: all required builds are `ready`; containers starting
- `running`: all health checks passed; preview URL is live
- `failed`: deploy failed or health checks did not pass
- `stopped`: explicitly stopped by user; containers removed; proxy route removed

Transition rules:
- create with Latest mode + all builds ready → skip pending, go directly to `deploying`
- create with Fresh mode → `pending` until all role builds reach `ready`
- all role builds ready → `deploying`
- health checks pass → `running`
- health check failure or container error → `failed`
- user stops → `stopped`
- multiple plans may be `running` simultaneously

### 8.3 Legacy Preview Bundle
The `preview_bundles` table and service remain in place from the earlier prototype. They are not actively used by the new deployment plan flow but are not removed in v1. The deployment plan system is the canonical path for previewing agent work going forward.

---

## 9. Build and Deploy Flow

### 9.1 Agent Build Trigger
Entry point: **agent card on the Agents page** (not the job detail page).

Visibility rule: the Build button is visible on an agent card only when that agent has at least one active job (status `assigned`, `busy`, or `pending_user`).

User flow:
1. User opens the Agents page.
2. User finds the agent card for the relevant worker.
3. User clicks **Build** → a mode picker appears: **Fresh** or **Latest**.
4. User selects mode and confirms.
5. Backend creates a `WorkerBuild` record (`status=queued`) and dispatches the build instruction to the agent.
6. Agent builds from current workspace, pushes image to `registry:5000/{name}/{role}:{timestamp}`.
7. Agent calls `POST /api/v1/workers/{id}/build-reports` with image reference, digest, and metadata.
8. Backend updates `WorkerBuild` to `ready` or `failed`.
9. Agent card shows current build status and last build info.

### 9.2 Build Dispatch Mechanism
The backend dispatches the build instruction using the existing `DispatcherService.SendToWorker` with a synthetic build-type job that carries:
- registry address
- expected image tag format
- build report callback URL
- `BUNDLE_ID` / `PLAN_ID` if available

The worker's CLI receives this job and runs the build and push steps.

### 9.3 Deployment Plan Creation
Entry point: **Deployments page** → **New Plan** button.

User flow:
1. User opens the Deployments page (`/deploys`).
2. User clicks **New Plan**.
3. User enters a **plan name** (slug, e.g. `feature-cart`).
4. User selects a **stack** (e.g. `fi-web-app`).
5. User selects a **build mode**: Fresh or Latest.
6. User assigns one agent per required role (filtered by `worker_group`).
   - Fresh: each agent will be forced to build before deploy starts.
   - Latest: the most recent `ready` build for each agent is used; if none, a fresh build is triggered.
7. User clicks **Deploy**.
8. Backend creates a `DeploymentPlan` record.
9. If Fresh or any Latest agent has no ready build:
   - Plan enters `pending` state; backend triggers builds for agents that need them.
   - When all role builds reach `ready`, plan advances to `deploying`.
10. Backend generates docker-compose YAML for the plan, runs containers.
11. Backend writes nginx upstream config entry and reloads nginx.
12. Backend polls health endpoints until all pass → `running`.
13. Preview URL `http://shiphide.{plan-name}.preview` is displayed.

### 9.4 Agent Build Rules
- Agent builds from its own local workspace state.
- Agent must not be required to commit or push code first.
- Agent pushes the built artifact to `registry:5000`.
- Agent reports immutable digest and build metadata back to the orchestrator.

### 9.5 Approval Rule
Preview approval does not imply code push. Git commit and push remain separate, post-approval actions.

---

## 10. Build Manifest Contract

Each successful or failed role build report creates one build manifest record.

### Required Manifest Fields
- `manifest_id`
- `worker_build_id`
- `stack_id`
- `role`
- `worker_id`
- `image_reference`
- `image_digest` when status is `ready`
- `status`
- `metadata`
- `created_at`

### Metadata Expectations
Metadata should be sufficient to trace:
- which worker built the artifact
- which workspace or local build context was used
- any builder-side labels or auxiliary references needed for audit/debug

---

## 11. API Contract

### Worker Build Endpoints (session-authenticated)

#### Trigger Build
`POST /api/v1/workers/{worker_id}/builds`

Request:
```json
{ "stack_id": "fi-web-app", "role": "backend", "mode": "fresh" }
```

Response: `WorkerBuildResponse`

#### List Builds
`GET /api/v1/workers/{worker_id}/builds`

Response: `WorkerBuildResponse[]` ordered by `created_at DESC`

### Worker Build Report (worker-key-authenticated)
`POST /api/v1/workers/{worker_id}/build-reports`

Request:
```json
{
  "worker_build_id": "...",
  "status": "ready",
  "image_reference": "registry:5000/fi-backend1/backend:20241201-120000",
  "image_digest": "sha256:...",
  "metadata": {},
  "error_message": ""
}
```

### Deployment Plan Endpoints (session-authenticated)

| Method | Path | Notes |
|---|---|---|
| GET | `/api/v1/deployment-plans` | List all plans for current user |
| POST | `/api/v1/deployment-plans` | Create and start deploying |
| GET | `/api/v1/deployment-plans/{id}` | Get plan detail with role states |
| POST | `/api/v1/deployment-plans/{id}/stop` | Stop containers, remove proxy route |
| DELETE | `/api/v1/deployment-plans/{id}` | Permanently remove (`failed` or `stopped` only) |

#### Create DeploymentPlan Request
```json
{
  "name": "feature-cart",
  "stack_id": "fi-web-app",
  "build_mode": "latest",
  "roles": [
    { "role": "backend",  "worker_id": "..." },
    { "role": "frontend", "worker_id": "..." }
  ]
}
```

#### DeploymentPlan Response
```json
{
  "id": "...",
  "name": "feature-cart",
  "stack_id": "fi-web-app",
  "build_mode": "latest",
  "status": "running",
  "preview_url": "http://shiphide.feature-cart.preview",
  "created_at": "...",
  "updated_at": "...",
  "roles": [
    {
      "role": "backend",
      "worker_id": "...",
      "worker_build_id": "...",
      "image_reference": "registry:5000/fi-backend1/backend:20241201-120000",
      "image_digest": "sha256:...",
      "container_name": "preview-feature-cart-backend",
      "build_status": "ready",
      "deploy_status": "running"
    }
  ]
}
```

---

## 12. Security Requirements

1. Agents building preview images require deliberate Docker/build capability.
2. Agents building preview images require registry credentials scoped for preview artifacts.
3. Registry credentials must not be broader than necessary.
4. Preview Postgres DB credentials are generated per plan and not shared.
5. Preview hostnames are internal-only convenience endpoints in v1.
6. Nginx reload command is executed within the Docker network by the backend container; it must not be exposed as an API endpoint.

---

## 13. Failure Handling

### Build Failure
If a worker reports a failed build:
- `WorkerBuild` status set to `failed`; error message stored
- If a `DeploymentPlan` in `pending` state depends on this build, the plan transitions to `failed`
- A new build may be triggered for the same worker; a new `WorkerBuild` record is created

### Deploy Failure
If deploy fails after all builds are ready:
- Keep recorded digests
- Mark plan `failed`
- Do not silently rebuild artifacts

### Stop Failure
If container teardown only partially succeeds:
- No intermediate `stopping` status is introduced in v1
- Stop returns an operation error and cleanup remains retryable
- Nginx route is only removed after containers are confirmed stopped

---

## 14. Acceptance Criteria

1. User can trigger a docker image build for an agent from the agent card without committing or pushing code.
2. Build button is only visible when the agent has an active job.
3. Fresh mode forces a new build; Latest mode reuses the most recent ready build (fallback to fresh if none).
4. A deployment plan can combine builds from different agents, one per required role.
5. Multiple deployment plans can be `running` simultaneously.
6. Each plan is accessible at `http://shiphide.{plan-name}.preview`.
7. Stopping a plan removes its containers and nginx route without affecting other running plans.
8. Plan name must be a valid slug (`[a-z0-9-]+`, max 40 chars); duplicate names are rejected.
9. Hosts-file entries are required per plan name per device; this limitation is documented.
10. No step in preview creation requires a Git commit or Git push.
11. Build history per agent is preserved across multiple builds (append-only).

---

## 15. Phase Plan

### Phase 11 — Local OCI Registry + Worker Build Trigger
- Add `registry:2` service to `docker-compose.yml`
- DB migration: `worker_builds` table
- Backend: model, repo, service, domain, controller, routes
- New endpoints: `POST /workers/{id}/builds`, `GET /workers/{id}/builds`, `POST /workers/{id}/build-reports`
- Update `ARCHITECTURE.md`

### Phase 12 — Agent Card Build UI
- Agent card shows Build button when agent has active job
- Click opens mode picker: Fresh / Latest
- Agent card shows last build: status badge, image reference, built_at
- Frontend types, API helpers, component update

### Phase 13 — Deployment Plan Backend
- DB migration: `deployment_plans`, `deployment_plan_roles` tables
- Backend: model, repo, service, domain, controller, routes
- Service handles: plan creation, build mode resolution, per-plan docker-compose YAML generation, container lifecycle, health polling, nginx config generation + reload
- New endpoints: `POST /deployment-plans`, `GET /deployment-plans`, `GET /deployment-plans/{id}`, `POST /deployment-plans/{id}/stop`, `DELETE /deployment-plans/{id}`
- Update `ARCHITECTURE.md`

### Phase 14 — Deployments Frontend Page
- New route `/deploys`
- New Plan modal: name input, stack picker, mode picker, per-role agent picker
- Deployment plan card: status badge, preview URL, role rows, Stop / Delete buttons
- Add `Deploys` nav link to shared layout
- Frontend types, API helpers, components

### Phase 15 — Dynamic Nginx + Deploy Executor
- Backend generates `infra/nginx/preview-upstreams.conf` on deploy/stop
- `default.conf` updated to include generated file; `*.preview` server block delegates to upstreams
- Backend calls `docker exec proxy nginx -s reload` after config change
- Health check poller per plan until running or timeout
- Dnsmasq setup guide (optional, host-based)

---

## 16. Final Decision Summary

- Preview source of truth: local agent workspace state, not pre-approval Git history
- Artifact store: local OCI registry (`registry:2` container)
- Build unit: per-agent `WorkerBuild` (not scoped to a job/task)
- Deployment unit: `DeploymentPlan` (user-named, multi-role, simultaneous)
- Build mode: Fresh (force rebuild) or Latest (reuse last ready build)
- Routing in v1: subdomain `shiphide.{plan-name}.preview`, nginx config regenerated on each deploy/stop
- DNS in v1: hosts-file entries per plan name, wildcard DNS optional via host-based dnsmasq
- Build trigger entry point: agent card (only when agent has active job)
- Deploy trigger entry point: Deployments page → New Plan
