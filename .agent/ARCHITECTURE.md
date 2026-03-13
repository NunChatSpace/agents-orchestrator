# Architecture Template

A reference architecture for Go backend + Svelte frontend projects.

---

## Project Structure

```
project/
├── backend/
│   ├── internal/
│   │   ├── controllers/   # HTTP handlers (thin layer)
│   │   ├── services/      # Business logic (interfaces + implementations)
│   │   ├── repository/    # Data access layer
│   │   ├── models/        # Database entities
│   │   ├── domains/       # Response/Request DTOs
│   │   ├── stackregistry/ # File-based runtime stack definitions
│   │   ├── views/         # HTTP response helpers
│   │   └── utils/         # Shared utilities
│   ├── migrations/        # SQL migration files
│   └── main.go            # Entry point, DI container, router setup
├── frontend/
│   ├── src/
│   │   ├── routes/        # SvelteKit pages (file-based routing)
│   │   ├── components/    # UI components (Atomic Design)
│   │   ├── lib/apis/      # API client functions
│   │   ├── stores/        # Svelte stores (global state)
│   │   └── types/         # TypeScript interfaces
│   └── svelte.config.js
├── k8s/                   # Kubernetes manifests (optional)
└── docker-compose.yml     # Local development
```

---

## Backend Layers

### 1. Entry Point (`main.go`)

Responsibilities:
- Build dependency injection container
- Run database migrations
- Configure middleware (CORS, rate limiting, logging)
- Mount controllers to router
- Start HTTP server

```go
func main() {
    container := buildContainer()  // Register all dependencies
    runMigrations(db)

    router := mux.NewRouter()
    api := router.PathPrefix("/api").Subrouter()

    // Mount controllers
    mountUserController(api, container)
    mountPostController(api, container)

    http.ListenAndServe(":8080", router)
}
```

### 2. Controllers (`/internal/controllers/`)

**Purpose:** HTTP request handling (thin layer)

**Responsibilities:**
- Parse and validate request input
- Extract user from context (if authenticated)
- Call service methods
- Return JSON response

**Pattern:**
```go
type Handler struct {
    Service services.SomeService
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // 1. Parse input
    // 2. Validate
    // 3. Call service
    // 4. Return response
}
```

**Do:**
- Keep handlers thin
- Validate input here
- Handle HTTP concerns (status codes, headers)

**Don't:**
- Put business logic here
- Access repository directly

### 3. Services (`/internal/services/`)

**Purpose:** Business logic

**Pattern:**
```go
// Interface (contract)
type UserService interface {
    Create(ctx context.Context, input CreateUserInput) (*User, error)
    GetByID(ctx context.Context, id string) (*User, error)
    Authenticate(ctx context.Context, email, password string) (*User, error)
}

// Implementation
type userService struct {
    repo     repository.UserRepository
    hasher   PasswordHasher
}

func NewUserService(repo repository.UserRepository) UserService {
    return &userService{repo: repo}
}
```

**Responsibilities:**
- Validate business rules
- Orchestrate repository calls
- Handle transactions if needed
- Transform data between layers

For this project, instruction composition belongs in the service layer. The backend stores three worker-owned instruction fields:
- `instruction_job`
- `instruction_plan`
- `instruction_discuss`

Factory defaults for those fields live in backend code. Worker creation seeds the three fields from the hardcoded defaults, and runtime dispatch reads only the worker record before invoking the CLI.

### 4. Repository (`/internal/repository/`)

**Purpose:** Data access (database queries)

**Pattern:**
```go
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Create(ctx context.Context, user *models.User) error
    Update(ctx context.Context, user *models.User) error
    SoftDelete(ctx context.Context, id string) error
}

type userRepo struct {
    db *sqlx.DB
}
```

**Conventions:**
- Use soft deletes (`deleted_at` column)
- Filter with `WHERE deleted_at IS NULL`
- Use prepared statements or query builder
- Implement Dataloader for batch loading (prevent N+1)

Do not create a singleton instruction-settings table for worker prompting. Factory defaults belong in backend code, while runtime instruction state belongs on the worker record.

### 5. Models (`/internal/models/`)

**Purpose:** Database entities (table mappings)

```go
type User struct {
    ID        string     `db:"id"`
    Email     string     `db:"email"`
    Password  string     `db:"password"`
    CreatedAt time.Time  `db:"created_at"`
    UpdatedAt time.Time  `db:"updated_at"`
    DeletedAt *time.Time `db:"deleted_at"`
}
```

### 6. Domains (`/internal/domains/`)

**Purpose:** Request/Response DTOs (what API returns)

```go
type UserResponse struct {
    ID    string `json:"id"`
    Email string `json:"email"`
    Name  string `json:"name"`
}

type CreateUserRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    Name     string `json:"name"`
}
```

---

## API Contract Conventions

Use consistent API request/response contracts across services. The
examples below are generic conventions for this architecture template.

### Request Format

**Base path convention**

```text
/api/v1
```

**Rules**
- Use JSON request bodies for `POST`, `PUT`, `PATCH`
- Use query parameters for filtering and pagination
- Keep request bodies resource-specific (do not wrap in an extra
  top-level `data` field unless the entire system adopts that pattern)
- Return `Content-Type: application/json`

**Example (list with pagination)**

```http
GET /api/v1/users?page=1&page_size=20&search=alice HTTP/1.1
Cookie: session=...
Accept: application/json
```

**Example (create resource)**

```http
POST /api/v1/users HTTP/1.1
Content-Type: application/json
Accept: application/json

{
  "email": "alice@example.com",
  "password": "secret123",
  "name": "Alice"
}
```

### Response Format (Success + Error)

**Success response envelope**

```json
{
  "data": {
    "id": "u_123",
    "email": "alice@example.com",
    "name": "Alice"
  },
  "meta": {
    "request_id": "req_abc123"
  }
}
```

**Error response envelope**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {
      "email": ["invalid format"]
    }
  },
  "meta": {
    "request_id": "req_abc123"
  }
}
```

**Guidelines**
- `data` is present only on successful responses
- `error` is present only on failed responses
- `meta` is optional, but `request_id` is recommended for tracing
- Use stable machine-readable `error.code` values

### Pagination Format (Default: Page-Based)

Pick one pagination strategy per service and keep it consistent. The
default in this template is **page-based pagination**.

**Query parameters**
- `page` (1-based integer, default `1`)
- `page_size` (integer, default `20`, max `100`)

**Paginated list response**

```json
{
  "data": [
    {
      "id": "u_123",
      "email": "alice@example.com",
      "name": "Alice"
    }
  ],
  "meta": {
    "request_id": "req_abc123",
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_items": 42,
      "total_pages": 3,
      "has_next": true,
      "has_prev": false
    }
  }
}
```

**Notes**
- For highly dynamic feeds, cursor pagination may be a better choice
  than page-based pagination.
- Do not mix page-based and cursor-based formats within the same API
  surface without a clear version boundary.

---

## Backend Patterns

### Dependency Injection

Use a DI container (e.g., Uber's `dig`) to wire dependencies:

```go
func buildContainer() *dig.Container {
    c := dig.New()
    c.Provide(NewDB)
    c.Provide(repository.NewUserRepository)
    c.Provide(services.NewUserService)
    c.Provide(controllers.NewUserHandler)
    return c
}
```

### Middleware Chain

```
Request
  ↓ RateLimiter
  ↓ RequestLogger
  ↓ AuthMiddleware (for private routes)
  ↓ Handler
Response
```

### Router Structure

```go
api := router.PathPrefix("/api").Subrouter()

// Public routes (no auth required)
public := api.NewRoute().Subrouter()
public.HandleFunc("/auth/login", handler.Login).Methods("POST")

// Private routes (auth required)
private := api.NewRoute().Subrouter()
private.Use(AuthMiddleware)
private.HandleFunc("/me", handler.GetMe).Methods("GET")
```

### Session Authentication

```
1. User logs in → Create session token (UUID) → Store in DB
2. Set HttpOnly cookie with token
3. On each request → Read cookie → Lookup session → Inject user to context
4. User logs out → Delete session from DB → Clear cookie
```

### Soft Deletes

Never hard delete. Set `deleted_at` timestamp instead:

```sql
-- All queries include this filter
WHERE deleted_at IS NULL

-- "Delete" operation
UPDATE users SET deleted_at = NOW() WHERE id = $1
```

### Dataloader (Prevent N+1)

When loading nested data, batch load related entities:

```go
// Bad: N+1 queries
for _, post := range posts {
    post.User = repo.GetUser(post.UserID)  // 1 query per post
}

// Good: 2 queries total
userIDs := extractUserIDs(posts)
users := repo.GetUsersByIDs(userIDs)  // 1 query for all users
mapUsersToPost(posts, users)
```

---

## Frontend Layers

### 1. Routes (`/src/routes/`)

SvelteKit file-based routing:

```
routes/
├── +layout.svelte     # Global layout (header, nav)
├── +page.svelte       # Home page (/)
├── items/
│   └── +page.svelte   # /items
└── profile/
    └── +page.svelte   # /profile
```

### 2. Components (Atomic Design)

```
components/
├── Atoms/        # Basic elements (Button, Badge, Spinner)
├── Molecules/    # Combinations (JobListItem, StatusBadge, MessageBubble)
└── Organisms/    # Complex sections (Sidebar, TopBar, MessageFeed, Composer, NewJobForm)
```

Component styling uses Svelte `<style>` blocks with CSS custom properties from `app.css`.
Tailwind is used for layout and spacing; NEXUS-specific styles (glassmorphism, glow) are in scoped CSS blocks.
Global reusable patterns (`.nx-card`, `.nx-input`) are defined in `app.css` under `@layer components`.

### 3. Stores (`/src/stores/`)

Global state with Svelte stores:

```typescript
// auth.ts
import { writable, derived } from 'svelte/store';

export const user = writable<User | null>(null);
export const isSignedIn = derived(user, $user => $user !== null);

export function signOut() {
    user.set(null);
    // Clear cookies/localStorage
}
```

### 4. API Layer (`/src/lib/apis/`)

Centralized API client:

```typescript
// fetcher.ts
const API_BASE = '/api';

type ApiSuccess<T> = {
    data: T;
    meta?: Record<string, unknown>;
};

type ApiError = {
    error: {
        code: string;
        message: string;
        details?: unknown;
    };
    meta?: Record<string, unknown>;
};

export async function GET<T>(url: string): Promise<T> {
    const res = await fetch(`${API_BASE}${url}`, {
        credentials: 'include'  // Send cookies
    });
    const json = await res.json();
    if (!res.ok) {
        const err = json as ApiError;
        throw new Error(err.error?.message || res.statusText);
    }
    return (json as ApiSuccess<T>).data;
}

export async function POST<T>(url: string, body: unknown): Promise<T> {
    const res = await fetch(`${API_BASE}${url}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(body)
    });
    const json = await res.json();
    if (!res.ok) {
        const err = json as ApiError;
        throw new Error(err.error?.message || res.statusText);
    }
    return (json as ApiSuccess<T>).data;
}
```

```typescript
// users.ts
import { GET, POST } from './fetcher';

export const getMe = () => GET<User>('/me');
export const signIn = (data: SignInRequest) => POST<User>('/sessions', data);
```

### 5. Types (`/src/types/`)

TypeScript interfaces matching backend DTOs:

```typescript
interface User {
    id: string;
    email: string;
    name: string;
}

interface PaginationMeta {
    page: number;
    page_size: number;
    total_items: number;
    total_pages: number;
    has_next: boolean;
    has_prev: boolean;
}

interface ApiResponse<T> {
    data: T;
    meta?: {
        request_id?: string;
        pagination?: PaginationMeta;
    };
}

interface ApiErrorResponse {
    error: {
        code: string;
        message: string;
        details?: unknown;
    };
    meta?: {
        request_id?: string;
    };
}
```

---

## Data Flow

### Request Lifecycle

```
Frontend Component
  ↓ calls API function
API Layer (lib/apis)
  ↓ HTTP request with credentials
Backend Controller
  ↓ validates, extracts user from context
Service Layer
  ↓ business logic
Repository
  ↓ SQL query
Database
  ↓ returns data
Repository → Service → Controller → JSON Response
  ↓
Frontend updates store/UI
```

### Authentication Flow

```
1. User submits login form
2. POST /api/sessions { email, password }
3. Backend validates credentials
4. Create session, set HttpOnly cookie
5. Return user data
6. Frontend stores user in auth store
7. Subsequent requests include cookie automatically
```

---

## Database Conventions

### Migration Files

```
migrations/
├── 0001_init.up.sql
├── 0001_init.down.sql
├── 0002_add_posts.up.sql
└── 0002_add_posts.down.sql
```

### Table Structure

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP  -- For soft deletes
);

-- Index for soft delete queries
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
```

### Full-Text Search (PostgreSQL)

```sql
ALTER TABLE posts ADD COLUMN search_vector tsvector;

CREATE INDEX idx_posts_search ON posts USING GIN(search_vector);

-- Query
SELECT * FROM posts
WHERE search_vector @@ plainto_tsquery('english', $1);
```

---

## Tech Stack

| Layer | Technology |
| --- | --- |
| Frontend | Svelte/SvelteKit, TypeScript, Tailwind CSS |
| Backend | Go, Gorilla mux, sqlx |
| Database | PostgreSQL |
| Storage | MinIO / S3 |
| Auth | HttpOnly session cookies |
| DI | Uber dig |
| Deploy | Docker Compose (single exposed proxy port), Kubernetes |
| Reverse Proxy | Nginx (routes `/` to frontend, `/api` and `/ws` to backend) |
| UI Fonts | Inter, Space Grotesk (Google Fonts) |
| UI Design System | NEXUS (dark cyber-premium theme, see spec §17.2) |
| Markdown | `marked` (parse) + `DOMPurify` (sanitize) — chat bubbles and instruction blocks |

---

## Project-Specific Notes (Agent Orchestrator)

### API Endpoints — Workers

| Method | Path | Handler | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/workers` | `WorkerController.List` | List all workers |
| POST | `/api/v1/workers` | `WorkerController.Create` | Register worker |
| GET | `/api/v1/workers/{worker_id}` | `WorkerController.Get` | Get single worker |
| PATCH | `/api/v1/workers/{worker_id}` | `WorkerController.Update` | Update worker settings (`cli_command`, `map_x`, `map_y`) |
| DELETE | `/api/v1/workers/{worker_id}` | `WorkerController.Delete` | Remove worker |
| POST | `/api/v1/workers/{worker_id}/ping` | `WorkerController.Ping` | Health check |
| POST | `/api/v1/workers/{worker_id}/plan` | `WorkerController.Plan` | Run CLI in plan mode; returns `{ prompt: string }` |

Worker response payloads include office placement fields:

- `map_x` (integer logical desk-grid X coordinate, default `0`)
- `map_y` (integer logical desk-grid Y coordinate, default `0`)
- `created_at` (used by frontend fallback desk ordering)

Schema change: migration `008_worker_map_position.sql` adds `workers.map_x` and `workers.map_y`.

### API Endpoints — Plan Sessions

| Method | Path | Handler | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/plan-sessions` | `PlanSessionController.List` | List non-discarded sessions for user |
| POST | `/api/v1/plan-sessions` | `PlanSessionController.Create` | Create new session `{ worker_id }` |
| GET | `/api/v1/plan-sessions/{id}` | `PlanSessionController.Get` | Get session + messages |
| PATCH | `/api/v1/plan-sessions/{id}` | `PlanSessionController.UpdateTitle` | Edit title |
| POST | `/api/v1/plan-sessions/{id}/message` | `PlanSessionController.SendMessage` | Send user message → SSE stream agent reply |
| POST | `/api/v1/plan-sessions/{id}/generate` | `PlanSessionController.Generate` | Final generate → SSE stream; stores `generated_prompt` |
| POST | `/api/v1/plan-sessions/{id}/complete` | `PlanSessionController.Complete` | Mark completed |
| POST | `/api/v1/plan-sessions/{id}/discard` | `PlanSessionController.Discard` | Soft-delete session |

### API Endpoints — Preview Runtime (Legacy Bundle System)

| Method | Path | Handler | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/preview-stacks` | `PreviewBundleController.ListStacks` | List file-based stack registry entries |
| GET | `/api/v1/preview-bundles` | `PreviewBundleController.List` | List preview bundles for current user |
| POST | `/api/v1/preview-bundles` | `PreviewBundleController.Create` | Create preview bundle from `{ stack_id, task_id, role_overrides[] }` |
| GET | `/api/v1/preview-bundles/{bundle_id}` | `PreviewBundleController.Get` | Get bundle detail, role states, manifest history |
| POST | `/api/v1/preview-bundles/{bundle_id}/destroy` | `PreviewBundleController.Destroy` | Mark bundle destroyed; cleanup hook remains explicit |
| POST | `/api/v1/preview-bundles/{bundle_id}/build-reports` | `PreviewBundleController.ReportBuild` | Worker-authenticated role build callback |

### API Endpoints — Worker Builds (Phase 11+)

| Method | Path | Handler | Auth | Notes |
| --- | --- | --- | --- | --- |
| POST | `/api/v1/workers/{worker_id}/builds` | `WorkerBuildController.TriggerBuild` | Session | Trigger build: `{ stack_id, role, mode }` |
| GET | `/api/v1/workers/{worker_id}/builds` | `WorkerBuildController.List` | Session | List builds ordered `created_at DESC` |
| POST | `/api/v1/workers/{worker_id}/build-reports` | `WorkerBuildController.ReportBuild` | Worker-key | Worker callback: report build result |

### API Endpoints — Deployment Plans (Phase 13+)

| Method | Path | Handler | Auth | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/v1/deployment-plans` | `DeploymentPlanController.List` | Session | List all plans for current user |
| POST | `/api/v1/deployment-plans` | `DeploymentPlanController.Create` | Session | Create and start deploying a named plan |
| GET | `/api/v1/deployment-plans/{id}` | `DeploymentPlanController.Get` | Session | Get plan with role states |
| POST | `/api/v1/deployment-plans/{id}/stop` | `DeploymentPlanController.Stop` | Session | Stop containers and remove proxy route |
| DELETE | `/api/v1/deployment-plans/{id}` | `DeploymentPlanController.Delete` | Session | Remove plan (stopped/failed only) |

### PreviewBundleService

`PreviewBundleService` manages preview-bundle state and stack-registry resolution (legacy system, retained for reference).

Responsibilities:

- load stack definitions from a file-based registry in the orchestrator repo
- create bundle rows plus one requested role row per required stack role
- validate optional per-role worker overrides at bundle creation time
- persist preselected builder worker assignment per role when explicitly chosen
- validate that worker build reports match the configured role `worker_group`
- record append-only build manifests per reported artifact
- move bundle state through `pending_build -> building -> ready_to_deploy -> failed -> destroyed`

### WorkerBuildService (Phase 11+)

`WorkerBuildService` manages per-agent image build records.

Responsibilities:

- create `worker_builds` records on trigger
- return existing `ready` build for the same `worker_id + stack_id + role` when `mode=latest`
- validate `stack_id` / `role` against the file-based stack registry before dispatch
- dispatch a synthetic build instruction to the agent via `DispatcherService`
- update build status on worker callback
- notify `DeploymentPlanService.OnBuildReady` after a terminal worker build report so pending deployment plans can react to both `ready` and `failed` results

### WorkerBuild Model (Phase 11+)

`WorkerBuild` stores one append-only image build attempt per worker.

Key fields:

- `id`
- `worker_id`
- `stack_id`
- `role`
- `build_mode` (`fresh | latest`)
- `status` (`queued | building | ready | failed`)
- `image_reference`
- `image_digest`
- `error_message`
- `triggered_by`
- `started_at`
- `completed_at`
- `created_at`

Rules:

- `mode=latest` may reuse an existing `ready` build, but only for the same `stack_id` and `role`
- synthetic build dispatch uses the worker's normal `instruction_job` prompt field plus a build-specific body
- worker callbacks must present the same `worker_id` in the path and the authenticated worker key

### DeploymentPlan Model (Phase 13+)

`DeploymentPlan` stores one named preview runtime request per user.

Key fields:

- `id`
- `user_id`
- `name` (global unique slug)
- `stack_id`
- `build_mode`
- `status` (`pending | deploying | running | failed | stopped`)
- `preview_url`
- `error`
- `stopped_at`
- `created_at`
- `updated_at`

### DeploymentPlanRole Model (Phase 13+)

`DeploymentPlanRole` stores one worker/build/runtime row per required stack role in a plan.

Key fields:

- `id`
- `plan_id`
- `role`
- `worker_id`
- `worker_build_id`
- `image_reference`
- `image_digest`
- `container_name`
- `host_port`
- `build_status` (`pending | ready | failed`)
- `deploy_status` (`pending | running | failed`)
- `created_at`
- `updated_at`

### DeploymentPlanService (Phase 13+)

`DeploymentPlanService` manages deployment plan lifecycle.

Responsibilities:

- validate plan name slug (`[a-z0-9-]{1,40}`, unique), stack, and per-role worker assignments
- resolve `latest` builds or trigger `fresh` builds per role at creation time
- execute deploy asynchronously via `ContainerService` + Docker CLI helpers in `deployment_plan_service.go`
- write and reload nginx upstream config via `NginxConfigService`
- perform blocking health probes per role after container start; the plan does not become `running` until all probes pass
- stop containers and regenerate nginx config on stop
- leave stop failures as operation errors; do not introduce a `stopping` status in Phase 13

### Docker Preview Execution (Phase 13+)

Deployment runtime currently lives inside `DeploymentPlanService`, with Docker CLI calls isolated behind `ContainerService`.

- Uses `os/exec` (not Docker SDK).
- `ContainerService` wraps `docker run`, `docker stop`, `docker rm`, `docker inspect`, and `docker exec`.
- Backend container mounts `/var/run/docker.sock`.
- Preview containers: `preview-{plan-name}-{role}`.
- Host port range: 8200+ (stored in `deployment_plan_roles.host_port`).
- Host-port allocation is database-backed: the repository locks `deployment_plan_roles`, reads all non-null `host_port` values in use, picks the next free port starting at `8200`, and stores it on the role row before container start.
- Deploy image references are translated from the compose-internal registry host (`registry:5000`) to the host-reachable registry endpoint (`localhost:5001`) before `docker run`.
- Preview containers join the network named by `DOCKER_PREVIEW_NETWORK` (default `bridge`).
- Per-container v1 env injection is limited to `PLAN_ID`, `STACK_ID`, `ROLE`, and `PREVIEW_HOSTNAME`.

### ContainerService (Phase 15+)

`ContainerService` is the dedicated preview-runtime Docker wrapper in `backend/internal/services/container_service.go`.

Responsibilities:

- launch role containers by digest reference through `docker run`
- stop and remove preview containers idempotently through `docker stop` / `docker rm -f`
- expose `docker inspect` helpers (`IsContainerRunning`, `GetContainerIP`) for future runtime checks
- execute commands in running containers through `docker exec`

Rationale:

- keeps Docker SDK dependencies out of the backend
- matches the existing `DispatcherService` subprocess pattern
- keeps all CLI error formatting in one place instead of duplicating `exec.CommandContext` logic across services

### NginxConfigService (Phase 13+)

`NginxConfigService` generates `infra/nginx/preview-upstreams.conf` and reloads nginx.

- Regenerates the entire file from all `running` deployment plans on each deploy or stop.
- Calls `docker exec proxy nginx -s reload` after write.
- Upstreams target `host.docker.internal:{host_port}`. This is a Docker Desktop for Mac requirement for the current compose topology; `127.0.0.1` inside the `proxy` or `backend` container would loop back into that container, not the Docker host.
- Routing is a pragmatic v1 `subdomain-preview` assumption owned directly by `NginxConfigService`: when both `backend` and `frontend` roles exist, `/api/` proxies to backend and `/` proxies to frontend; when only one role exists, nginx proxies every request to that single upstream.
- `infra/nginx/default.conf` includes `preview-upstreams.conf` at the top; there is no placeholder `*.preview` server block anymore.
- Backend and proxy share the same bind-mounted `preview-upstreams.conf` file; backend writes it through `NGINX_PREVIEW_CONF`.

### DispatcherService

`DispatcherService` (in `services/dispatcher_service.go`) manages all CLI subprocess execution.

Key methods:

- `SendToWorker` — launches a job asynchronously (goroutine), streams JSONL output, stores messages.
- `RunBuildInstruction(ctx, workerID, buildID, message) error` — synchronous synthetic build execution path used by `WorkerBuildService`; does not create normal job/message records.
- `RunPlan(ctx, workerID, task) (string, error)` — **synchronous** (2-minute timeout). Runs the worker's CLI with a prompt-refinement instruction. Returns the final text. Used by `WorkerController.Plan`.
- `RunPlanStream(ctx, w, workerID, resumeID, message) (reply, newResumeID, error)` — **streaming**. Runs the worker's CLI and writes SSE events to `w` as the agent responds. Used by `PlanSessionController` for discussion turns and plan generation.

### Frontend Stores

| Store | File | Type | Purpose |
| --- | --- | --- | --- |
| `allWorkers` | `stores/workers.ts` | `Worker[]` | All registered workers, populated via WS + REST |
| `selectedWorker` | `stores/selectedWorker.ts` | `Worker \| null` | Currently viewed agent detail |
| `selectedGroup` | `stores/selectedGroup.ts` | `string` | Active workspace group; persists across navigation |
| `jobs` | `stores/jobs.ts` | `Job[]` | All jobs; updated via `upsertJob` |

### Frontend Routes

| Route | File | Purpose |
| --- | --- | --- |
| `/` | `routes/(app)/+page.svelte` | Agents page — card grid filtered by selected workspace |
| `/agents/[worker_id]/jobs` | `routes/(app)/agents/[worker_id]/jobs/+page.svelte` | Job list for a specific agent |
| `/agents/[worker_id]/settings` | `routes/(app)/agents/[worker_id]/settings/+page.svelte` | Agent settings + health check |
| `/plans` | `routes/(app)/plans/+page.svelte` | Plans page — discussion-first job creation |
| `/office` | `routes/(app)/office/+page.svelte` | Three.js open-floor cyberpunk Office scene with HUD overlay, avatar movement, click selection, and worker interaction panel |
| `/jobs/[job_id]` | `routes/(app)/jobs/[job_id]/+page.svelte` | Job chat feed |
| `/deploys` | `routes/(app)/deploys/+page.svelte` | Deployment plans page — create, view, stop, delete plans (Phase 14+) |

Shared layout `routes/(app)/+layout.svelte` renders the persistent top bar (logo, workspace dropdown, Agents \| Plans \| Office \| Deploys nav) for all app routes.

Shared layout `routes/(app)/agents/[worker_id]/+layout.svelte` renders the agent header and Jobs \| Settings sub-nav for agent sub-pages.

### Office 3D Modules

Office rendering under `frontend/src/lib/office/` is centered on one scene module plus small supporting data/input modules:

- `OfficeScene.ts` — Three.js renderer lifecycle, scene graph, locked follow camera, avatar movement, desk collision, raycast hover/click interaction, and animation
- `mapConfig.ts` — logical desk-grid layout, zone metadata, desk configuration, fallback desk resolution, and world-space mapping
- `playerController.ts` — keyboard input state with normalized X/Z movement control

Interaction UI is implemented in `frontend/src/components/organisms/OfficeInteractionPanel.svelte` and reuses existing job APIs (`POST /jobs`, `POST /jobs/{id}/submit`) without adding a new worker-jobs endpoint.

The Office route is a documented visual exception to the global NEXUS theme. Its source of truth is `office-demo.html`, so Office-specific components intentionally use the brighter open-floor cyan / magenta / purple cyberpunk interface from that prototype while the rest of the app remains on the standard NEXUS design system.

### Stack Registry

Previewable app stacks are defined in a file-based registry stored with the backend codebase, not in the database.

Each stack entry defines:

- `stack_id`
- `display_name`
- `deployment_template`
- required roles such as `frontend` and `backend`
- per-role worker group mapping
- per-role service name
- per-role healthcheck path
- required image labels

This keeps preview bundle assembly versioned with orchestrator code while avoiding a mutable singleton config table.

### Frontend Preview Flow

The canonical preview flow is agent-centric, not job-centric.

**Build trigger (Phase 12+):**

- agent card shows **Build** button when the agent has an active job
- active-job detection is derived from jobs assigned to that worker with status `assigned`, `busy`, or `pending_user`; it does not rely on `worker.status === 'assigned'`
- click opens mode picker: Fresh (force rebuild) or Latest (reuse last ready build)
- confirm calls `POST /api/v1/workers/{id}/builds`
- card updates to show current build status badge and last image info
- Phase 12 frontend uses a temporary hardcoded mapping:
  - `fi-backend` → `{ stack_id: 'fi-web-app', role: 'backend' }`
  - `fi-frontend` → `{ stack_id: 'fi-web-app', role: 'frontend' }`
  - unmapped groups keep the button disabled until explicit stack selection exists

**Deployment plan (Phase 14+):**

- user goes to `/deploys` page
- clicks **New Plan**
- modal: name (slug), stack, build mode, one agent per role
- submit calls `POST /api/v1/deployment-plans`
- plan card shows status, preview URL when running, role-level build + deploy badges
- frontend files:
  - `routes/(app)/deploys/+page.svelte`
  - `components/organisms/CreateDeployPlanModal.svelte`
  - `components/molecules/DeployPlanCard.svelte`

### Preview URL Convention

Each deployment plan has one preview URL keyed by plan name:

```text
http://shiphide.{plan-name}.preview
```

- `{plan-name}` is the user-supplied slug (`[a-z0-9-]{1,40}`)
- multiple plans can be `running` simultaneously at distinct URLs
- each plan's URL is unique — there is no per-agent URL in the deployment plan system
- access requires a hosts-file entry per device per plan name: `127.0.0.1 shiphide.{plan-name}.preview`
- optional: install host-based dnsmasq for automatic wildcard `*.preview` → `127.0.0.1` on Mac

### Deployment Networking (Compose)

Local/LAN compose topology:

| Service | Host port | Internal port | Notes |
| --- | --- | --- | --- |
| `proxy` (nginx) | `5174:80`, `80:80` | 80 | Main app + preview routing |
| `registry` (OCI) | `5001:5000` | 5000 | Phase 11+; named volume `registrydata`; workers push to `registry:5000` |
| `backend` | — | 8080 | Mounts `/var/run/docker.sock` and shared `preview-upstreams.conf`; writes nginx preview config through `NGINX_PREVIEW_CONF`; health-checks preview roles through `host.docker.internal:{host_port}` |
| `frontend` | — | 5173 | |
| `postgres` | — | 5432 | |

Additional backend mounts (Phase 11+):
- `/var/run/docker.sock:/var/run/docker.sock` — required for worker-triggered `docker build` / `docker push` and later container lifecycle management
- `./infra/nginx/preview-upstreams.conf:/app/infra/nginx/preview-upstreams.conf` — shared writable preview route file used by backend
- `./infra/nginx/preview-upstreams.conf:/etc/nginx/conf.d/preview-upstreams.conf` — same shared file mounted into `proxy` so nginx reloads the latest generated preview config

Routing:

- browser `GET /` → `proxy` → `frontend:5173`
- browser `GET/POST /api/*` → `proxy` → `backend:8080/api/*`
- browser `WS /ws` → `proxy` → `backend:8080/ws`
- browser `GET http://shiphide.{plan}.preview/` → `proxy` → generated upstream in `preview-upstreams.conf` → `host.docker.internal:{host_port}`

DNS note:

- v1 still supports manual hosts-file entries per plan
- optional Mac setup: install host-based dnsmasq so any `*.preview` hostname resolves to `127.0.0.1`
- the compose-based preview runtime itself still depends on Docker Desktop exposing `host.docker.internal` inside containers

Frontend should keep `VITE_BACKEND_URL` empty in this mode so API calls resolve to same-origin `/api/v1`.

---

## Checklist for New Projects

- [ ] Set up project structure (backend/frontend folders)
- [ ] Configure DI container in main.go
- [ ] Create base migration (users, sessions tables)
- [ ] Implement auth flow (login, logout, session middleware)
- [ ] Set up API client in frontend
- [ ] Create auth store in frontend
- [ ] Configure CORS for frontend origin
- [ ] Set up rate limiting middleware
- [ ] Configure soft delete pattern in repositories
