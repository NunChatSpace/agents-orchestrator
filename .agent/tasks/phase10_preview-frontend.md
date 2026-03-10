# Task: Preview Request Modal + Status Panel (Frontend)

## Phase

phase10

## Status

pending

## Plan Reference

`.agent/plans/plan_v4_preview_request_and_builder_selection.md`

## Goal

Add a "Request Preview" entry point on the job detail page.
The user picks a stack and optionally one builder worker per required role,
then submits. The job detail page shows the preview bundle status inline.

---

## What Already Exists

- `routes/(app)/jobs/[job_id]/+page.svelte` — job detail page (chat feed); add button + modal + status panel here
- `lib/apis/` — API client modules (add `preview.ts` here)
- `stores/workers.ts` — `allWorkers` store with all registered workers
- No preview types or API client exist yet

---

## Files to Create

- `frontend/src/lib/apis/preview.ts` — API client for preview endpoints
- `frontend/src/types/preview.ts` — TypeScript types for preview bundle
- `frontend/src/components/organisms/RequestPreviewModal.svelte` — Request preview modal
- `frontend/src/components/molecules/PreviewStatusPanel.svelte` — Inline status panel

## Files to Modify

- `frontend/src/routes/(app)/jobs/[job_id]/+page.svelte` — Add "Request Preview" button, modal, and status panel

---

## Implementation Instructions

### 1. `frontend/src/types/preview.ts`

Define TypeScript interfaces matching the backend response shapes:

```ts
export interface PreviewStackRole {
  role: string;
  worker_group: string;
  service_name: string;
  healthcheck_path: string;
  required_image_labels: Record<string, string>;
}

export interface PreviewStack {
  stack_id: string;
  display_name: string;
  deployment_template: string;
  roles: PreviewStackRole[];
}

export interface PreviewBundleRoleState {
  id: string;
  role: string;
  worker_group: string;
  assigned_worker_id?: string;   // omitempty — absent when no override
  build_status: 'requested' | 'ready' | 'failed';
  latest_manifest_id?: string;   // omitempty — absent until first report
  image_reference?: string;      // omitempty — absent until ready
  image_digest?: string;         // omitempty — absent until ready
  error_message?: string;        // omitempty — absent unless failed
  created_at: string;
  updated_at: string;
}

export interface PreviewBuildManifest {
  id: string;
  role: string;
  worker_id: string;
  status: 'requested' | 'ready' | 'failed';
  image_reference: string;       // always present in manifest
  image_digest?: string;         // omitempty — absent unless ready
  metadata: Record<string, string>;
  error_message?: string;        // omitempty — absent unless failed
  created_at: string;
}

export interface PreviewBundle {
  id: string;
  stack_id: string;
  task_id: string;
  status: 'pending_build' | 'building' | 'ready_to_deploy' | 'deploying' | 'healthy' | 'failed' | 'destroyed';
  preview_url?: string;          // omitempty — absent until healthy
  created_at: string;
  updated_at: string;
  destroyed_at?: string;         // omitempty — absent unless destroyed
  roles: PreviewBundleRoleState[];
  manifests?: PreviewBuildManifest[];  // omitempty — absent when empty
}

// NOTE: The API client (lib/apis/client.ts) returns body.data as-is.
// Omitted fields arrive as undefined, not null.
// Use optional chaining (?.) and nullish coalescing (??) when accessing optional fields.

export interface RoleOverride {
  role: string;
  worker_id: string;
}

export interface CreatePreviewBundleRequest {
  stack_id: string;
  task_id: string;
  role_overrides?: RoleOverride[];
}
```

### 2. `frontend/src/lib/apis/preview.ts`

Use the existing `GET` / `POST` pattern from `lib/apis/client.ts` (the file is `client.ts`, not `fetcher.ts`):

```ts
import { GET, POST } from './client';
import type { PreviewStack, PreviewBundle, CreatePreviewBundleRequest } from '../../types/preview';

// client.ts already prefixes all paths with /api/v1 — do not include /v1 here
export const listPreviewStacks = () =>
  GET<PreviewStack[]>('/preview-stacks');

export const listPreviewBundles = () =>
  GET<PreviewBundle[]>('/preview-bundles');

export const createPreviewBundle = (body: CreatePreviewBundleRequest) =>
  POST<PreviewBundle>('/preview-bundles', body);

export const destroyPreviewBundle = (bundleId: string) =>
  POST<PreviewBundle>(`/preview-bundles/${bundleId}/destroy`, {});
```

### 3. `RequestPreviewModal.svelte` (Organism)

Props:

- `jobId: string` — used as `task_id` in the request body
- `open: boolean` — controls modal visibility
- `on:close` event — emitted when user cancels or after successful submit
- `on:created` event — emitted with the new `PreviewBundle` on success

Modal behavior:

1. On open, fetch `GET /api/v1/preview-stacks`. Show a loading state.
2. Show a stack selector (only one stack for now, but render as a `<select>` for extensibility).
3. After stack is selected, show one worker selector per required role from the selected stack.
4. Each worker selector:
   - Label: role name (e.g. "Backend Builder", "Frontend Builder")
   - Options: workers from `allWorkers` store where `worker.group_name === role.worker_group`
   - First option: "Auto-pick" (no override, value = `""`)
   - Remaining options: matching worker names + IDs
5. Submit button calls `POST /api/v1/preview-bundles` with `stack_id`, `task_id = jobId`, and `role_overrides` for non-empty selections.
6. On success, emit `on:created` with the response bundle.
7. On error, show error message inline — do not close the modal.

Use the existing `Modal` and `Button` atoms:

- Wrap content in `<Modal on:close={...}>` (`components/atoms/Modal.svelte`) — it provides the backdrop, `nx-card` container, and close button.
- Use `<Button variant="primary|secondary|danger|ghost">` (`components/atoms/Button.svelte`) for all interactive buttons.
- Use `.nx-card` and `.nx-input` CSS classes for inner layout elements where needed.

### 4. `PreviewStatusPanel.svelte` (Molecule)

Props:

- `bundle: PreviewBundle`

Events:

- `on:destroyed` — emitted with no payload when destroy succeeds; the page calls `loadPreviewBundle(jobId)` on receipt to re-evaluate which bundle (if any) should now be shown

Render (minimal):

- Bundle status badge
- Per-role row: role name | build status badge | assigned worker ID (or "auto") | image digest (truncated, if present) | error message (if present)
- Preview URL link when `bundle.status === 'healthy'` and `bundle.preview_url` is set — open in new tab
- "Destroy" button — rendered only when `bundle.status !== 'destroyed'`; calls `destroyPreviewBundle(bundle.id)`; on success emits `on:destroyed` with no payload; the destroy response is not forwarded to the caller.

### 5. `routes/(app)/jobs/[job_id]/+page.svelte`

Changes:

1. Import `RequestPreviewModal`, `PreviewStatusPanel`, `listPreviewBundles`.
   - `createPreviewBundle` is owned by `RequestPreviewModal` — do not import it on the page.
   - `getPreviewBundle` is not needed — the page clears `activeBundle` on destroy and does not inspect the response.
2. Add state: `let previewModalOpen = false`, `let activeBundle: PreviewBundle | undefined = undefined`.
3. Bundle restore must be reactive, not `onMount`-only. The page already handles job changes with
   `$: if (jobId && jobId !== currentJobId) { currentJobId = jobId; load(jobId); }`.
   Add a separate `loadPreviewBundle(id)` async function — do **not** inline this into the existing
   `load()` try/catch. The existing `load()` uses a single try/catch to control the job page error
   state; a failed preview fetch must not put the whole page into the error state.

   ```ts
   async function loadPreviewBundle(id: string) {
     activeBundle = undefined;
     try {
       const bundles = await listPreviewBundles();
       // list is already sorted updated_at DESC by the backend — first match is most recent
       activeBundle = bundles
         .find(b => b.task_id === id && b.status !== 'destroyed');
     } catch {
       // best-effort — preview panel is optional, do not surface this error
     }
   }
   ```

   Call `loadPreviewBundle(id)` from the reactive block alongside `load(id)`:
   `$: if (jobId && jobId !== currentJobId) { currentJobId = jobId; load(jobId); loadPreviewBundle(jobId); }`

   `jobId` is derived from `$page.params.job_id`. Use `jobId`, not `$activeJob.job_id` or `job.id`.

4. Add a "Request Preview" button below `<TopBar>` in the page template — **do not modify TopBar.svelte**,
   which has no slot or extension point. Add a thin strip directly after the `<TopBar>` tag in the page
   template. Use `<Button variant="secondary">Request Preview</Button>` from `components/atoms/Button.svelte`.
5. Mount `<RequestPreviewModal bind:open={previewModalOpen} jobId={jobId} on:created={handleBundleCreated} />`.
6. `handleBundleCreated`: set `activeBundle` from the event detail, close modal.
7. When `activeBundle` is not undefined, render `<PreviewStatusPanel bundle={activeBundle} on:destroyed={handleBundleDestroyed} />` below the preview strip.
8. `handleBundleDestroyed`: call `loadPreviewBundle(jobId)` (do not just clear `activeBundle`).
   This is the same logic as a page reload: find the most recent non-destroyed bundle for this job.
   If an older non-destroyed bundle exists it will surface; if none exists `activeBundle` becomes
   undefined and the panel disappears. Both in-session destroy and reload produce the same result.
9. Do not remove or restructure the existing chat feed — append the new elements above or alongside it.

---

## Design Rules

- The only utility CSS classes defined in `app.css` are `.nx-card`, `.nx-input`, and `.nx-label`. There is no `.nx-btn` or `.nx-badge` class — use the Svelte atoms instead:
  - Buttons: `<Button variant="primary|secondary|danger|ghost">` from `components/atoms/Button.svelte`
  - Badges: `<Badge color="gray|blue|yellow|green|red|purple|orange">` from `components/atoms/Badge.svelte`
- Do **not** reuse `StatusBadge.svelte` — it only maps job statuses. Use `Badge` directly with inline color maps. Required mappings:
  - **Bundle status** (`bundle.status`): `pending_build` → gray, `building` → yellow, `ready_to_deploy` → blue, `deploying` → purple, `healthy` → green, `failed` → red, `destroyed` → gray.
  - **Role build status** (`role.build_status`): `requested` → gray, `ready` → green, `failed` → red.
- Do not introduce new colors, fonts, or CSS variables outside the existing system.

---

## Notes

- `task_id` in the preview API is the `job_id` from the URL — no separate task entity exists yet.
- `allWorkers` store is already populated via WS + REST on app load; no extra fetch needed for worker list.
- If no workers match a role's worker group, show a note like "No workers available — auto-pick will be used" for that role's selector, but do **not** disable submit. The backend creates the bundle regardless; unmatched roles are simply left without an override and the backend assigns a worker later.
- The status panel does not need to poll — a manual refresh or re-fetch on demand is sufficient for this slice.
