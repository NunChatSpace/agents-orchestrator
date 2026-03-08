# Agent Orchestrator

A personal web app for managing AI worker agents. Create jobs, dispatch them to workers, and continue conversations — all from one interface.

---

## Overview

- **Proxy (single entrypoint)** — Nginx on host port `5174`
- **Frontend** — SvelteKit Node server inside Docker (internal only)
- **Backend** — Go API server inside Docker (internal only)
- **Database** — PostgreSQL 16 inside Docker (internal only)

---

## Prerequisites

- [Docker](https://www.docker.com/) + Docker Compose v2
- [Node.js](https://nodejs.org/) 22+ (for local frontend dev)
- [Go](https://golang.org/) 1.23+ (optional, for local backend dev only)
- CLI agents installed on the host (for workers to invoke):
  - [Claude Code](https://github.com/anthropics/claude-code): `npm install -g @anthropic-ai/claude-code`
  - [Codex](https://github.com/openai/codex): `npm install -g @openai/codex`

---

## Quick Start

### 1. Clone and configure environment

```bash
git clone <repo-url>
cd agent-orchestrator
cp .env.example .env   # if it exists, or create manually
```

Create a `.env` file in the project root. Docker Compose reads this file automatically.

```env
# REQUIRED — absolute path on your host machine.
# Docker bind-mounts this to /workspaces inside the container.
# The frontend also reads it (via Vite's PUBLIC_ prefix) to build
# correct vscode://file/... deep links for each agent card.
WORKSPACES_PATH=/absolute/path/to/your/workspaces
```

### 2. Start all services

```bash
docker compose up -d
```

This starts:
- `proxy` — the only host-exposed service on `:5174`
- `frontend` — internal web app server
- `backend` — internal API server (runs migrations automatically on start)
- `postgres` — internal database

### 3. Log in

Open `http://localhost:5174` (or your LAN host like `http://192.168.1.172:5174`).

Default credentials (set via `SEED_USERNAME` / `SEED_PASSWORD` in docker-compose):

```
Username: admin
Password: changeme
```

---

## Rebuilding the Backend

After any Go code change:

```bash
docker compose up -d --build backend
```

---

## Worker Setup

Workers are registered via the UI or API. Each worker needs:

| Field | Description |
|---|---|
| `name` | Display name (e.g. `fi-backend1`) |
| `group_name` | Workspace group (e.g. `fi-backend`) |
| `workspace` | Absolute path inside the container (e.g. `/workspaces/fi-backend1`) |
| `cli_command` | CLI to invoke (`claude` or `codex`) |
| `git_repo_url` | Optional — shown in Settings |

The workspace path must exist inside the container. Since `WORKSPACES_PATH` is bind-mounted to `/workspaces`, create a subdirectory on the host:

```bash
mkdir -p /path/to/your/workspaces/fi-backend1
```

### CLI agent authentication

- **Claude Code**: `~/.claude` is automatically shared from the host (via Docker volume mount). Log in once on the host with `claude` and workers inside the container will reuse the session.
- **Codex**: `~/.codex` is mounted similarly. Run `codex` on the host first to authenticate.
- **SSH / Git**: `~/.ssh` and `~/.gitconfig` are mounted read-only so workers can use git without reconfiguration.

---

## Project Structure

```
agent-orchestrator/
├── backend/
│   ├── internal/
│   │   ├── controllers/    # HTTP handlers
│   │   ├── services/       # Business logic (dispatcher, plan sessions, etc.)
│   │   ├── repository/     # Database queries
│   │   ├── models/         # DB entity structs
│   │   └── domains/        # Request/Response DTOs
│   ├── migrations/         # SQL migration files (auto-run on start)
│   ├── Dockerfile
│   └── main.go
├── frontend/
│   ├── src/
│   │   ├── routes/         # SvelteKit pages
│   │   ├── components/     # UI components (Atomic Design)
│   │   ├── lib/apis/       # API client functions
│   │   ├── stores/         # Svelte global state
│   │   └── types/          # TypeScript interfaces
│   └── package.json
├── .agent/
│   ├── specs/              # Product spec
│   └── ARCHITECTURE.md     # System architecture reference
├── AGENTS.md               # Working contract for AI agents
└── docker-compose.yml
```

---

## Environment Variables

All backend env vars are set in `docker-compose.yml`. Key ones:

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | postgres://oagent:secret@postgres:5432/oagent | Postgres connection string |
| `SESSION_SECRET` | `changeme32byteslongsecretkey1234` | Cookie signing secret — **change in production** |
| `SEED_USERNAME` | `admin` | Admin account created on first run |
| `SEED_PASSWORD` | `changeme` | Admin password — **change in production** |
| `PORT` | `8080` | Backend listen port |
| `FRONTEND_ORIGIN` | `http://localhost:5174` | CORS allowed origin |
| `VITE_BACKEND_URL` | *(empty)* | Leave empty for same-origin `/api` via proxy. Set only if bypassing proxy. |
| `WORKSPACES_PATH` | — | **Required.** Host path bind-mounted to `/workspaces`; also used by the frontend for VSCode deep links |

---

## Development Notes

- Migrations run automatically at backend startup via the embedded `migrations/` directory.
- WebSocket events (`job_updated`, `message_added`, `worker_updated`) hydrate the frontend in real time — no polling.
- Plan sessions use SSE (Server-Sent Events) for streaming agent replies.
- The frontend design system is **NEXUS** (dark cyber-premium). See `spec §17.2` for color tokens and CSS patterns.

---

## Useful Commands

```bash
# Start everything
docker compose up -d

# Rebuild app stack after config/frontend/backend changes
docker compose up -d --build proxy frontend backend

# View backend logs
docker compose logs -f backend

# Stop everything
docker compose down

# Stop and remove volumes (wipes database)
docker compose down -v

# Frontend type check
cd frontend && npm run check
```
