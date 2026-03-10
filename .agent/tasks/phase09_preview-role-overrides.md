# Task: Preview Bundle Role Overrides (Backend)

## Phase
phase09

## Status
done

## Plan Reference
`.agent/plans/plan_v4_preview_request_and_builder_selection.md`

## Goal
Extend `POST /api/v1/preview-bundles` to accept optional per-role worker overrides
so the user can pre-select one builder worker per required role at request time.
The orchestrator uses auto-pick when no override is provided.

---

## What Already Exists

- `domains/preview_bundle.go` — `CreatePreviewBundleRequest` has `stack_id` + `task_id` only
- `models/preview_bundle.go` — `PreviewBundleRole.AssignedWorkerID *string` already exists
- `repository/preview_bundle_repo.go` — `Create()` already writes `assigned_worker_id` to the DB; no repo change needed
- `services/preview_bundle_service.go` — `Create()` does not validate or apply role overrides yet
- `services/preview_bundle_service.go` — `ReportBuild()` validates worker group but does not enforce preselected worker lock

---

## Files to Change

- `backend/internal/domains/preview_bundle.go`
- `backend/internal/services/preview_bundle_service.go`
- `backend/internal/services/preview_bundle_service_test.go`

---

## Implementation Instructions

### 1. `domains/preview_bundle.go`

Add a new struct for a single role override:

```go
type RoleOverrideRequest struct {
    Role     string `json:"role"`
    WorkerID string `json:"worker_id"`
}
```

Add `RoleOverrides` field to `CreatePreviewBundleRequest`:

```go
type CreatePreviewBundleRequest struct {
    StackID       string                `json:"stack_id"`
    TaskID        string                `json:"task_id"`
    RoleOverrides []RoleOverrideRequest `json:"role_overrides,omitempty"`
}
```

### 2. `services/preview_bundle_service.go` — `Create()`

After resolving the stack and before creating bundle rows, validate role overrides:

**Validation rules (in order):**
1. For each override, `role` must exist in the selected stack.
2. Duplicate overrides for the same role are invalid — return an error.
3. For each override, `worker_id` must parse as a valid UUID.
4. Look up the worker by ID using `s.workerRepo.GetByID`. If not found, return an error.
5. The worker's `GroupName` must match the stack role's `WorkerGroup`. If not, return an error.

**After validation passes:**
- Build a map of `role → *string workerID` from the validated overrides.
- When creating `PreviewBundleRole` rows, set `AssignedWorkerID` from the map if an override exists for that role. Leave `nil` for roles without an override.

### 3. `services/preview_bundle_service.go` — `ReportBuild()`

After confirming the worker belongs to the correct group, add a preselected worker check:

- If `roleRow.AssignedWorkerID != nil`, the reporting `workerID` must match `*roleRow.AssignedWorkerID`.
- If it does not match, return an error: `"role %q is reserved for a different worker"`.

---

## Validation Rules Summary

| Rule | Error |
|------|-------|
| role not in stack | `"role %q is not valid for stack %q"` |
| duplicate role override | `"duplicate role override for role %q"` |
| worker_id not valid UUID | `"invalid worker_id for role %q"` — `%q` is the **role** name |
| worker not found | `"worker not found for role override %q"` — `%q` is the **role** name |
| worker group mismatch | `"worker group %q cannot be assigned to role %q (expected group %q)"` |
| preselected worker mismatch at report | `"role %q is reserved for a different worker"` |

---

## Test Cases (existing test file: `services/preview_bundle_service_test.go`)

Add tests for:
- create with no overrides — succeeds, `assigned_worker_id` is nil on all roles
- create with valid per-role override — succeeds, correct worker assigned to role
- create with role not in stack — rejected
- create with duplicate role override — rejected
- create with `worker_id` that is not a valid UUID — rejected
- create with `worker_id` that is a valid UUID but does not exist in the DB — rejected
- create with worker from wrong group — rejected
- `ReportBuild` by non-preselected worker when role has preselected worker — rejected

---

## Notes
- No migration needed: `assigned_worker_id` column already exists.
- No repository change needed: `repo.Create()` already writes `assigned_worker_id`.
- `repo.RecordBuildManifest()` overwrites `assigned_worker_id` on the role row when a build report comes in — this is intentional; the reported worker becomes the actual builder.
- Do not change the response shape; `PreviewBundleRoleResponse.AssignedWorkerID` already exists.

## Completed At

2026-03-10
