# spec_v2_office_2d_map.md — Office Feature (Gather-Style 2D Map)

## 1. Goal

Add an `/office` page — a top-down 2D pixel-art style virtual office where the user can walk their avatar around, see worker agent status visually in real-time, and interact with individual agents to view job history or dispatch a new job.

---

## 2. Design Philosophy

- Minimal but immersive — consistent with NEXUS dark cyber-premium theme
- No game engine dependency — pure HTML5 Canvas (keeps bundle small)
- Functional-first — the map is a navigation UI, not a game
- Real-time status — leverages existing WebSocket hub for live worker updates

---

## 3. Map Design

### 3.1 Visual Style

- Top-down, tile-based grid (each tile = 32×32 px)
- Dark floor tiles matching `--nx-bg` palette with subtle grid pattern
- Desk/workstation tiles for each worker agent
- Wall tiles to define room boundaries
- Ambient violet glow effects consistent with NEXUS theme

### 3.2 Map Layout

```
+---------------------------------------+
|         [NEXUS OFFICE — fi-backend]   |
|   [Desk: fi-backend1] [Desk: fi-backend2] |
|                                       |
|         [NEXUS OFFICE — fi-frontend]  |
|   [Desk: fi-frontend1] [Desk: fi-frontend2] |
|                                       |
|         [NEXUS OFFICE — ib-kha]       |
|   [Desk: ib-kha]                      |
+---------------------------------------+
```

Map layout is organized by `group_name`, with desks clustered into room zones.

### 3.3 Map Data Format (Frontend)

Map is defined as a static TypeScript config file (no DB migration needed):

```typescript
// src/lib/office/mapConfig.ts

export type TileType = 'floor' | 'wall' | 'desk' | 'void';

export interface MapConfig {
  width: number;     // in tiles
  height: number;    // in tiles
  tileSize: number;  // px (default: 32)
  tiles: TileType[][]; // [row][col]
}

export interface DeskConfig {
  workerId: string;
  tileX: number;
  tileY: number;
  labelOffset?: { x: number; y: number };
}

export const MAP_CONFIG: MapConfig = { ... };
export const DESK_CONFIGS: DeskConfig[] = [ ... ];
```

Worker agent positions are pre-configured per group. New workers added via Settings get assigned to a default desk position (configurable in Settings page — see §9).

---

## 4. User Avatar

### 4.1 Appearance

- Small 16×16 px pixel sprite (simple colored character, violet/white palette)
- Name label floating above avatar: `"You"`
- Smooth movement (interpolated between tiles)

### 4.2 Movement

- **Keyboard:** WASD or Arrow Keys
- **Mobile:** D-pad overlay buttons (bottom-left of canvas)
- Speed: 4 tiles/second
- Collision detection against wall tiles and desk tiles (player cannot walk through)
- Canvas pans/follows the player (camera system) for large maps

### 4.3 Player State (Frontend only, no DB)

```typescript
interface PlayerState {
  x: number;      // pixel position
  y: number;      // pixel position
  facing: 'up' | 'down' | 'left' | 'right';
  moving: boolean;
}
```

---

## 5. Worker Agent NPCs

### 5.1 Appearance

Each worker agent appears as a small sprite sitting at their desk.

Status is shown through visual cues:

| Worker Status | Visual |
|---|---|
| `idle` | Green glowing sprite, static |
| `busy` | Orange pulsing sprite, slight animation |
| `pending_user` | Blue pulsing sprite + exclamation icon |
| `offline` | Grey/dim sprite, no glow |

Status badge (small colored dot) floats above the agent sprite.

### 5.2 Agent Label

- Worker `name` shown below sprite
- `group_name` shown in smaller muted text
- Current job count badge (number of active/assigned jobs)

### 5.3 Real-Time Status Updates

Worker status updates are received via the **existing WebSocket hub** (same WS connection used by the agents page).

Frontend `allWorkers` store already holds live worker data — office map subscribes to the same store and re-renders affected NPCs when status changes.

---

## 6. Interaction System

### 6.1 Proximity Detection

When the player avatar is within **2 tiles** of a worker agent's desk, an interaction prompt appears:

```
[ Press E to interact — fi-backend1 ]
```

Only the nearest agent within range shows the prompt.

### 6.2 Interaction Trigger

- **Keyboard:** Press `E`
- **Click/Tap:** Click directly on the agent sprite

### 6.3 Interaction Panel

When triggered, an **interaction side panel** slides in from the right (does not navigate away from the map). The panel is a glassmorphism overlay matching `.nx-card` style.

Panel sections:

1. **Agent Header** — name, group, status badge, workspace path
2. **Active Job** — current job title + status (if any)
3. **Recent Jobs** — last 3 jobs (title, status, timestamp) with link to `/jobs/{id}`
4. **Action buttons:**
   - `New Job →` — opens New Job modal (target_group pre-filled, manual_worker_override pre-filled)
   - `View All Jobs →` — navigates to `/agents/{id}/jobs`
   - `Settings →` — navigates to `/agents/{id}/settings`

Panel closes on:
- Press `Escape`
- Click outside the panel
- Press `E` again while within range

---

## 7. New Job Modal (from Office)

When user clicks `New Job →` from the interaction panel:

- Reuses the existing New Job modal component
- Pre-fills:
  - `target_group` = agent's `group_name`
  - `manual_worker_override` = this specific agent's `worker_id`
- User fills in `prompt` (and optional `title`)
- On submit: `POST /api/v1/jobs` + `POST /api/v1/jobs/{id}/submit`
- On success: panel shows confirmation, optionally navigate to `/jobs/{id}`

---

## 8. Navigation

### 8.1 Top Bar

Office page is added to the top bar as a new nav link:

```
◈ NEXUS | [Workspace Dropdown] | Agents | Plans | Office
```

Route: `/office`

### 8.2 Active State

Top bar "Office" link is active when path is `/office`.

---

## 9. Worker Desk Position (Settings Integration)

To support dynamic worker registration, each worker has an optional `map_x` and `map_y` field added to the worker model. This allows admins to pin agents to specific desk locations.

### 9.1 Backend Model Change

```go
// Additive migration (no breaking change)
ALTER TABLE workers ADD COLUMN map_x INTEGER DEFAULT 0;
ALTER TABLE workers ADD COLUMN map_y INTEGER DEFAULT 0;
```

### 9.2 Settings Page

Agent Settings page (`/agents/{id}/settings`) gains a new section:

**Office Position**
- Input: `X tile`, `Y tile`
- Button: `Save Position`
- API: `PATCH /api/v1/workers/{id}` (existing endpoint, add `map_x`, `map_y` to payload)

### 9.3 Default Assignment

Workers with `map_x = 0, map_y = 0` are auto-assigned to a default desk position based on their `group_name` and registration order (frontend resolves this from `DESK_CONFIGS` fallback).

---

## 10. Canvas Rendering Architecture

### 10.1 Tech Choice

Pure HTML5 Canvas — no Phaser, no PixiJS. Keeps bundle size minimal.

### 10.2 Render Loop

```typescript
// src/lib/office/OfficeCanvas.ts

class OfficeEngine {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private lastTime = 0;

  start() {
    requestAnimationFrame(this.loop.bind(this));
  }

  private loop(timestamp: number) {
    const delta = timestamp - this.lastTime;
    this.lastTime = timestamp;
    this.update(delta);
    this.render();
    requestAnimationFrame(this.loop.bind(this));
  }

  private update(delta: number) {
    // Move player, check collisions, check proximity
  }

  private render() {
    // 1. Clear canvas
    // 2. Draw tiles (floor, wall, desk)
    // 3. Draw agent NPCs (status glow, sprite, label)
    // 4. Draw player avatar
    // 5. Draw proximity prompt overlay
  }
}
```

### 10.3 Camera System

Canvas dimensions = viewport size (responsive).
Camera follows player with smooth lerp.

```typescript
const cameraX = lerp(cameraX, player.x - canvas.width / 2, 0.1);
const cameraY = lerp(cameraY, player.y - canvas.height / 2, 0.1);
```

### 10.4 Sprites

Sprites are drawn programmatically (CSS pixel-art style using Canvas API fillRect) — no external sprite sheets required. This avoids asset loading complexity.

**Worker NPC sprite (schematic):**
```
 ██     ← head (white)
████    ← body (group color)
 ██     ← legs
```

**Player avatar:**
Same structure with violet body.

---

## 11. Frontend File Structure

```
frontend/src/
├── routes/(app)/
│   └── office/
│       └── +page.svelte          ← Office page (mounts canvas)
├── lib/
│   └── office/
│       ├── OfficeEngine.ts       ← Canvas render loop
│       ├── mapConfig.ts          ← Static map & desk layout
│       ├── playerController.ts   ← Keyboard input + movement
│       ├── npcRenderer.ts        ← Worker NPC drawing helpers
│       ├── proximityDetector.ts  ← Nearest agent within range
│       └── tileRenderer.ts       ← Tile drawing helpers
└── components/
    └── Organisms/
        └── OfficeInteractionPanel.svelte  ← Side panel overlay
```

---

## 12. Backend Changes

Minimal changes required:

| Change | Scope |
|---|---|
| `ALTER TABLE workers ADD COLUMN map_x, map_y` | Migration |
| `PATCH /api/v1/workers/{id}` — accept `map_x`, `map_y` | Controller / Service |
| Worker list response includes `map_x`, `map_y` | Domain DTO |

No new endpoints required. The office page uses existing:
- `GET /api/v1/workers` — load all worker positions + statuses
- WebSocket hub — live worker status updates
- `POST /api/v1/jobs` + `POST /api/v1/jobs/{id}/submit` — job dispatch
- `GET /api/v1/jobs` — recent jobs for panel (filtered client-side by `assigned_worker_id`)

---

## 13. Agent Persona System

### 13.1 Overview

Each worker agent can be assigned a **persona** — a cyberpunk character archetype that determines how their NPC sprite looks on the Office map. Persona is set per-agent in the Settings page and stored in the database.

This section defines planned v2 behavior. It is not part of current v1 office behavior until implementation is completed.

### 13.2 Available Personas

| Persona ID | Name | Visual Style | Primary Color |
|---|---|---|---|
| `netrunner` | Netrunner | Sleek, hooded, data-stream cyberware (Songbird-style) | Cyan `#00f0ff` |
| `tech` | Tech | Mechanical arms, tool belt, goggle visor | Amber `#f59e0b` |
| `fixer` | Fixer | Long coat, calm posture, minimal cyberware | Gold `#eab308` |
| `solo` | Solo | Combat armor, broad build, red visor | Red `#ef4444` |
| `nomad` | Nomad | Rugged jacket, worn aesthetic, earth tones | Orange `#f97316` |

Default persona if not set: `netrunner`.

### 13.3 Backend Changes

```sql
-- Additive migration (no breaking change)
ALTER TABLE workers ADD COLUMN persona VARCHAR(32) DEFAULT 'netrunner';
```

Allowed values enforced at service layer: `netrunner | tech | fixer | solo | nomad`.

`PATCH /api/v1/workers/{id}` — accepts optional `persona` field alongside existing fields.

Worker API response includes `persona` field.

### 13.4 Settings Page — Persona Selector

Agent Settings page (`/agents/{id}/settings`) gains a **Persona** section:

- Visual grid of 5 persona cards — each showing:
  - Persona name
  - Small sprite preview (canvas-drawn, same as office NPC)
  - Short description (e.g. "Netrunner — data specialist, sleek cyberware")
- Selected card highlighted with neon border
- Auto-saves on click via `PATCH /api/v1/workers/{id}` — no extra submit button needed

### 13.5 NPC Sprite Design per Persona

Each persona has a unique pixel sprite drawn programmatically via Canvas API. All sprites follow the Cyberpunk Songbird art direction (dark palette, neon accents, cyberware details).

**Netrunner (default — Songbird-style):**
- Slim build, hooded silhouette
- Cyan spine cyberware line running down the back
- Data stream particles floating around wrists
- Color: Cyan

**Tech:**
- Stocky build, mechanical shoulder mount
- Amber visor glow
- Tool/wrench detail at hip
- Color: Amber

**Fixer:**
- Long coat silhouette, slightly taller
- Gold collar/lapel highlight
- Cigarette ember glow (small orange dot)
- Color: Gold

**Solo:**
- Broad-shouldered, armored chest
- Red visor across face
- Angular, combat-ready stance
- Color: Red

**Nomad:**
- Rugged layered jacket
- Dusty texture (lower opacity base)
- Orange scarf detail
- Color: Orange

### 13.6 Office Map Integration

`npcRenderer.ts` reads `worker.persona` and delegates to a persona-specific draw function:

```typescript
// npcRenderer.ts
const PERSONA_RENDERERS: Record<string, PersonaRenderer> = {
  netrunner: drawNetrunner,
  tech:      drawTech,
  fixer:     drawFixer,
  solo:      drawSolo,
  nomad:     drawNomad,
};

export function drawWorkerNPC(ctx, worker, cx, cy, pulse) {
  const render = PERSONA_RENDERERS[worker.persona] ?? drawNetrunner;
  render(ctx, cx, cy, statusColor(worker.status), pulse);
}
```

Each renderer accepts `(ctx, cx, cy, statusColor, pulse)` — the status color drives the glow/ring regardless of persona, so idle/busy/offline feedback remains consistent across all character types.

### 13.7 Persona Sprite Preview Component

A small reusable `PersonaSprite.svelte` component renders a live canvas preview of a persona sprite (same draw functions, scaled down). Used in:
- Settings page persona selector cards
- Interaction panel agent header (small avatar icon)

---

## 14. Non-Goals for v2 (Office)

- No audio / spatial audio
- No real-time multiplayer (other humans walking on the map)
- No custom tilemap editor
- No avatar customization
- No chat or @mention within the office map
- No drag-to-reposition desks in UI (done via Settings page only)

---

## 15. Acceptance Criteria

### 15.1 Map Rendering
- Office map renders at `/office` with tile-based layout
- Worker agent NPCs appear at configured desk positions
- Status glow/color reflects real-time worker status

### 15.2 Player Movement
- WASD / Arrow keys move the player avatar smoothly
- Player cannot walk through walls or desk tiles

### 15.3 Proximity Interaction
- Interaction prompt appears when within 2 tiles of a worker agent
- Press E or click agent to open interaction panel
- Panel shows agent info, recent jobs, and action buttons

### 15.4 Job Dispatch
- New Job modal opens from panel with pre-filled group + worker
- Job is submitted successfully and user can navigate to job chat

### 15.5 Real-Time Status
- Worker status changes (idle → busy etc.) update NPC appearance without page reload

### 15.6 Navigation
- "Office" nav link in top bar is active on `/office`
- Links in interaction panel navigate correctly to existing pages

---

## 16. Open Questions for v3

- Should we support map zones with different visual themes per group?
- Should there be a minimap overlay in the corner?
- Should player position persist across page reloads (localStorage)?
- Should mobile touch-drag also control avatar movement?
