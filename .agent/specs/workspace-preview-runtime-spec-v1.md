# Workspace Preview Runtime Spec v1

## 1. Purpose
Define a deterministic runtime contract for manual preview environments built from agent workspaces before approval.

Goals:
- The user can review a preview without requiring a Git commit or Git push first.
- Preview artifacts are built from the agent's current local workspace state, including uncommitted changes.
- Frontend and backend artifacts stay isolated per preview request.
- The orchestrator can request, track, inspect, and destroy previews without repo-specific if/else logic.

Non-goals:
- Automatic preview creation on every task update.
- Pre-approval Git commits as a prerequisite for preview.
- Shared app runtime containers across previews.
- Production-grade multi-tenant preview hosting.

---

## 2. High-Level Model

### Shared Infrastructure
Shared across all previews on one host:
- Postgres instance
- MinIO instance
- Reverse proxy
- OCI image registry
- Optional log collector / metrics

### Preview Bundle
A preview request creates one preview bundle.

A preview bundle owns:
- `bundle_id`
- `stack_id`
- `task_id`
- one fresh image artifact per required role
- optional preselected builder worker per required role
- build status per role
- deploy status
- preview route

### Required Role Pairing
For the current web-app shape, one preview bundle contains:
- one frontend image
- one backend image

The orchestrator must not deploy until all required role images for the selected stack have been reported.

### Isolation Rule
Shared infra is allowed only for stateful platform services and artifact storage.
Application runtime must be isolated per preview bundle.

---

## 3. Core Terminology

### Agent
A worker that executes one task at a time and owns a local workspace checkout.

### Task
A single implementation or review job assigned to one agent.
`task_id` is the preview coordination identifier supplied to the preview API.

### Workspace State
The local agent workspace contents at preview-build time.
Workspace state may include uncommitted changes.

### Preview Bundle
A deployable preview unit assembled by the orchestrator from multiple role artifacts, identified by `bundle_id`.

### Build Manifest
Metadata recorded for each uploaded preview artifact so the orchestrator can trace which worker, role, and task produced it without relying on Git history.

### Stack Registry
A file-based registry stored in the orchestrator repo that defines which roles belong to a stack and how those roles map to worker groups and deployment services.

---

## 4. Design Principles

1. No pre-approval Git push
   - Agents must not be forced to commit or push code before preview.
   - Preview artifacts come from the current local workspace state.

2. Manual preview trigger
   - Preview creation happens only when explicitly requested from the frontend.
   - The system must not build preview artifacts automatically on every agent turn.

3. Fresh artifacts per preview request
   - Every preview bundle requires fresh artifacts for all required roles.
   - Reusing a previously approved image for one side is not allowed in v1.

4. Immutable deploy inputs
   - Deployment must use immutable image digests, not mutable tags.

5. File-based stack registry
   - Stack definitions are versioned with the orchestrator codebase.
   - Runtime stack resolution must not depend on ad hoc repo-specific logic.

6. Recoverable failures
   - One failed role artifact must not erase successful role artifacts already reported for the same bundle.

7. Destroyability
   - Every preview bundle must be disposable.
   - Cleanup must leave no orphan deployment resources or stale bundle state.

---

## 5. Stack Registry Contract

The orchestrator owns the stack registry.
It is stored as files in the orchestrator repo and loaded by the backend at startup.

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
- The frontend may present one worker picker per role, filtered by that role's `worker_group`.

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
- build output must be pushable to an OCI registry
- build output must include role-specific labels required by the stack registry

### Required Health Endpoints
- Frontend: returns HTTP 200 on preview route root or `/healthz`
- Backend: returns HTTP 200 on `/healthz`

### Required Env Variables
#### Preview Identity
- `BUNDLE_ID`
- `STACK_ID`
- `TASK_ID`
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
- `MEDIA_PREFIX` (optional if using per-preview bucket)
- `S3_ENDPOINT`
- `S3_ACCESS_KEY_ID`
- `S3_SECRET_ACCESS_KEY`

#### Frontend Runtime
- `PUBLIC_API_BASE_URL`
- `PUBLIC_BUNDLE_ID`
- `PUBLIC_PREVIEW_URL`

---

## 7. Shared Infrastructure Specification

### 7.1 OCI Registry
One OCI registry stores preview artifacts for all bundles.

Rules:
- every reported artifact must be addressable by immutable digest
- tags may exist for convenience, but deploy must resolve to digest
- retention policy must support short-lived preview artifacts and cleanup

### 7.2 Postgres
One Postgres instance may be shared by all previews.

Per-preview requirements:
- unique database name
- unique database user
- unique password

Rules:
- no shared DB user across previews
- orchestrator creates DB and credentials before backend deploy
- migrations run only against the preview bundle database

### 7.3 MinIO
One MinIO instance may be shared by all previews.

Preferred isolation:
- one bucket per preview bundle

Fallback isolation:
- shared bucket with strict prefix per preview bundle

Rules:
- a preview must not write outside its own bucket or prefix
- orchestrator provisions bucket or validates prefix policy before backend deploy

### 7.4 Reverse Proxy
One reverse proxy routes preview traffic.

#### v1 Route Strategy
Subdomain-based routing keyed by **agent (worker) name**, with hosts-file management on the devices that need access.

URL format:

```text
http://shiphide.{worker_name}.preview
```

Examples:

- `http://shiphide.alice.preview`
- `http://shiphide.bob.preview`

`{worker_name}` is the worker's `name` field, lowercased and slugified.

Rules:

- each agent has exactly one active preview URL at any time
- the URL is stable and reused across redeployments for the same agent
- when a new preview for an agent is deployed, the previous active bundle for that agent is automatically marked `destroyed` and its runtime is cleaned up after a short TTL
- multiple agents can have simultaneous active previews at distinct URLs
- v1 does not assume wildcard LAN DNS is available
- hosts-file entry required per device that needs access: `127.0.0.1 shiphide.{worker_name}.preview`
- this limitation must be documented, not hidden

---

## 8. Preview Bundle Lifecycle

### 8.1 Bundle States
- `pending_build`
- `building`
- `ready_to_deploy`
- `deploying`
- `healthy`
- `failed`
- `destroyed`

### 8.2 Role Build States
- `requested`
- `ready`
- `failed`

### 8.3 Transition Rules

- create preview request -> bundle `pending_build`; any previously `healthy` bundle for the same agent is auto-marked `destroyed` and scheduled for TTL cleanup
- first successful or failed role report -> bundle `building` unless terminal rules apply
- all required roles `ready` -> bundle `ready_to_deploy`
- any role `failed` -> bundle `failed`
- deployment start -> bundle `deploying`
- successful runtime health checks -> bundle `healthy`
- destroy request -> bundle `destroyed`

Successful role reports must remain recorded even if another role later fails.

### 8.4 Auto-Destroy on Redeploy
When the orchestrator deploys a new preview bundle for an agent, it must:

1. Find the current `healthy` bundle (if any) for that agent's worker URL.
2. Mark it `destroyed` immediately.
3. Schedule runtime cleanup (container teardown, proxy route removal) after a short TTL.
4. The old bundle record remains in the database for audit history — only its runtime is removed.

Only one bundle per agent may be in `healthy` state at a time.

---

## 9. Build and Deploy Flow

### 9.1 Manual Preview Request
User explicitly requests a preview with:
- `stack_id`
- `task_id`
- optional role-based builder overrides

The user-facing entry point is a frontend **Request Preview** action.
For the current product shape, the action is expected to live on the job detail page.

### 9.2 Orchestrator Steps
1. Resolve the stack from the stack registry.
2. Validate any supplied role override workers against the stack role `worker_group`.
3. Find any currently `healthy` bundle for the same agent and mark it `destroyed`; schedule TTL cleanup of its runtime.
4. Create preview bundle state with one requested role record per stack role.
5. Persist optional preselected worker assignment per role.
6. Ask the required agents to build their role artifacts.
7. Wait for worker build reports containing immutable digests.
8. When all required role digests are present, mark bundle `ready_to_deploy`.
9. Deploy runtime from exact digests at `http://shiphide.{worker_name}.preview`.
10. Run health checks.
11. Mark bundle `healthy` or `failed`.

### 9.3 Agent Build Rules
- agent builds from its own local workspace state
- agent must not be required to commit or push code first
- agent pushes the built artifact to the OCI registry
- agent reports immutable digest and build metadata back to the orchestrator

### 9.4 Approval Rule
Preview approval does not imply code push.
Git commit and push remain separate, post-approval actions.

---

## 10. Build Manifest Contract

Each successful or failed role build report creates one build manifest record.

### Required Manifest Fields
- `manifest_id`
- `bundle_id`
- `stack_id`
- `task_id`
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
- which preview bundle the artifact belongs to
- any builder-side labels or auxiliary references needed for audit/debug

---

## 11. API Contract

### User-Facing Preview Request API
Create a preview bundle from:
- `stack_id`
- `task_id`
- optional `role_overrides[]`

Each role override contains:
- `role`
- `worker_id`

Response must include:
- `bundle_id`
- `stack_id`
- `task_id`
- bundle status
- per-role build state
- selected worker per role when explicitly provided
- `preview_url` in format `http://shiphide.{worker_name}.preview` when bundle is `healthy`

### Worker Build Report API
Authenticated worker reports:
- `bundle_id`
- `role`
- `status`
- `image_reference`
- `image_digest`
- `metadata`
- `error_message` when failed

Rules:
- if a role override was supplied, only that selected worker may report for that role
- only workers in the configured role `worker_group` may report for that role
- reporting success for one role must not overwrite another role
- immutable digest is mandatory for successful reports

### Bundle Inspection API
Must support:
- bundle status
- per-role build state
- manifest history
- preview URL when assigned
- timestamps

### Bundle Destroy API
Must:
- mark bundle destroyed
- trigger runtime cleanup
- preserve inspectable historical state

---

## 12. Security Requirements

1. Agents building preview images require deliberate Docker/build capability.
2. Agents building preview images require registry credentials scoped for preview artifacts.
3. Registry credentials must not be broader than necessary.
4. Shared Postgres superuser credentials must not be exposed to app runtime containers.
5. Shared MinIO root credentials must not be exposed to app runtime containers unless unavoidable.
6. Preview hostnames should be treated as internal-only convenience endpoints in v1.

---

## 13. Failure Handling

### Role Build Failure
If a worker reports a failed role build:
- keep previously successful role manifests
- mark that role `failed`
- mark the bundle `failed`
- allow a later retry report for that same role

### Partial Success
If one role is ready and another is still pending or fails:
- successful role metadata remains visible
- bundle must stay inspectable

### Deploy Failure
If deploy fails after all artifacts are ready:
- keep recorded digests
- mark bundle `failed`
- do not silently rebuild artifacts

### Destroy Failure
If runtime cleanup only partially succeeds:
- bundle remains marked for destroy workflow
- cleanup must be retryable

---

## 14. Acceptance Criteria

1. User can request a preview without committing or pushing code first.
2. Preview bundle state is created from `stack_id` + `task_id`.
3. Preview bundle records one required role entry per stack role.
4. User may optionally preselect one builder worker per required role from the frontend request flow.
5. Invalid per-role builder selection is rejected when the worker does not belong to the role's configured worker group.
6. Workers can report build success or failure per role using authenticated callbacks.
7. Successful role reports persist even if another role later fails.
8. Bundle becomes `ready_to_deploy` only when all required roles report immutable digests.
9. Bundle inspection returns build state and manifest history.
10. Bundle destroy does not remove unrelated preview bundles.
11. Subdomain routing requirements and hosts-file limitation are explicitly documented.
12. No step in preview creation requires a Git commit or Git push.

---

## 15. Recommended Phase Plan

### Phase 1
- file-based stack registry
- preview bundle DB state
- worker build report API
- immutable digest tracking
- manual preview request API

### Phase 2
- actual build-request dispatch to workers
- deploy runtime from reported digests
- health checks and preview URL assignment
- TTL cleanup for old bundles and registry artifacts

### Phase 3
- wildcard DNS or a better local routing story
- dedicated builder service instead of agent-owned builds
- richer manifest audit data

---

## 16. Final Decision Summary
For the current system:
- Preview source of truth: local agent workspace state, not pre-approval Git history
- Artifact store: OCI registry
- Bundle shape: fresh frontend image + fresh backend image per preview request
- Trigger: manual user request only
- Routing in v1: subdomains managed through hosts-file entries
- Stack definition source: file-based stack registry in the orchestrator repo

This is the baseline contract the agent should implement.
