# Phase 14 — Deployment Plan Frontend

## Goal

Add a Deployments page (`/deploys`) where users can create deployment plans, view running and past plans, and stop or delete them.

After this phase:
- Nav shows "Deploys" link.
- `/deploys` page lists all plans with status badges and preview URLs.
- "New Plan" modal lets the user name the plan, pick a stack, pick a build mode, and assign one agent per role.
- Running plans show a clickable preview URL.
- Stop and Delete buttons are present per plan.

---

## Spec Reference

`.agent/specs/workspace-preview-runtime-spec-v1.md` §8.2, §9.3, §11, §14

---

## Prerequisites

Phase 13 must be complete (deployment plan endpoints exist).

---

## Affected Files

| File | Change |
|---|---|
| `frontend/src/types/deploymentPlan.ts` | New TypeScript types |
| `frontend/src/lib/apis/deploymentPlans.ts` | New API helpers |
| `frontend/src/routes/(app)/deploys/+page.svelte` | New route |
| `frontend/src/components/organisms/CreateDeployPlanModal.svelte` | New modal |
| `frontend/src/components/molecules/DeployPlanCard.svelte` | New card component |
| `frontend/src/routes/(app)/+layout.svelte` | Add "Deploys" nav link |

---

## Types: `frontend/src/types/deploymentPlan.ts`

```typescript
export type DeploymentPlanStatus = 'pending' | 'deploying' | 'running' | 'failed' | 'stopped';

export interface DeploymentPlanRole {
  id: string;
  role: string;
  worker_id: string;
  worker_build_id?: string;
  image_reference?: string;
  image_digest?: string;
  container_name?: string;
  build_status: 'pending' | 'ready' | 'failed';
  deploy_status: 'pending' | 'running' | 'failed';
}

export interface DeploymentPlan {
  id: string;
  name: string;
  stack_id: string;
  build_mode: 'fresh' | 'latest';
  status: DeploymentPlanStatus;
  preview_url?: string;
  error?: string;
  roles: DeploymentPlanRole[];
  created_at: string;
  updated_at: string;
}

export interface CreateDeploymentPlanRequest {
  name: string;
  stack_id: string;
  build_mode: 'fresh' | 'latest';
  roles: { role: string; worker_id: string }[];
}
```

---

## API Helpers: `frontend/src/lib/apis/deploymentPlans.ts`

```typescript
import { GET, POST, DELETE } from './client';
import type { DeploymentPlan, CreateDeploymentPlanRequest } from '../../types/deploymentPlan';

export const listDeploymentPlans = () =>
  GET<DeploymentPlan[]>('/v1/deployment-plans');

export const createDeploymentPlan = (body: CreateDeploymentPlanRequest) =>
  POST<DeploymentPlan>('/v1/deployment-plans', body);

export const getDeploymentPlan = (id: string) =>
  GET<DeploymentPlan>(`/v1/deployment-plans/${id}`);

export const stopDeploymentPlan = (id: string) =>
  POST<DeploymentPlan>(`/v1/deployment-plans/${id}/stop`, {});

export const deleteDeploymentPlan = (id: string) =>
  DELETE<void>(`/v1/deployment-plans/${id}`);
```

Note: add `DELETE` helper to `frontend/src/lib/apis/client.ts` if it does not already exist.

---

## Deploys Page: `frontend/src/routes/(app)/deploys/+page.svelte`

State:
- `plans: DeploymentPlan[]` — loaded on mount
- `createModalOpen: boolean`
- `loading: boolean`
- `error: string`

On mount: call `listDeploymentPlans()`.

Layout:
```
[Page header: "Deployments"]  [New Plan button]
[DeployPlanCard per plan, sorted: running first, then pending/deploying, then stopped/failed]
[Empty state if no plans: "No deployment plans yet."]
```

Polling: poll `listDeploymentPlans()` every 5 seconds while any plan has status `pending` or `deploying`. Stop polling when all plans are `running`, `failed`, or `stopped`.

Event handlers:
- `handleCreated(plan)`: prepend plan to list, close modal.
- `handleStop(id)`: call `stopDeploymentPlan(id)`, update list entry.
- `handleDelete(id)`: call `deleteDeploymentPlan(id)`, remove from list.

---

## New Plan Modal: `frontend/src/components/organisms/CreateDeployPlanModal.svelte`

Uses `Modal` atom. Props: `open: boolean`. Events: `created: DeploymentPlan`.

Fields:

1. **Plan name** — text input, placeholder `e.g. feature-cart`, validation `[a-z0-9-]{1,40}`, show inline error if invalid.
2. **Stack** — dropdown populated from `GET /api/v1/preview-stacks`. Shows `display_name`. Required.
3. **Build mode** — radio/toggle: `Fresh` (force rebuild) | `Latest` (reuse last ready build).
4. **Roles section** — rendered after stack is selected. One agent picker per role in the stack. Each picker:
   - Label: role name (e.g. `backend`, `frontend`)
   - Options: workers filtered by the role's `worker_group`, loaded from existing `allWorkers` store
   - Shows worker name + current status badge
   - Required: all roles must be assigned before submit

Submit button: disabled while any required field is empty or while submitting.

On submit:
1. Set `submitting = true`.
2. Call `createDeploymentPlan(...)`.
3. On success: emit `created` event with returned plan. Reset form.
4. On error: show error inline.
5. Set `submitting = false`.

Race condition guard: use a token pattern (same as `RequestPreviewModal`). If modal is closed while submitting, ignore the response.

---

## Deploy Plan Card: `frontend/src/components/molecules/DeployPlanCard.svelte`

Props: `plan: DeploymentPlan`. Events: `stop`, `delete`.

Layout:

```
┌─────────────────────────────────────────────────────┐
│ [plan name]                    [status badge]        │
│ stack: {stack_id}  mode: {build_mode}                │
│                                                      │
│ Preview URL: http://shiphide.{name}.preview  [open↗] │
│ (shown only when status=running)                     │
│                                                      │
│ Roles:                                               │
│  backend  [build badge] [deploy badge]  worker-name  │
│  frontend [build badge] [deploy badge]  worker-name  │
│                                                      │
│ [error message if status=failed]                     │
│                                                      │
│ [Stop]  [Delete]                                     │
└─────────────────────────────────────────────────────┘
```

Plan status badge colors:

| status | color |
|---|---|
| pending | gray |
| deploying | yellow |
| running | green |
| failed | red |
| stopped | gray |

Role build status badge colors:

| status | color |
|---|---|
| pending | gray |
| ready | green |
| failed | red |

Role deploy status badge colors:

| status | color |
|---|---|
| pending | gray |
| running | green |
| failed | red |

Buttons:
- `[Stop]`: visible when status is `pending`, `deploying`, or `running`. Calls `on:stop`. Confirms before calling.
- `[Delete]`: visible when status is `stopped` or `failed`. Calls `on:delete`. Confirms before calling.

Use `Button` atom: Stop → `variant="danger"`, Delete → `variant="secondary"`.

---

## Nav Link: `frontend/src/routes/(app)/+layout.svelte`

Add "Deploys" to the existing nav alongside "Agents", "Plans", "Office":

```svelte
<a href="/deploys" class="...">Deploys</a>
```

Match the styling of existing nav links.

---

## NEXUS Design Rules

- Use `Badge` atom for all status badges.
- Use `Button` atom for all buttons.
- Use `Modal` atom for create modal.
- Use `.nx-card` for the plan card wrapper.
- Use `.nx-input` for the plan name text input.
- No new colors outside the existing color map.

---

## Acceptance Criteria

1. Nav shows "Deploys" link; clicking navigates to `/deploys`.
2. Deploys page shows all plans for the current user.
3. "New Plan" button opens the create modal.
4. Stack selection populates the role picker section.
5. Each role picker shows only workers from the correct `worker_group`.
6. Submit is disabled until all required fields are filled.
7. Successful plan creation closes modal and shows the new plan card.
8. Running plans show a clickable preview URL link.
9. Stop button is visible for active plans; Delete button is visible for stopped/failed plans.
10. Page polls while plans are in `pending` or `deploying` state and stops polling when all are terminal.
11. Plan name with invalid characters shows inline validation error before submit.
