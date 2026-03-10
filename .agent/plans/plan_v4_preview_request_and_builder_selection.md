# Plan v4 — Preview Request UI and Per-Role Builder Selection

## Summary
The current preview-bundle backend is only a foundation. The user-facing preview feature is not complete because there is no frontend entrypoint, no builder selection UI, and no backend create contract for explicit role-based builder assignment.

This plan defines the next coherent slice:
- add a frontend **Request Preview** action on the job detail page
- let the user choose one builder worker per required stack role
- extend the preview create API to accept per-role worker overrides
- keep orchestrator auto-pick as the fallback when no override is provided

## Product Behavior

### Entry Point
- Add a **Request Preview** button on the job detail page.
- The button opens a modal, not a separate page.

### Request Preview Modal
- Show stack selector.
- Show one worker selector per required role returned by the selected stack.
- Current web-app stack roles are `backend` and `frontend`.
- Each role selector is filtered to workers in that role's configured `worker_group`.
- Each role selector supports:
  - explicit worker selection
  - auto-pick / no override

### Request Submission
- Frontend sends `stack_id`, `task_id`, and optional per-role worker overrides.
- `task_id` remains the preview coordination identifier for now.
- Successful create navigates or refreshes into preview bundle detail/status view.

### Status Visibility
- The job detail page should show preview bundle state for the current job/task when available.
- Minimum visible fields:
  - bundle status
  - role build status
  - assigned worker per role
  - preview URL when bundle is `healthy` — format: `http://shiphide.{worker_name}.preview`

### Preview URL Routing

- Each agent has exactly one active preview URL at any time: `http://shiphide.{worker_name}.preview`
- When a new preview is deployed for the same agent, the previous bundle is auto-marked `destroyed` and its runtime cleaned up after a short TTL.
- Multiple agents may have simultaneous active previews at distinct URLs.
- Access requires a hosts-file entry on the viewing device: `127.0.0.1 shiphide.{worker_name}.preview`

## Backend Contract Changes

### Preview Create Request
Add per-role builder overrides to `POST /api/v1/preview-bundles`.

Expected shape:
- `stack_id`
- `task_id`
- `role_overrides[]`

Per override:
- `role`
- `worker_id`

### Validation Rules
- `role` must exist in the selected stack.
- selected `worker_id` must exist.
- selected worker must belong to that role's configured `worker_group`.
- duplicate overrides for the same role are invalid.
- omitted role override means orchestrator auto-pick later.

### Persistence
- preview bundle role rows store optional preselected worker assignment before build starts
- build report validation still enforces role/group match

## Implementation Changes

### Frontend
- Add preview API client module.
- Add preview request modal component.
- Add job-page integration and status panel.
- Reuse existing NEXUS modal/form patterns.

### Backend
- Extend preview bundle domain types, service validation, repository persistence, and response payloads for role overrides.
- Keep actual build dispatch and deploy execution out of this slice unless a real orchestrator path is added in the same task.

### Docs
- Update workspace preview runtime spec to define frontend request flow and per-role worker selection.
- Update main product spec to place the button on the job detail page.
- Update architecture doc for API shape and persisted role override behavior.

## Test Plan
- create preview bundle with no overrides
- create preview bundle with valid per-role worker overrides
- reject override when worker belongs to wrong group
- reject duplicate role overrides
- frontend modal lists only valid workers per role
- created preview bundle response includes selected worker per role

## Assumptions
- Job detail page is the primary preview entrypoint.
- One preview request chooses one builder per required role, not one generic builder.
- `task_id` continues to be supplied explicitly and is not yet replaced by a dedicated task entity.
