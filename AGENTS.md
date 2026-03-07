# AGENTS.md

## Purpose
This file defines the general working contract for any agent contributing to this project.

It is intentionally generic.
Product rules belong in the spec.
System design belongs in the architecture document.

This file defines **how the agent should work**, not the full product behavior.

---

## Required Reads
Before planning, editing, or proposing implementation, read these files in order:

1. `AGENTS.md`
2. `.agent/specs/spec_v1_oagent_wagent_webapp.md`
3. `.agent/ARCHITECTURE.md`

Do not start from memory alone.
Do not assume the current code fully reflects the intended design.

### Frontend UI context

The frontend uses the NEXUS design system (dark cyber-premium theme).
Visual rules, color tokens, and CSS class contracts are documented in spec §17.2.
Do not introduce new color values, fonts, or component patterns outside that system without updating the spec first.

---

## Source of Truth
If information conflicts, use this priority:

1. explicit user instruction in chat
2. `AGENTS.md`
3. `.agent/specs/spec_v1_oagent_wagent_webapp.md`
4. `.agent/ARCHITECTURE.md`
5. current repository code

If two sources conflict, stop and point out the exact conflict before continuing.

Do not silently resolve contradictions on your own.

---

## Working Rules

### 1. Plan first
Before changing code, always provide:
- a short plan
- affected files or areas
- important assumptions or missing information

Do not jump straight into implementation.

### 2. Do not guess
If required information is missing, ask for the missing decision instead of inventing behavior.

If a best-effort proposal is still possible, state assumptions clearly.

### 3. Keep scope tight
Prefer the smallest coherent implementation slice.
Do not introduce extra abstractions, features, or architecture unless needed.

### 4. Challenge assumptions
Do not agree by default.
If the requested direction is inconsistent, unclear, or overengineered, say so directly.

### 5. Respect existing boundaries
Do not move product rules into `AGENTS.md`.
Do not move architecture decisions into random implementation files.
Keep concerns separated:
- `AGENTS.md` = how to work
- `spec` = what to build
- `architecture` = how the system is structured

### 6. Use consistent naming
Do not rename important concepts casually.
Follow naming already defined in the spec and architecture documents.

### 7. Keep changes inspectable
Prefer explicit, easy-to-review changes over hidden magic.
Avoid clever behavior that makes the system harder to reason about.

---

## Conflict Handling
If the user request conflicts with:
- this file
- the current spec
- the architecture
- existing implementation constraints

the agent must:

1. stop
2. identify the exact conflict
3. explain the impact briefly
4. ask whether to update the rule/spec/architecture first

Do not continue as if the conflict does not matter.

---

## Documentation Rules
If implementation changes behavior, update the relevant project documents.

Typical expectation:
- product behavior changes -> update `.agent/specs/spec_v1_oagent_wagent_webapp.md`
- system structure changes -> update `.agent/ARCHITECTURE.md`

Do not leave docs behind after changing behavior.

---

## Definition of Done
A task is not done unless all of these are true:

1. the change matches the current source of truth
2. assumptions are stated when relevant
3. important naming stays consistent
4. related docs are updated if behavior or structure changed
5. the result is small enough to review clearly

---

## Default Behavior
When unsure:
- prefer asking over guessing
- prefer smaller scope over broader redesign
- prefer consistency over cleverness
- prefer explicit rules over hidden inference

Build the right thing, not the most elaborate thing.
