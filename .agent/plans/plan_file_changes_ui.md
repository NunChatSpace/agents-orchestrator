# Plan: Agent File Changes UI

## Summary

Add visibility into file changes made by agents during job execution.
Three phases, all implemented:

1. **File-change messages** — detect agent file-write tool calls, store as `kind: "file_change"` messages with structured JSON content
2. **Diff data** — enrich message content with old/new strings from tool inputs so actual diff is available
3. **Split-panel layout** — two-column job view: chat on left, Changes panel on right (GitHub PR style)

---

## Phase 1 — File Change Message Kind

### Backend

- `backend/migrations/006_message_kind_file_change.sql` — drop + recreate CHECK constraint to add `'file_change'` and `'thinking'`
- `backend/internal/models/message.go` — add `MessageKindFileChange = "file_change"`
- `backend/internal/services/dispatcher_service.go` — add `JobResultHandler.HandleFileChange`, `FileChangePayload`, `DiffHunk`, `fileChangePayload()` helper that extracts diff data from Claude tool inputs (Write, Edit, MultiEdit, str_replace_editor, NotebookEdit)
- `backend/internal/services/job_service.go` — add `HandleFileChange` to `JobService` interface and implement it (creates message, broadcasts via WS)

### Frontend

- `frontend/src/types/message.ts` — add `'file_change'` to `MessageKind`

---

## Phase 2 — Diff Data in Tool Inputs

Tool input fields captured per tool:

| Tool | Fields |
|---|---|
| `Write` | `content` → `new` field |
| `Edit` | `old_string`, `new_string` → single hunk |
| `MultiEdit` | `edits[].old_string`, `edits[].new_string` → multiple hunks |
| `str_replace_editor create` | `file_text` → `new` field |
| `str_replace_editor str_replace` | `old_str`, `new_str` → single hunk |
| `str_replace_editor insert` | `new_str` → hunk with empty old |

Content truncated at 4000 chars per hunk with `…(truncated)` marker.

Message `content` JSON format:
```json
{ "op": "edit", "path": "src/main.go", "hunks": [{"old":"...","new":"..."}] }
{ "op": "write", "path": "src/new.go", "new": "..." }
```

---

## Phase 3 — Split-Panel Layout

### Files changed

- `frontend/src/components/organisms/ChangesPanel.svelte` — NEW right-side panel:
  - Filters messages to `kind === 'file_change'`
  - Shows header: `Changes (N)`
  - Each file: `+`/`~` badge + path, click to expand diff
  - Diff: red lines (`-`) for old, green lines (`+`) for new
  - `···` separator between hunks (MultiEdit)

- `frontend/src/routes/(app)/jobs/[job_id]/+page.svelte` — split layout:
  - Left: `chat-panel` (flex-1) — MessageFeed + Composer
  - Right: `changes-panel` (360px fixed) — ChangesPanel
  - Divider: `border-right: 1px solid rgba(139,92,246,0.12)`

- `frontend/src/components/molecules/MessageBubble.svelte` — simplify file_change row:
  - Compact static row (badge + path), no expand
  - Full diff lives in ChangesPanel

---

## Assumptions

- Claude CLI tool names are stable (`Write`, `Edit`, `MultiEdit`, `str_replace_editor`, `NotebookEdit`)
- Codex does not expose file-change tool details in its JSONL format (only Claude is handled)
- 360px fixed width for ChangesPanel is suitable for most paths; can be made resizable later

---

## Deploy

```bash
docker compose up -d --build backend
```

Frontend hot-reloads automatically.
