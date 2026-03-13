package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/stackregistry"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

var (
	ErrDeploymentPlanNotFound      = errors.New("deployment plan not found")
	ErrDeploymentPlanDuplicateName = errors.New("deployment plan name already exists")
	ErrDeploymentPlanValidation    = errors.New("deployment plan validation failed")
	ErrDeploymentPlanInvalidState  = errors.New("deployment plan invalid state")
)

const previewHostPortStart = 8200

var deploymentPlanNamePattern = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

type DeploymentPlanDetail struct {
	Plan  *models.DeploymentPlan
	Roles []*models.DeploymentPlanRole
}

type DeploymentPlanService interface {
	Create(ctx context.Context, userID uuid.UUID, req domains.CreateDeploymentPlanRequest) (*DeploymentPlanDetail, error)
	Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error)
	List(ctx context.Context, userID uuid.UUID) ([]*DeploymentPlanDetail, error)
	Stop(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error)
	Retry(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error)
	Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
	OnBuildReady(ctx context.Context, workerBuildID uuid.UUID) error
}

type deploymentPlanService struct {
	repo                 repository.DeploymentPlanRepository
	workerRepo           repository.WorkerRepository
	buildRepo            repository.WorkerBuildRepository
	buildSvc             WorkerBuildService
	registry             *stackregistry.Registry
	containerSvc         ContainerService
	nginx                NginxConfigService
	httpClient           *http.Client
	dockerPreviewNetwork string
	baseDomain           string
	tld                  string
	healthCheckRetries   int
	healthCheckInterval  time.Duration
}

func NewDeploymentPlanService(
	repo repository.DeploymentPlanRepository,
	workerRepo repository.WorkerRepository,
	buildRepo repository.WorkerBuildRepository,
	buildSvc WorkerBuildService,
	registry *stackregistry.Registry,
	containerSvc ContainerService,
	nginx NginxConfigService,
) DeploymentPlanService {
	return &deploymentPlanService{
		repo:                 repo,
		workerRepo:           workerRepo,
		buildRepo:            buildRepo,
		buildSvc:             buildSvc,
		registry:             registry,
		containerSvc:         containerSvc,
		nginx:                nginx,
		httpClient:           &http.Client{Timeout: 2 * time.Second},
		dockerPreviewNetwork: dockerPreviewNetwork(),
		baseDomain:           previewBaseDomain(),
		tld:                  previewTLD(),
		healthCheckRetries:   healthCheckRetries(),
		healthCheckInterval:  healthCheckInterval(),
	}
}

func (s *deploymentPlanService) Create(ctx context.Context, userID uuid.UUID, req domains.CreateDeploymentPlanRequest) (*DeploymentPlanDetail, error) {
	name := strings.TrimSpace(req.Name)
	stackID := strings.TrimSpace(req.StackID)
	buildMode := strings.TrimSpace(req.BuildMode)

	if !deploymentPlanNamePattern.MatchString(name) {
		return nil, validationError("name must match ^[a-z0-9-]{1,40}$")
	}
	if !models.IsWorkerBuildMode(buildMode) {
		return nil, validationError("build_mode must be fresh or latest")
	}
	if s.registry == nil {
		return nil, fmt.Errorf("stack registry unavailable")
	}

	if existing, err := s.repo.GetByName(ctx, name); err != nil {
		return nil, fmt.Errorf("check deployment plan name: %w", err)
	} else if existing != nil {
		return nil, fmt.Errorf("%w: %s", ErrDeploymentPlanDuplicateName, name)
	}

	def, ok := s.registry.Get(stackID)
	if !ok {
		return nil, validationError("stack not found")
	}
	if def.DeploymentTemplate != "subdomain-preview" {
		return nil, validationError(fmt.Sprintf("deployment template %q is not supported", def.DeploymentTemplate))
	}

	requestedRoles, err := normalizeRoleInputs(req.Roles)
	if err != nil {
		return nil, err
	}
	if len(requestedRoles) != len(def.Roles) {
		return nil, validationError("roles must match every required stack role exactly once")
	}

	now := time.Now().UTC()
	plan := &models.DeploymentPlan{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		StackID:   stackID,
		BuildMode: models.WorkerBuildMode(buildMode),
		Status:    models.DeploymentPlanStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	roles := make([]*models.DeploymentPlanRole, 0, len(def.Roles))
	for _, roleDef := range def.Roles {
		input, exists := requestedRoles[roleDef.Role]
		if !exists {
			return nil, validationError(fmt.Sprintf("missing required role %q", roleDef.Role))
		}

		workerID, err := uuid.Parse(input.WorkerID)
		if err != nil {
			return nil, validationError(fmt.Sprintf("invalid worker_id for role %q", roleDef.Role))
		}

		worker, err := s.workerRepo.GetByID(ctx, workerID)
		if err != nil {
			return nil, validationError(fmt.Sprintf("worker not found for role %q", roleDef.Role))
		}
		if worker.GroupName != roleDef.WorkerGroup {
			return nil, validationError(
				fmt.Sprintf("worker group %q cannot be assigned to role %q (expected group %q)", worker.GroupName, roleDef.Role, roleDef.WorkerGroup),
			)
		}

		role := &models.DeploymentPlanRole{
			ID:           uuid.New(),
			PlanID:       plan.ID,
			Role:         roleDef.Role,
			WorkerID:     workerID,
			BuildStatus:  models.DeploymentPlanRoleBuildStatusPending,
			DeployStatus: models.DeploymentPlanRoleDeployStatusPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		switch models.WorkerBuildMode(buildMode) {
		case models.WorkerBuildModeLatest:
			build, err := s.buildRepo.LatestReadyForWorkerRole(ctx, workerID, stackID, roleDef.Role)
			if err != nil {
				return nil, fmt.Errorf("lookup latest ready build for role %q: %w", roleDef.Role, err)
			}
			if build != nil {
				role.WorkerBuildID = uuidPtr(build.ID)
				role.ImageReference = build.ImageReference
				role.ImageDigest = build.ImageDigest
				role.BuildStatus = models.DeploymentPlanRoleBuildStatusReady
				break
			}

			build, err = s.buildSvc.TriggerBuild(ctx, workerID, stackID, roleDef.Role, string(models.WorkerBuildModeFresh), userID)
			if err != nil {
				return nil, wrapBuildTriggerError(roleDef.Role, err)
			}
			role.WorkerBuildID = uuidPtr(build.ID)
		case models.WorkerBuildModeFresh:
			build, err := s.buildSvc.TriggerBuild(ctx, workerID, stackID, roleDef.Role, string(models.WorkerBuildModeFresh), userID)
			if err != nil {
				return nil, wrapBuildTriggerError(roleDef.Role, err)
			}
			role.WorkerBuildID = uuidPtr(build.ID)
		}

		roles = append(roles, role)
	}

	if err := s.repo.Create(ctx, plan, roles); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: %s", ErrDeploymentPlanDuplicateName, name)
		}
		return nil, fmt.Errorf("create deployment plan: %w", err)
	}

	for _, role := range roles {
		if role.WorkerBuildID == nil || role.BuildStatus != models.DeploymentPlanRoleBuildStatusPending {
			continue
		}

		build, err := s.buildRepo.GetByID(ctx, *role.WorkerBuildID)
		if err != nil {
			continue
		}
		if build.Status == models.WorkerBuildStatusReady || build.Status == models.WorkerBuildStatusFailed {
			if err := s.OnBuildReady(ctx, build.ID); err != nil {
				log.Error().Err(err).Str("deployment_plan_id", plan.ID.String()).Str("worker_build_id", build.ID.String()).Msg("sync deployment plan after immediate build completion failed")
			}
		}
	}

	if allRolesReady(roles) {
		go s.executeDeploy(plan.ID)
	}

	return &DeploymentPlanDetail{Plan: plan, Roles: roles}, nil
}

func (s *deploymentPlanService) Get(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error) {
	return s.loadOwnedPlanDetail(ctx, userID, id)
}

func (s *deploymentPlanService) List(ctx context.Context, userID uuid.UUID) ([]*DeploymentPlanDetail, error) {
	plans, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list deployment plans: %w", err)
	}

	result := make([]*DeploymentPlanDetail, 0, len(plans))
	for _, plan := range plans {
		detail, err := s.loadOwnedPlanDetail(ctx, userID, plan.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, detail)
	}
	return result, nil
}

func (s *deploymentPlanService) Stop(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error) {
	detail, err := s.loadOwnedPlanDetail(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if detail.Plan.Status == models.DeploymentPlanStatusStopped {
		return nil, invalidStateError("deployment plan is already stopped")
	}

	if err := s.executeStop(ctx, detail); err != nil {
		return nil, err
	}
	return detail, nil
}

func (s *deploymentPlanService) Retry(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*DeploymentPlanDetail, error) {
	detail, err := s.loadOwnedPlanDetail(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if detail.Plan.Status != models.DeploymentPlanStatusFailed && detail.Plan.Status != models.DeploymentPlanStatusPending {
		return nil, invalidStateError("retry is only allowed for failed or pending deployment plans")
	}

	// Reset plan to pending.
	if err := s.repo.UpdateStatus(ctx, id, models.DeploymentPlanStatusPending, nil, nil); err != nil {
		return nil, fmt.Errorf("reset deployment plan status: %w", err)
	}
	detail.Plan.Status = models.DeploymentPlanStatusPending
	detail.Plan.Error = nil

	// Re-trigger builds for all roles, resetting their state.
	for _, role := range detail.Roles {
		build, err := s.buildSvc.TriggerBuild(ctx, role.WorkerID, detail.Plan.StackID, role.Role, string(models.WorkerBuildModeFresh), userID)
		if err != nil {
			errMsg := fmt.Sprintf("retry trigger build for role %q: %v", role.Role, err)
			_ = s.repo.UpdateStatus(ctx, id, models.DeploymentPlanStatusFailed, nil, &errMsg)
			return nil, fmt.Errorf("%s", errMsg)
		}
		role.WorkerBuildID = uuidPtr(build.ID)
		role.BuildStatus = models.DeploymentPlanRoleBuildStatusPending
		role.DeployStatus = models.DeploymentPlanRoleDeployStatusPending
		role.ImageReference = nil
		role.ImageDigest = nil
		role.HostPort = nil
		role.ContainerName = nil
		if err := s.repo.UpdateRole(ctx, role); err != nil {
			return nil, fmt.Errorf("reset deployment role %q: %w", role.Role, err)
		}
	}

	return s.loadOwnedPlanDetail(ctx, userID, id)
}

func (s *deploymentPlanService) Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	detail, err := s.loadOwnedPlanDetail(ctx, userID, id)
	if err != nil {
		return err
	}
	if detail.Plan.Status != models.DeploymentPlanStatusStopped && detail.Plan.Status != models.DeploymentPlanStatusFailed {
		return invalidStateError("deployment plan must be stopped or failed before delete")
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete deployment plan: %w", err)
	}
	return nil
}

func (s *deploymentPlanService) OnBuildReady(ctx context.Context, workerBuildID uuid.UUID) error {
	build, err := s.buildRepo.GetByID(ctx, workerBuildID)
	if err != nil {
		return fmt.Errorf("worker build not found: %w", err)
	}

	waitingRoles, err := s.repo.ListRolesByWorkerBuildID(ctx, workerBuildID)
	if err != nil {
		return fmt.Errorf("list deployment plan roles by worker build: %w", err)
	}

	for _, waitingRole := range waitingRoles {
		plan, roles, err := s.repo.GetByID(ctx, waitingRole.PlanID)
		if err != nil {
			return fmt.Errorf("load deployment plan %s: %w", waitingRole.PlanID.String(), err)
		}
		if plan.Status != models.DeploymentPlanStatusPending {
			continue
		}

		role := findDeploymentRole(roles, waitingRole.ID)
		if role == nil {
			continue
		}

		switch build.Status {
		case models.WorkerBuildStatusReady:
			role.BuildStatus = models.DeploymentPlanRoleBuildStatusReady
			role.ImageReference = build.ImageReference
			role.ImageDigest = build.ImageDigest
			if err := s.repo.UpdateRole(ctx, role); err != nil {
				return fmt.Errorf("update deployment role %q: %w", role.Role, err)
			}
			if allRolesReady(roles) {
				go s.executeDeploy(plan.ID)
			}
		case models.WorkerBuildStatusFailed:
			role.BuildStatus = models.DeploymentPlanRoleBuildStatusFailed
			if err := s.repo.UpdateRole(ctx, role); err != nil {
				return fmt.Errorf("update failed deployment role %q: %w", role.Role, err)
			}
			errMsg := buildFailureMessage(build, role.Role)
			if err := s.repo.UpdateStatus(ctx, plan.ID, models.DeploymentPlanStatusFailed, nil, &errMsg); err != nil {
				return fmt.Errorf("mark deployment plan failed: %w", err)
			}
		}
	}

	return nil
}

func (s *deploymentPlanService) executeDeploy(planID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	detail, err := s.loadPlanDetail(ctx, planID)
	if err != nil {
		log.Error().Err(err).Str("deployment_plan_id", planID.String()).Msg("load deployment plan for deploy failed")
		return
	}
	if detail.Plan.Status != models.DeploymentPlanStatusPending {
		return
	}

	if err := s.repo.UpdateStatus(ctx, planID, models.DeploymentPlanStatusDeploying, nil, nil); err != nil {
		log.Error().Err(err).Str("deployment_plan_id", planID.String()).Msg("mark deployment plan deploying failed")
		return
	}
	detail.Plan.Status = models.DeploymentPlanStatusDeploying
	detail.Plan.Error = nil

	previewURL := buildPreviewURL(s.baseDomain, detail.Plan.Name, s.tld)
	previewHostname := buildPreviewHostname(s.baseDomain, detail.Plan.Name, s.tld)

	for _, role := range detail.Roles {
		if role.BuildStatus != models.DeploymentPlanRoleBuildStatusReady {
			s.failDeployment(ctx, detail, fmt.Sprintf("role %q is not ready to deploy", role.Role))
			return
		}
		if role.ImageReference == nil || role.ImageDigest == nil {
			s.failDeployment(ctx, detail, fmt.Sprintf("role %q is missing image metadata", role.Role))
			return
		}

		containerName := previewContainerName(detail.Plan.Name, role.Role)
		if err := s.containerSvc.StopContainer(ctx, containerName); err != nil {
			s.failDeployment(ctx, detail, fmt.Sprintf("stop stale container %q: %v", containerName, err))
			return
		}
		if err := s.containerSvc.RemoveContainer(ctx, containerName); err != nil {
			s.failDeployment(ctx, detail, fmt.Sprintf("remove stale container %q: %v", containerName, err))
			return
		}

		hostPort, err := s.repo.AllocateNextHostPort(ctx, role.ID, previewHostPortStart)
		if err != nil {
			s.failDeployment(ctx, detail, fmt.Sprintf("allocate host port for role %q: %v", role.Role, err))
			return
		}
		role.HostPort = intPtr(hostPort)

		containerPort, err := roleContainerPort(role.Role)
		if err != nil {
			s.failDeployment(ctx, detail, err.Error())
			return
		}

		imageRef := deployImageReference(*role.ImageReference, *role.ImageDigest)
		env := map[string]string{
			"PLAN_ID":          detail.Plan.ID.String(),
			"STACK_ID":         detail.Plan.StackID,
			"ROLE":             role.Role,
			"PREVIEW_HOSTNAME": previewHostname,
		}
		labels := map[string]string{
			"preview.plan_id":   detail.Plan.ID.String(),
			"preview.plan_name": detail.Plan.Name,
			"preview.role":      role.Role,
		}

		runningContainerName, err := s.containerSvc.RunContainer(ctx, RunContainerOptions{
			Name:          containerName,
			Image:         imageRef,
			Env:           env,
			NetworkName:   s.dockerPreviewNetwork,
			Labels:        labels,
			HostPort:      hostPort,
			ContainerPort: containerPort,
		})
		if err != nil {
			role.HostPort = nil
			role.ContainerName = nil
			role.DeployStatus = models.DeploymentPlanRoleDeployStatusFailed
			_ = s.repo.UpdateRole(ctx, role)
			s.failDeployment(ctx, detail, fmt.Sprintf("start container for role %q: %v", role.Role, err))
			return
		}

		role.ContainerName = stringPtr(runningContainerName)
		role.DeployStatus = models.DeploymentPlanRoleDeployStatusRunning
		if err := s.repo.UpdateRole(ctx, role); err != nil {
			s.failDeployment(ctx, detail, fmt.Sprintf("persist deployment role %q: %v", role.Role, err))
			return
		}
	}

	if err := s.nginx.WriteAndReload(ctx); err != nil {
		s.failDeployment(ctx, detail, fmt.Sprintf("write nginx config before health check: %v", err))
		return
	}

	if err := s.waitForRoleHealthChecks(ctx, detail); err != nil {
		s.failDeployment(ctx, detail, err.Error())
		return
	}

	if err := s.repo.UpdateStatus(ctx, planID, models.DeploymentPlanStatusRunning, &previewURL, nil); err != nil {
		s.failDeployment(ctx, detail, fmt.Sprintf("mark deployment plan running: %v", err))
		return
	}
	detail.Plan.Status = models.DeploymentPlanStatusRunning
	detail.Plan.PreviewURL = stringPtr(previewURL)
	detail.Plan.Error = nil

	if err := s.nginx.WriteAndReload(ctx); err != nil {
		s.failDeployment(ctx, detail, fmt.Sprintf("write nginx config after plan ready: %v", err))
		return
	}
}

func (s *deploymentPlanService) executeStop(ctx context.Context, detail *DeploymentPlanDetail) error {
	// Cancel any in-progress build goroutines first.
	for _, role := range detail.Roles {
		if role != nil && role.WorkerBuildID != nil {
			s.buildSvc.CancelBuild(*role.WorkerBuildID)
		}
	}

	for _, role := range detail.Roles {
		if role == nil {
			continue
		}

		containerName := previewContainerName(detail.Plan.Name, role.Role)
		if role.ContainerName != nil && strings.TrimSpace(*role.ContainerName) != "" {
			containerName = *role.ContainerName
		}

		if err := s.containerSvc.StopContainer(ctx, containerName); err != nil {
			return fmt.Errorf("stop container for role %q: %w", role.Role, err)
		}
		if err := s.containerSvc.RemoveContainer(ctx, containerName); err != nil {
			return fmt.Errorf("remove container for role %q: %w", role.Role, err)
		}

		role.ContainerName = nil
		role.HostPort = nil
		role.DeployStatus = models.DeploymentPlanRoleDeployStatusPending
		if err := s.repo.UpdateRole(ctx, role); err != nil {
			return fmt.Errorf("reset deployment role %q: %w", role.Role, err)
		}
	}

	if err := s.nginx.WriteAndReload(ctx); err != nil {
		// Nginx reload failure does not block stop — containers are already down.
		// Log the error and continue so the plan is still marked stopped.
		log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Msg("write nginx config after stop")
	}

	stoppedAt := time.Now().UTC()
	if err := s.repo.MarkStopped(ctx, detail.Plan.ID, stoppedAt); err != nil {
		return fmt.Errorf("mark deployment plan stopped: %w", err)
	}

	detail.Plan.Status = models.DeploymentPlanStatusStopped
	detail.Plan.StoppedAt = &stoppedAt
	detail.Plan.PreviewURL = nil
	detail.Plan.Error = nil
	return nil
}

func (s *deploymentPlanService) waitForRoleHealthChecks(ctx context.Context, detail *DeploymentPlanDetail) error {
	def, ok := s.registry.Get(detail.Plan.StackID)
	if !ok {
		return fmt.Errorf("stack registry entry %q not found", detail.Plan.StackID)
	}

	for _, roleDef := range def.Roles {
		role := findDeploymentRoleByName(detail.Roles, roleDef.Role)
		if role == nil || role.HostPort == nil {
			return fmt.Errorf("running host port missing for role %q", roleDef.Role)
		}

		path := strings.TrimSpace(roleDef.HealthcheckPath)
		if path == "" {
			continue
		}
		targetURL := fmt.Sprintf("http://host.docker.internal:%d%s", *role.HostPort, path)

		var lastErr error
		for attempt := 1; attempt <= s.healthCheckRetries; attempt++ {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
			if err != nil {
				return fmt.Errorf("build health check request for role %q: %w", role.Role, err)
			}

			resp, err := s.httpClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
					lastErr = nil
					break
				}
				lastErr = fmt.Errorf("unexpected status %d", resp.StatusCode)
			} else {
				lastErr = err
			}

			if attempt < s.healthCheckRetries {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(s.healthCheckInterval):
				}
			}
		}

		if lastErr != nil {
			return fmt.Errorf("health check failed for role %q at %s after %d attempts: %w", role.Role, targetURL, s.healthCheckRetries, lastErr)
		}
	}
	return nil
}

func (s *deploymentPlanService) failDeployment(ctx context.Context, detail *DeploymentPlanDetail, errMsg string) {
	log.Error().Str("deployment_plan_id", detail.Plan.ID.String()).Msg(errMsg)

	for _, role := range detail.Roles {
		if role == nil {
			continue
		}

		containerName := previewContainerName(detail.Plan.Name, role.Role)
		if role.ContainerName != nil && strings.TrimSpace(*role.ContainerName) != "" {
			containerName = *role.ContainerName
		}

		if err := s.containerSvc.StopContainer(ctx, containerName); err != nil {
			log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Str("role", role.Role).Msg("stop preview container failed")
		}
		if err := s.containerSvc.RemoveContainer(ctx, containerName); err != nil {
			log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Str("role", role.Role).Msg("remove preview container failed")
		}

		role.ContainerName = nil
		role.HostPort = nil
		if role.BuildStatus == models.DeploymentPlanRoleBuildStatusReady {
			role.DeployStatus = models.DeploymentPlanRoleDeployStatusFailed
		}
		if err := s.repo.UpdateRole(ctx, role); err != nil {
			log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Str("role", role.Role).Msg("persist failed deployment role failed")
		}
	}

	if err := s.nginx.WriteAndReload(ctx); err != nil {
		log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Msg("remove nginx route after deploy failure failed")
	}

	if err := s.repo.UpdateStatus(ctx, detail.Plan.ID, models.DeploymentPlanStatusFailed, nil, &errMsg); err != nil {
		log.Error().Err(err).Str("deployment_plan_id", detail.Plan.ID.String()).Msg("mark deployment plan failed failed")
	}
}

func (s *deploymentPlanService) loadOwnedPlanDetail(ctx context.Context, userID, id uuid.UUID) (*DeploymentPlanDetail, error) {
	detail, err := s.loadPlanDetail(ctx, id)
	if err != nil {
		return nil, err
	}
	if detail.Plan.UserID != userID {
		return nil, ErrDeploymentPlanNotFound
	}
	return detail, nil
}

func (s *deploymentPlanService) loadPlanDetail(ctx context.Context, id uuid.UUID) (*DeploymentPlanDetail, error) {
	plan, roles, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeploymentPlanNotFound
		}
		return nil, fmt.Errorf("load deployment plan: %w", err)
	}
	return &DeploymentPlanDetail{Plan: plan, Roles: roles}, nil
}

func normalizeRoleInputs(inputs []domains.DeploymentPlanRoleInput) (map[string]domains.DeploymentPlanRoleInput, error) {
	result := make(map[string]domains.DeploymentPlanRoleInput, len(inputs))
	for _, input := range inputs {
		role := strings.TrimSpace(input.Role)
		workerID := strings.TrimSpace(input.WorkerID)
		if role == "" || workerID == "" {
			return nil, validationError("role and worker_id are required for every role entry")
		}
		if _, exists := result[role]; exists {
			return nil, validationError(fmt.Sprintf("duplicate role %q", role))
		}
		result[role] = domains.DeploymentPlanRoleInput{
			Role:     role,
			WorkerID: workerID,
		}
	}
	return result, nil
}

func wrapBuildTriggerError(role string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("trigger build for role %q: %w", role, err)
}

func roleContainerPort(role string) (int, error) {
	switch role {
	case "frontend":
		return 5173, nil
	case "backend":
		return 8080, nil
	default:
		return 0, fmt.Errorf("unsupported runtime port mapping for role %q", role)
	}
}

func previewContainerName(planName, role string) string {
	return fmt.Sprintf("preview-%s-%s", planName, role)
}

func buildPreviewHostname(baseDomain, planName, tld string) string {
	return strings.TrimSpace(baseDomain) + "." + planName + "." + strings.TrimSpace(tld)
}

func buildPreviewURL(baseDomain, planName, tld string) string {
	return "http://" + buildPreviewHostname(baseDomain, planName, tld)
}

func deployImageReference(imageReference, digest string) string {
	ref := strings.TrimSpace(imageReference)
	if strings.HasPrefix(ref, "registry:5000/") {
		ref = "localhost:5001/" + strings.TrimPrefix(ref, "registry:5000/")
	}
	if strings.TrimSpace(digest) == "" || strings.Contains(ref, "@") {
		return ref
	}

	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		ref = ref[:lastColon]
	}
	return ref + "@" + strings.TrimSpace(digest)
}

func findDeploymentRole(roles []*models.DeploymentPlanRole, id uuid.UUID) *models.DeploymentPlanRole {
	for _, role := range roles {
		if role.ID == id {
			return role
		}
	}
	return nil
}

func findDeploymentRoleByName(roles []*models.DeploymentPlanRole, name string) *models.DeploymentPlanRole {
	for _, role := range roles {
		if role.Role == name {
			return role
		}
	}
	return nil
}

func allRolesReady(roles []*models.DeploymentPlanRole) bool {
	if len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if role.BuildStatus != models.DeploymentPlanRoleBuildStatusReady {
			return false
		}
	}
	return true
}

func buildFailureMessage(build *models.WorkerBuild, role string) string {
	if build.ErrorMessage != nil && strings.TrimSpace(*build.ErrorMessage) != "" {
		return *build.ErrorMessage
	}
	return fmt.Sprintf("build failed for role %q", role)
}

func previewBaseDomain() string {
	if value := strings.TrimSpace(getEnvOrDefault("PREVIEW_BASE_DOMAIN", "shiphide")); value != "" {
		return value
	}
	return "shiphide"
}

func previewTLD() string {
	if value := strings.TrimSpace(getEnvOrDefault("PREVIEW_TLD", "preview")); value != "" {
		return value
	}
	return "preview"
}

func dockerPreviewNetwork() string {
	if value := strings.TrimSpace(getEnvOrDefault("DOCKER_PREVIEW_NETWORK", "bridge")); value != "" {
		return value
	}
	return "bridge"
}

func healthCheckRetries() int {
	return envInt("HEALTH_CHECK_RETRIES", 20)
}

func healthCheckInterval() time.Duration {
	return time.Duration(envInt("HEALTH_CHECK_INTERVAL_SECONDS", 3)) * time.Second
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(getEnvOrDefault(key, ""))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getEnvOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validationError(message string) error {
	return fmt.Errorf("%w: %s", ErrDeploymentPlanValidation, message)
}

func invalidStateError(message string) error {
	return fmt.Errorf("%w: %s", ErrDeploymentPlanInvalidState, message)
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && string(pqErr.Code) == "23505"
}

func uuidPtr(value uuid.UUID) *uuid.UUID {
	v := value
	return &v
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func intPtr(value int) *int {
	v := value
	return &v
}
