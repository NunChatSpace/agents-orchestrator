# Phase 12 — Agent Card Build UI

## Goal

Add a Build button to each agent card on the Agents page. The button is visible only when the agent has an active job. Clicking it opens a mode picker (Fresh / Latest). The card shows the agent's last build status and image info.

After this phase:
- Each agent card with an active job shows a **Build** button.
- Clicking Build opens an inline mode picker: Fresh / Latest.
- Confirming triggers `POST /api/v1/workers/{id}/builds`.
- The card shows the most recent build: status badge, image reference (truncated), time ago.

---

## Spec Reference

`.agent/specs/workspace-preview-runtime-spec-v1.md` §9.1, §12

---

## Prerequisites

Phase 11 must be complete (`/workers/{id}/builds` endpoints exist).

---

## Affected Files

| File | Change |
|---|---|
| `frontend/src/types/workerBuild.ts` | New TypeScript types |
| `frontend/src/lib/apis/workerBuilds.ts` | New API helpers |
| `frontend/src/components/molecules/WorkerBuildStatus.svelte` | New — shows last build inline on card |
| `frontend/src/routes/(app)/+page.svelte` | Load builds per agent; pass to card |
| `frontend/src/components/organisms/WorkerCard.svelte` (or wherever worker cards render) | Add Build button + mode picker |

Identify the exact agent card component file before editing. Check `routes/(app)/+page.svelte` and any `WorkerCard` / `AgentCard` component.

---

## Types: `frontend/src/types/workerBuild.ts`

```typescript
export type WorkerBuildStatus = 'queued' | 'building' | 'ready' | 'failed';

export interface WorkerBuild {
  id: string;
  worker_id: string;
  stack_id: string;
  role: string;
  build_mode: 'fresh' | 'latest';
  status: WorkerBuildStatus;
  image_reference?: string;
  image_digest?: string;
  error_message?: string;
  created_at: string;
  completed_at?: string;
}

export interface TriggerBuildRequest {
  stack_id: string;
  role: string;
  mode: 'fresh' | 'latest';
}
```

---

## API Helpers: `frontend/src/lib/apis/workerBuilds.ts`

```typescript
import { GET, POST } from './client';
import type { WorkerBuild, TriggerBuildRequest } from '../../types/workerBuild';

export const triggerBuild = (workerId: string, body: TriggerBuildRequest) =>
  POST<WorkerBuild>(`/v1/workers/${workerId}/builds`, body);

export const listWorkerBuilds = (workerId: string) =>
  GET<WorkerBuild[]>(`/v1/workers/${workerId}/builds`);
```

---

## WorkerBuildStatus Component: `frontend/src/components/molecules/WorkerBuildStatus.svelte`

Props:
- `build: WorkerBuild | undefined` — the most recent build for this worker

Renders:
- If `build` is undefined: nothing (or a subtle "No builds yet" dim text)
- If `build` exists: status badge + truncated image reference + relative time

Badge color map:

| status | color |
|---|---|
| queued | gray |
| building | yellow |
| ready | green |
| failed | red |

Image reference display: show last 30 chars of `image_reference` if longer (e.g. `…backend:20241201-120000`). Full value in `title` attribute for hover.

Use `Badge` atom. No new colors.

---

## Agents Page Changes: `frontend/src/routes/(app)/+page.svelte`

On load, for each worker that has an active job (status `assigned`, `busy`, or `pending_user`), call `listWorkerBuilds(worker.worker_id)` and store the most recent build.

```typescript
// Key: worker_id → most recent WorkerBuild or undefined
let latestBuilds: Record<string, WorkerBuild | undefined> = {};

async function loadBuildsForActiveWorkers(workers: Worker[]) {
  const activeWorkers = workers.filter(w =>
    ['assigned', 'busy', 'pending_user'].includes(w.status)
  );
  const results = await Promise.allSettled(
    activeWorkers.map(w => listWorkerBuilds(w.worker_id))
  );
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value.length > 0) {
      latestBuilds[activeWorkers[i].worker_id] = r.value[0]; // sorted DESC by created_at
    }
  });
}
```

Call `loadBuildsForActiveWorkers` after the worker list is fetched.

---

## Build Button + Mode Picker

Within the agent card component (identify exact file first), add:

Visibility rule: show Build section only when `worker.status` is `assigned`, `busy`, or `pending_user`.

UI layout (below the agent status area, above the card footer):

```
[ WorkerBuildStatus ] ← shows last build inline
[ Build ▼ ]           ← button, opens mode picker
```

Mode picker (inline, not a modal):

```
Fresh   — force a new build from current workspace
Latest  — use the most recent ready build
[ Cancel ]  [ Confirm ]
```

On confirm:
1. Set `building = true` on that card.
2. Call `triggerBuild(worker.worker_id, { stack_id, role, mode })`.
   - `stack_id` and `role` must be derived from the worker's `group_name` or a hardcoded mapping. For phase 12, use `stack_id = 'fi-web-app'` and derive `role` from `worker.group_name` (e.g. `fi-backend` → `backend`, `fi-frontend` → `frontend`). Document this mapping assumption.
3. On success: update `latestBuilds[worker_id]` with the returned build.
4. On error: show an inline error message on the card.
5. Set `building = false`.

---

## Stack / Role Mapping Assumption

Phase 12 uses a hardcoded mapping from `worker_group` to `role` and `stack_id`. This is a placeholder until phase 13 introduces the deployment plan UI with explicit stack selection.

Example mapping (document this in code as a comment):

```typescript
const WORKER_GROUP_TO_ROLE: Record<string, { stack_id: string; role: string }> = {
  'fi-backend':  { stack_id: 'fi-web-app', role: 'backend' },
  'fi-frontend': { stack_id: 'fi-web-app', role: 'frontend' },
};
```

If a worker's group is not in the map, show "Build not supported for this worker group" and disable the button.

---

## NEXUS Design Rules

- Use `Badge` atom for build status. Color map above.
- Use `Button` atom for the Build button (`variant="secondary"`).
- Mode picker is an inline div, not a modal. Use `.nx-card` styling or a `border border-purple-500/20 rounded-xl p-3 bg-purple-900/10` surface.
- No new colors, fonts, or component patterns outside the existing system.

---

## Acceptance Criteria

1. Agent card shows Build button only when agent status is `assigned`, `busy`, or `pending_user`.
2. Agent card does not show Build button for idle or offline agents.
3. Clicking Build reveals the mode picker inline.
4. Selecting Fresh and confirming calls `POST /workers/{id}/builds` with `mode=fresh`.
5. Selecting Latest and confirming calls `POST /workers/{id}/builds` with `mode=latest`.
6. After trigger, the card shows the new build with `queued` status badge.
7. If the agent's worker group has no stack/role mapping, the button is disabled with a tooltip.
8. API errors show an inline message on the card; button re-enables.
