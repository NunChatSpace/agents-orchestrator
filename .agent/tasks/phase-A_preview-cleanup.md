# Phase A — Delete Old Preview Files & Clean Up References

## Goal
Remove all files and references from the old image-build preview approach (phases 11–15). Leave the repo in a clean state with no references to WorkerBuild, DeploymentPlan, ContainerService, or OCI registry.

---

## STEP 1 — Delete untracked files (safe to rm)

Delete these files:
- `backend/internal/controllers/worker_build_controller.go`
- `backend/internal/controllers/deployment_plan_controller.go`
- `backend/internal/domains/worker_build.go`
- `backend/internal/domains/deployment_plan.go`
- `backend/internal/models/worker_build.go`
- `backend/internal/models/deployment_plan.go`
- `backend/internal/repository/worker_build_repo.go`
- `backend/internal/repository/deployment_plan_repo.go`
- `backend/internal/services/worker_build_service.go`
- `backend/internal/services/deployment_plan_service.go`
- `backend/internal/services/container_service.go`
- `backend/migrations/012_worker_builds.sql`
- `backend/migrations/013_deployment_plans.sql`
- `backend/migrations/014_worker_builds_callback_token.sql`
- `backend/migrations/015_worker_builds_build_log.sql`
- `backend/migrations/016_worker_build_command.sql`
- `frontend/src/components/molecules/DeployPlanCard.svelte`
- `frontend/src/components/molecules/WorkerBuildStatus.svelte`
- `frontend/src/components/organisms/CreateDeployPlanModal.svelte`
- `frontend/src/lib/apis/deploymentPlans.ts`
- `frontend/src/lib/apis/workerBuilds.ts`
- `frontend/src/types/deploymentPlan.ts`
- `frontend/src/types/workerBuild.ts`
- `frontend/src/routes/(app)/deploys/` (entire directory)

---

## STEP 2 — Clean up backend modified files

For each file below: read it, remove ONLY references to the old approach (WorkerBuild, DeploymentPlan, ContainerService, registry, RunBuildInstruction, worker_build_service, deployment_plan_service). Do not touch unrelated code.

### `backend/internal/app/container.go`
Remove DI wiring for: WorkerBuildRepository, WorkerBuildService, DeploymentPlanRepository, DeploymentPlanService, ContainerService, NginxConfigService (old version).

### `backend/internal/app/routes.go`
Remove route registrations for:
- `/workers/{id}/builds` (GET, POST)
- `/workers/{id}/build-reports` (POST)
- `/deployment-plans` (GET, POST)
- `/deployment-plans/{id}` (GET, DELETE)
- `/deployment-plans/{id}/stop` (POST)

### `backend/internal/repository/interfaces.go`
Remove interface declarations for WorkerBuildRepository and DeploymentPlanRepository if they were added here.

### `backend/internal/services/dispatcher_service.go`
Remove `RunBuildInstruction` method if it was added. Keep: `SendToWorker`, `RunPlan`, `RunPlanStream`.

### `backend/internal/services/worker_service.go`
Remove any build-related methods added for old approach.

### `backend/internal/services/job_service.go`
Remove any build-related additions.

### `backend/internal/domains/worker.go`
Remove fields: `BuildCommand`, `LastBuildStatus`, `LastBuildImage`, or any build-related fields added.

### `backend/internal/models/worker.go`
Remove fields: `BuildCommand`, `LastBuildID`, or any build-related db-tagged fields added.

### `backend/internal/repository/worker_repo.go`
Remove build-related columns from SELECT/UPDATE queries if added.

### `docker-compose.yml`
Remove the `registry` service block and `registrydata` volume.

---

## STEP 3 — Clean up frontend modified files

### `frontend/src/types/worker.ts`
Remove build-related fields (`build_command`, `last_build_status`, `last_build_image`, etc.) if added.

### `frontend/src/routes/(app)/+layout.svelte`
Remove the "Deploys" nav link if it was added pointing to the old deploys route. (It will be re-added in Phase E for the new route.)

### `frontend/src/routes/(app)/+page.svelte`
Remove the Build button and any build-mode picker that calls `POST /api/v1/workers/{id}/builds`.

### `frontend/src/routes/(app)/agents/[worker_id]/settings/+page.svelte`
Remove `build_command` field and any build-status UI if added.

---

## STEP 4 — Clean up nginx

### `infra/nginx/default.conf`
Remove the `include /etc/nginx/conf.d/preview-upstreams.conf;` line if it was added (it will be re-added correctly in Phase E).

---

## Acceptance Criteria

1. `go build ./...` from `backend/` compiles without errors.
2. No file in the repo references `WorkerBuild`, `DeploymentPlan`, `ContainerService`, `RunBuildInstruction`, `worker_build_service`, `deployment_plan_service`, or `registry:5000`.
3. `docker-compose.yml` has no `registry` service.
4. No deleted frontend files are imported anywhere.
