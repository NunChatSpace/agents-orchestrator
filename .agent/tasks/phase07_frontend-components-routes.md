# Task: Frontend Components & Routes

## Phase
phase07

## Status
done

## Completed At
2026-03-06

## Files Created / Modified
- frontend/src/components/atoms/Badge.svelte
- frontend/src/components/atoms/Button.svelte
- frontend/src/components/atoms/Spinner.svelte
- frontend/src/components/molecules/StatusBadge.svelte
- frontend/src/components/molecules/JobListItem.svelte
- frontend/src/components/molecules/MessageBubble.svelte
- frontend/src/components/organisms/Sidebar.svelte
- frontend/src/components/organisms/TopBar.svelte
- frontend/src/components/organisms/MessageFeed.svelte
- frontend/src/components/organisms/Composer.svelte
- frontend/src/components/organisms/NewJobForm.svelte
- frontend/src/routes/+layout.svelte
- frontend/src/routes/+page.svelte
- frontend/src/routes/login/+page.svelte
- frontend/src/routes/(app)/+layout.svelte
- frontend/src/routes/(app)/+page.svelte
- frontend/src/routes/(app)/jobs/new/+page.svelte
- frontend/src/routes/(app)/jobs/[job_id]/+page.svelte

## Notes
- Root layout: calls getMe() on mount; redirects to /login on auth failure; connects WS
- (app)/layout: loads jobs + workers in parallel into stores; renders Sidebar + main slot
- Job detail: wires TopBar cancel/close events, Composer send event; uses composerEnabled derived store
- NewJobForm: dispatches 'draft' and 'submit' events; page handles createJob + optional submitJob
- API calls return T directly (client.ts unwraps body.data) — no .data property access needed
