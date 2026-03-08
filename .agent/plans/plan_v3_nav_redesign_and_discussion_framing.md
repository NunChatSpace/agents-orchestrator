# Plan: Navigation Redesign + Discussion Prompt Framing

## Context

Two improvements approved after discussion:

1. **Discussion prompt framing** — prepend a scoping instruction to every plan session message turn so the worker agent stays concise and focused instead of running a full reasoning loop.
2. **Navigation redesign** — replace the current sidebar + home page tab layout with a persistent top bar (logo, workspace dropdown, Agents | Plans nav) and dedicated pages for each section.

---

## Change 1 — Backend: Discussion prompt framing

**File:** `backend/internal/services/plan_session_service.go`

In `SendMessage`, prepend the following before the user's content when calling `RunPlanStream`:

```
You are in a task scoping conversation. Your role is to ask focused clarifying questions to help the user define a clear, well-scoped task. Keep responses short and conversational — 2-4 sentences max. Do not execute any code or make file changes. If the user's goal seems too broad or vague, say so briefly and suggest a narrower starting point.

User: {content}
```

Applied to `/message` turns only. `/generate` already has its own instruction and is unchanged.

Hardcoded in service — not configurable in v1.

---

## Change 2 — Frontend: Navigation redesign

### New route structure

| Route | File | Purpose |
|---|---|---|
| `/` | `routes/(app)/+page.svelte` | Agents page (agent cards, filtered by workspace) |
| `/agents/[worker_id]/jobs` | `routes/(app)/agents/[worker_id]/jobs/+page.svelte` | Job list for this agent |
| `/agents/[worker_id]/settings` | `routes/(app)/agents/[worker_id]/settings/+page.svelte` | Agent settings (current `/agents/[worker_id]` content) |
| `/plans` | `routes/(app)/plans/+page.svelte` | Plans page (plan sessions, as-is) |
| `/jobs/[job_id]` | `routes/(app)/jobs/[job_id]/+page.svelte` | Job chat (unchanged) |

### Files to change

| File | Change |
|---|---|
| `routes/(app)/+layout.svelte` | Add persistent top bar: logo, workspace dropdown, Agents \| Plans nav links. Remove sidebar. |
| `routes/(app)/+page.svelte` | Simplify to agents-only page (remove Plan tab, remove sidebar, keep agent grid) |
| `routes/(app)/agents/[worker_id]/+page.svelte` | **Delete** (content moves to `/settings`) |
| `routes/(app)/agents/[worker_id]/settings/+page.svelte` | **New** — move current agent settings content here |
| `routes/(app)/agents/[worker_id]/jobs/+page.svelte` | **New** — job list for this agent, click → `/jobs/{id}` |
| `routes/(app)/plans/+page.svelte` | **New** — move plan sessions UI here from old home Plan tab |
| `routes/(app)/jobs/[job_id]/+page.svelte` | Remove sidebar reference if any |
| `stores/selectedGroup.ts` | No change — workspace dropdown reads from this store |

### Top bar layout

```
┌─────────────────────────────────────────────────────────┐
│  ◈ NEXUS   [workspace ▾]        Agents    Plans         │
└─────────────────────────────────────────────────────────┘
```

- Logo links to `/`
- Workspace dropdown: one option per unique `group_name`, updates `selectedGroup` store
- **Agents** link → `/` (active when path is `/` or `/agents/*`)
- **Plans** link → `/plans` (active when path starts with `/plans`)

### Agents page (`/`)

- Agent cards grid filtered by `selectedGroup`
- Card footer: **Jobs** button (→ `/agents/{id}/jobs`) + **Settings** button (→ `/agents/{id}/settings`) + **VSCode** link
- No workspace group-bar pills (moved to top bar dropdown)
- No Plan tab

### Agent jobs page (`/agents/{id}/jobs`)

- Header: agent name + status + back link to `/`
- Job list: title, status badge, target group, updated time
- Click job row → `/jobs/{id}`
- Jobs filtered to `assigned_worker_id = {id}` (uses existing list jobs API with worker filter, or frontend filters from store)
- Empty state if no jobs

### Agent settings page (`/agents/{id}/settings`)

- Same content as current `/agents/{id}` — no changes to functionality

### Plans page (`/plans`)

- Same plan sessions UI as current Plan tab — list view + session detail chat
- No changes to functionality

### Sidebar

- Removed entirely from layout

---

## Assumptions

1. The job list for an agent uses frontend filtering from the `jobs` store (`assigned_worker_id === worker_id`), no new backend endpoint needed
2. The workspace dropdown shows all unique groups from `allWorkers` store, same as the current group-bar
3. Active nav link is determined by `$page.url.pathname` from SvelteKit
4. The current `/agents` list page (`routes/(app)/agents/+page.svelte`) is removed — the home page `/` now serves this role
5. VSCode button stays on the agent card; Settings button now routes to `/agents/{id}/settings`

---

## Docs to update

- `spec_v1_oagent_wagent_webapp.md` §17.1 — rewrite layout section for new nav structure
- `spec_v1_oagent_wagent_webapp.md` §17.1.2 — note discussion framing
- `ARCHITECTURE.md` — update frontend routes table
