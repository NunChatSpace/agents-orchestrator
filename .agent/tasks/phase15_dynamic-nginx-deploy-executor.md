# Phase 15 — Dynamic Nginx + Deploy Executor

## Goal

Complete the deployment executor: spin up containers from image digests, wire nginx routing dynamically, and run health checks until the plan is `running` or `failed`.

After this phase:
- A deployed plan actually starts Docker containers using the specified images.
- Nginx is updated with upstream + server blocks for each running plan.
- Health checks confirm containers are up before marking plan `running`.
- Stopping a plan tears down containers and removes its nginx entries.
- `http://shiphide.{plan-name}.preview` serves the deployed app.

---

## Spec Reference

`.agent/specs/workspace-preview-runtime-spec-v1.md` §7.4, §9.3, §13, §15

---

## Prerequisites

Phases 11–14 complete. Registry is running. At least one agent has a ready build.

---

## Affected Files

| File | Change |
|---|---|
| `backend/internal/services/deployment_plan_service.go` | Complete deploy + stop execution |
| `backend/internal/services/nginx_config_service.go` | Complete nginx write + reload |
| `backend/internal/services/container_service.go` | New — Docker SDK or exec wrapper |
| `docker-compose.yml` | Confirm Docker socket mount on backend; confirm `preview-upstreams.conf` mount on proxy |
| `infra/nginx/default.conf` | Add `include` for generated upstreams file |
| `infra/nginx/preview-upstreams.conf` | Initial empty file (committed to repo) |
| `.agent/ARCHITECTURE.md` | Document container service and exec strategy |

---

## Container Service: `backend/internal/services/container_service.go`

This service wraps Docker operations. Use `os/exec` to call `docker` CLI commands rather than the Docker SDK to keep dependencies minimal.

Interface:

```go
type ContainerService interface {
    RunContainer(ctx context.Context, opts RunContainerOptions) (containerName string, err error)
    StopContainer(ctx context.Context, name string) error
    RemoveContainer(ctx context.Context, name string) error
    IsContainerRunning(ctx context.Context, name string) (bool, error)
    GetContainerIP(ctx context.Context, name string) (string, error)
    ExecInContainer(ctx context.Context, name string, cmd []string) (string, error)
}

type RunContainerOptions struct {
    Name        string
    Image       string            // must be a digest ref: registry:5000/...@sha256:...
    Env         map[string]string
    NetworkName string            // Docker network to join
    Labels      map[string]string
}
```

Naming convention for preview containers:
- `preview-{plan-name}-{role}` e.g. `preview-feature-cart-backend`

Network convention:
- Each plan gets its own Docker network: `preview-{plan-name}`
- Create network before starting containers; remove after all containers are stopped

Port allocation:
- For v1, the nginx upstream uses the container IP (not host port). All preview containers join the same Docker network as `proxy`.
- Alternatively: allocate a free host port per role starting from 8100. Store in `deployment_plan_roles.host_port`. Nginx upstream uses `127.0.0.1:{host_port}`.
- Recommended for simplicity: use host port allocation (no cross-network routing complexity).

Host port allocation:
- Query `deployment_plan_roles` for all `host_port` values currently in use.
- Pick next free port starting from 8200.
- Store in `deployment_plan_roles.host_port` before starting the container.

---

## Deploy Execution: `DeploymentPlanService.executeDeploy`

Called asynchronously after all role builds are `ready`. Runs in a goroutine.

Steps:

1. For each role in the plan:
   a. Determine `image_reference` (already stored on the role record as `image_reference` + `image_digest`).
   b. Build full digest ref: `{image_reference}@{image_digest}`.
   c. Allocate a free host port.
   d. Call `containerService.RunContainer` with:
      - `Name`: `preview-{plan-name}-{role}`
      - `Image`: digest ref
      - `Env`: inject `PLAN_ID`, `STACK_ID`, `ROLE`, `PREVIEW_HOSTNAME`, etc.
      - Plus any role-specific env from the stack registry `deployment_template`.
   e. Update `deployment_plan_roles` record with `container_name` and `host_port`.

2. After all containers started: call `nginxConfigService.WriteAndReload`.

3. Start health check loop:
   - For each role, call `GET http://localhost:{host_port}{healthcheck_path}`.
   - Retry every 3 seconds, up to 20 attempts (60s total).
   - All roles must pass health check before marking plan `running`.
   - If any role fails all retries: mark plan `failed`, stop all containers.

4. On success: update plan `status=running`, set `preview_url`.

---

## Stop Execution: `DeploymentPlanService.executeStop`

Steps:

1. For each role:
   a. Call `containerService.StopContainer(containerName)`.
   b. Call `containerService.RemoveContainer(containerName)`.
   c. Update role `deploy_status=pending`.

2. Call `nginxConfigService.WriteAndReload` (regenerates file without this plan's blocks).

3. Update plan `status=stopped`, set `stopped_at`.

---

## Nginx Config Service: Complete Implementation

`WriteAndReload`:

1. Load all `running` deployment plans with their roles from the repository.
2. Build the config string:
   - One `upstream preview_{plan_slug}` block per role using `server 127.0.0.1:{host_port};`
   - One `server` block per plan proxying to its role containers
3. Write to `NGINX_PREVIEW_CONF` path.
4. Run `docker exec proxy nginx -s reload` via `ContainerService.ExecInContainer` or `os/exec`.

Template for one plan with two roles:

```nginx
# Plan: feature-cart
upstream preview_feature_cart_backend {
    server 127.0.0.1:8201;
}
upstream preview_feature_cart_frontend {
    server 127.0.0.1:8200;
}
server {
    listen 80;
    server_name shiphide.feature-cart.preview;

    location /api/ {
        proxy_pass http://preview_feature_cart_backend/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location / {
        proxy_pass http://preview_feature_cart_frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

Note: the routing split (`/api/` → backend, `/` → frontend) is stack-specific and must come from the stack registry `deployment_template` field. For the current `fi-web-app` stack, this is the expected split.

---

## `infra/nginx/default.conf` Changes

Add include at the very top of the file (before any server block):

```nginx
include /etc/nginx/conf.d/preview-upstreams.conf;
```

Remove the existing placeholder `*.preview` server block (already contains `return 200 "preview: $agent\n"`).

---

## `infra/nginx/preview-upstreams.conf`

Commit this empty file:

```nginx
# AUTO-GENERATED — do not edit manually
# Managed by DeploymentPlanService. Rewritten on each deploy/stop.
```

Must be non-empty so nginx does not error on include. The comment lines are safe.

---

## Environment Variables

Add to `docker-compose.yml` backend service:

```yaml
NGINX_PREVIEW_CONF: /workspaces/../infra/nginx/preview-upstreams.conf
DOCKER_PREVIEW_NETWORK: bridge
PREVIEW_BASE_DOMAIN: shiphide
PREVIEW_TLD: preview
HEALTH_CHECK_RETRIES: 20
HEALTH_CHECK_INTERVAL_SECONDS: 3
```

Adjust `NGINX_PREVIEW_CONF` path to wherever the file is mounted into the backend container. The proxy and backend must share the same file via a volume mount.

---

## DNS Setup Guide (Host-Based dnsmasq — Optional)

To avoid adding a hosts entry per plan name on the Mac, install dnsmasq on the host:

```bash
brew install dnsmasq
echo "address=/.preview/127.0.0.1" >> /opt/homebrew/etc/dnsmasq.conf
sudo brew services start dnsmasq
sudo mkdir -p /etc/resolver
echo "nameserver 127.0.0.1" | sudo tee /etc/resolver/preview
```

After this, any `*.preview` domain resolves to `127.0.0.1` on the Mac automatically. No per-plan hosts entries needed.

For Windows (LAN access from another device): still requires manual hosts entries per plan name pointing to the Mac's LAN IP (`192.168.1.170 shiphide.{plan-name}.preview`).

This guide should live in `infra/README.md` or the project CLAUDE.md.

---

## ARCHITECTURE.md Updates Required

- Document `ContainerService` and exec strategy (Docker CLI, not SDK).
- Document host-port allocation range (8200+).
- Document nginx include approach and generated file path.
- Document dnsmasq optional setup.
- Update Compose topology to confirm final service list: `postgres`, `registry`, `backend`, `frontend`, `proxy`.

---

## Acceptance Criteria

1. `POST /api/v1/deployment-plans` with all-ready builds transitions plan to `deploying` → `running`.
2. Containers named `preview-{plan-name}-{role}` are visible in `docker ps`.
3. `http://shiphide.{plan-name}.preview` (with hosts entry) serves the deployed app.
4. Multiple plans can run simultaneously at distinct URLs.
5. `POST /deployment-plans/{id}/stop` removes containers; `docker ps` no longer shows them.
6. After stop, nginx is reloaded; the stopped plan's subdomain returns connection refused or nginx 404.
7. A second plan started while the first is running does not break the first.
8. Health check failure after container start marks plan `failed` and stops containers.
9. `infra/nginx/preview-upstreams.conf` is updated on every deploy and stop.
