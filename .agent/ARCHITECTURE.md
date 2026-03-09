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

### DispatcherService

`DispatcherService` (in `services/dispatcher_service.go`) manages all CLI subprocess execution.

Key methods:

- `SendToWorker` — launches a job asynchronously (goroutine), streams JSONL output, stores messages.
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

Shared layout `routes/(app)/+layout.svelte` renders the persistent top bar (logo, workspace dropdown, Agents \| Plans \| Office nav) for all app routes.

Shared layout `routes/(app)/agents/[worker_id]/+layout.svelte` renders the agent header and Jobs \| Settings sub-nav for agent sub-pages.

### Office 3D Modules

Office rendering under `frontend/src/lib/office/` is centered on one scene module plus small supporting data/input modules:

- `OfficeScene.ts` — Three.js renderer lifecycle, scene graph, locked follow camera, avatar movement, desk collision, raycast hover/click interaction, and animation
- `mapConfig.ts` — logical desk-grid layout, zone metadata, desk configuration, fallback desk resolution, and world-space mapping
- `playerController.ts` — keyboard input state with normalized X/Z movement control

Interaction UI is implemented in `frontend/src/components/organisms/OfficeInteractionPanel.svelte` and reuses existing job APIs (`POST /jobs`, `POST /jobs/{id}/submit`) without adding a new worker-jobs endpoint.

The Office route is a documented visual exception to the global NEXUS theme. Its source of truth is `office-demo.html`, so Office-specific components intentionally use the brighter open-floor cyan / magenta / purple cyberpunk interface from that prototype while the rest of the app remains on the standard NEXUS design system.

### Deployment Networking (Compose)

Local/LAN compose topology uses a single host-exposed port:

- `proxy` (nginx) exposes `5174:80` on host
- `frontend` has no host port mapping (internal only)
- `backend` has no host port mapping (internal only)
- `postgres` has no host port mapping (internal only)

Routing:

- browser `GET /` -> `proxy` -> `frontend:5173`
- browser `GET/POST /api/*` -> `proxy` -> `backend:8080/api/*`
- browser `WS /ws` -> `proxy` -> `backend:8080/ws`

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
