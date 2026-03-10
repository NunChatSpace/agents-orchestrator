# Tasks

This directory contains task files. A task file serves one of two purposes:

- **Work order** (`Status: pending`) — created before implementation to give an implementing agent clear, scoped instructions.
- **Completion record** (`Status: done`) — updated or created after a phase is fully implemented to record what changed and why.

**Filename format:** `{phase_id}_{short-slug}.md`

Example: `phase01_project-bootstrap.md`

## Task File Structure

```md
# Task: {Phase Name}

## Phase
phase01

## Status
pending | done

## Plan Reference
`.agent/plans/plan_vN_...md`   ← include when task comes from a plan

## Goal
One or two sentences describing what this task accomplishes and why.

## What Already Exists
- list of relevant existing files and what they currently do
- helps the agent avoid redundant work

## Files to Change
- path/to/file.go
- path/to/file.ts

## Implementation Instructions
Step-by-step instructions. Include code shapes, validation rules, and any ordering constraints.

## Validation Rules Summary
Table of error cases and exact error strings when relevant.

## Test Cases
Explicit list of test scenarios (both positive and negative).

## Notes
Deviations from the plan, assumptions, gotchas, and things future readers should know.

## Completed At         ← fill in when marking done
YYYY-MM-DD
```

## Workflow

1. Before implementation: create task file with `Status: pending` and full instructions.
2. Implementing agent reads task, presents a plan, waits for approval (per AGENTS.md).
3. After implementation: agent updates `Status` to `done`, fills in `Completed At`, and adds any Notes about deviations or assumptions.
4. Update the Index table below.

## Index

| File | Phase | Status |
|------|-------|--------|
| phase01_project-bootstrap.md | phase01 | done |
| phase02_database-migrations.md | phase02 | done |
| phase03_models-repos-services.md | phase03 | done |
| phase04_controllers-routes.md | phase04 | done |
| phase05_websocket-hub.md | phase05 | done |
| phase06_frontend-stores-api.md | phase06 | done |
| phase07_frontend-components-routes.md | phase07 | done |
| phase08_scheduling-logic.md | phase08 | done |
| phase09_preview-role-overrides.md | phase09 | done |
| phase10_preview-frontend.md | phase10 | pending |
