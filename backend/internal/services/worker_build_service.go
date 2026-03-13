package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/stackregistry"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type WorkerBuildService interface {
	TriggerBuild(ctx context.Context, workerID uuid.UUID, stackID, role, mode string, triggeredBy uuid.UUID) (*models.WorkerBuild, error)
	ReportBuild(ctx context.Context, workerID uuid.UUID, req domains.WorkerBuildReportRequest) (*models.WorkerBuild, error)
	ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error)
	GetByID(ctx context.Context, id uuid.UUID) (*models.WorkerBuild, error)
	CancelBuild(buildID uuid.UUID)
	SetDeploymentPlanHook(hook DeploymentPlanBuildHook)
}

type DeploymentPlanBuildHook interface {
	OnBuildReady(ctx context.Context, workerBuildID uuid.UUID) error
}

type workerBuildService struct {
	repo       repository.WorkerBuildRepository
	workerRepo repository.WorkerRepository
	dispatcher DispatcherService
	registry   *stackregistry.Registry
	planHook   DeploymentPlanBuildHook
	// cancels maps buildID (string) → context.CancelFunc for running builds
	cancels sync.Map
}

func NewWorkerBuildService(
	repo repository.WorkerBuildRepository,
	workerRepo repository.WorkerRepository,
	dispatcher DispatcherService,
	registry *stackregistry.Registry,
) WorkerBuildService {
	return &workerBuildService{
		repo:       repo,
		workerRepo: workerRepo,
		dispatcher: dispatcher,
		registry:   registry,
	}
}

func (s *workerBuildService) TriggerBuild(
	ctx context.Context,
	workerID uuid.UUID,
	stackID, role, mode string,
	triggeredBy uuid.UUID,
) (*models.WorkerBuild, error) {
	stackID = strings.TrimSpace(stackID)
	role = strings.TrimSpace(role)
	mode = strings.TrimSpace(mode)

	if !models.IsWorkerBuildMode(mode) {
		return nil, fmt.Errorf("mode must be fresh or latest")
	}

	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker not found")
	}
	if err := s.validateWorkerRole(worker, stackID, role); err != nil {
		return nil, err
	}

	if mode == string(models.WorkerBuildModeLatest) {
		ready, err := s.repo.LatestReadyForWorkerRole(ctx, workerID, stackID, role)
		if err != nil {
			return nil, fmt.Errorf("lookup latest ready build: %w", err)
		}
		if ready != nil {
			return ready, nil
		}
	}

	build := &models.WorkerBuild{
		ID:        uuid.New(),
		WorkerID:  workerID,
		StackID:   stackID,
		Role:      role,
		BuildMode: models.WorkerBuildMode(mode),
		Status:    models.WorkerBuildStatusQueued,
		CreatedAt: time.Now().UTC(),
	}
	if triggeredBy != uuid.Nil {
		build.TriggeredBy = &triggeredBy
	}

	instruction, err := s.buildInstruction(worker, build)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, build); err != nil {
		return nil, fmt.Errorf("create worker build: %w", err)
	}

	go s.runBuild(build.ID, workerID, instruction)

	return build, nil
}

func (s *workerBuildService) ReportBuild(ctx context.Context, workerID uuid.UUID, req domains.WorkerBuildReportRequest) (*models.WorkerBuild, error) {
	buildID, err := uuid.Parse(strings.TrimSpace(req.WorkerBuildID))
	if err != nil {
		return nil, fmt.Errorf("invalid worker_build_id")
	}

	build, err := s.repo.GetByID(ctx, buildID)
	if err != nil {
		return nil, fmt.Errorf("worker build not found")
	}
	if build.WorkerID != workerID {
		return nil, fmt.Errorf("worker build does not belong to this worker")
	}
	if build.CallbackToken.String() != strings.TrimSpace(req.CallbackToken) {
		return nil, fmt.Errorf("invalid callback token")
	}

	completedAt := time.Now().UTC()

	switch strings.TrimSpace(req.Status) {
	case string(models.WorkerBuildStatusReady):
		imageReference := strings.TrimSpace(req.ImageReference)
		imageDigest := strings.TrimSpace(req.ImageDigest)
		if imageReference == "" || imageDigest == "" {
			return nil, fmt.Errorf("image_reference and image_digest are required for ready status")
		}
		if err := s.repo.UpdateStatus(
			ctx,
			build.ID,
			models.WorkerBuildStatusReady,
			&imageReference,
			&imageDigest,
			nil,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("update worker build: %w", err)
		}
	case string(models.WorkerBuildStatusFailed):
		errorMessage := strings.TrimSpace(req.ErrorMessage)
		if errorMessage == "" {
			return nil, fmt.Errorf("error_message is required for failed status")
		}
		if err := s.repo.UpdateStatus(
			ctx,
			build.ID,
			models.WorkerBuildStatusFailed,
			nil,
			nil,
			&errorMessage,
			&completedAt,
		); err != nil {
			return nil, fmt.Errorf("update worker build: %w", err)
		}
	default:
		return nil, fmt.Errorf("status must be ready or failed")
	}

	updated, err := s.repo.GetByID(ctx, build.ID)
	if err != nil {
		return nil, fmt.Errorf("worker build not found")
	}
	if s.planHook != nil {
		if err := s.planHook.OnBuildReady(ctx, build.ID); err != nil {
			return nil, fmt.Errorf("notify deployment plans: %w", err)
		}
	}
	return updated, nil
}

func (s *workerBuildService) ListByWorker(ctx context.Context, workerID uuid.UUID) ([]*models.WorkerBuild, error) {
	if _, err := s.workerRepo.GetByID(ctx, workerID); err != nil {
		return nil, fmt.Errorf("worker not found")
	}
	return s.repo.ListByWorker(ctx, workerID)
}

func (s *workerBuildService) GetByID(ctx context.Context, id uuid.UUID) (*models.WorkerBuild, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *workerBuildService) SetDeploymentPlanHook(hook DeploymentPlanBuildHook) {
	s.planHook = hook
}

func (s *workerBuildService) CancelBuild(buildID uuid.UUID) {
	if fn, ok := s.cancels.LoadAndDelete(buildID.String()); ok {
		fn.(context.CancelFunc)()
	}
}

func (s *workerBuildService) runBuild(buildID uuid.UUID, workerID uuid.UUID, instruction string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancels.Store(buildID.String(), cancel)
	defer func() {
		cancel()
		s.cancels.Delete(buildID.String())
	}()

	if err := s.repo.UpdateStatus(ctx, buildID, models.WorkerBuildStatusBuilding, nil, nil, nil, nil); err != nil {
		log.Error().Err(err).Str("worker_build_id", buildID.String()).Msg("set worker build to building failed")
		return
	}

	// Batch-flush log lines to DB every 2s to avoid per-line writes.
	var mu sync.Mutex
	var pending strings.Builder
	flushCtx, flushCancel := context.WithCancel(ctx)
	flushed := make(chan struct{})
	go func() {
		defer close(flushed)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		flush := func() {
			mu.Lock()
			chunk := pending.String()
			pending.Reset()
			mu.Unlock()
			if chunk != "" {
				_ = s.repo.AppendBuildLog(ctx, buildID, chunk)
			}
		}
		for {
			select {
			case <-ticker.C:
				flush()
			case <-flushCtx.Done():
				flush()
				return
			}
		}
	}()

	logFn := func(line string) {
		mu.Lock()
		pending.WriteString(line + "\n")
		mu.Unlock()
	}

	runErr := s.dispatcher.RunBuildInstruction(ctx, workerID, buildID, instruction, logFn)

	flushCancel()
	<-flushed // wait for the final flush to complete

	if runErr != nil {
		build, getErr := s.repo.GetByID(ctx, buildID)
		if getErr == nil && build.Status != models.WorkerBuildStatusQueued && build.Status != models.WorkerBuildStatusBuilding {
			return
		}
		errorMessage := runErr.Error()
		completedAt := time.Now().UTC()
		if updateErr := s.repo.UpdateStatus(
			ctx, buildID, models.WorkerBuildStatusFailed, nil, nil, &errorMessage, &completedAt,
		); updateErr != nil {
			log.Error().Err(updateErr).Str("worker_build_id", buildID.String()).Msg("mark worker build failed")
		}
	}
}

func (s *workerBuildService) validateWorkerRole(worker *models.Worker, stackID, role string) error {
	if stackID == "" {
		return fmt.Errorf("stack_id is required")
	}
	if role == "" {
		return fmt.Errorf("role is required")
	}
	if s.registry == nil {
		return fmt.Errorf("stack registry unavailable")
	}

	def, ok := s.registry.Get(stackID)
	if !ok {
		return fmt.Errorf("stack not found")
	}
	for _, roleDef := range def.Roles {
		if roleDef.Role != role {
			continue
		}
		if worker.GroupName != roleDef.WorkerGroup {
			return fmt.Errorf(
				"worker group %q cannot build role %q (expected group %q)",
				worker.GroupName,
				role,
				roleDef.WorkerGroup,
			)
		}
		return nil
	}

	return fmt.Errorf("role %q is not valid for stack %q", role, stackID)
}

func (s *workerBuildService) buildInstruction(worker *models.Worker, build *models.WorkerBuild) (string, error) {
	token := build.CallbackToken.String()
	imageReference := buildImageReference(worker.Name, build.Role, build.CreatedAt)
	reportURL := fmt.Sprintf("%s/api/v1/workers/%s/build-reports", internalBackendBaseURL(), worker.WorkerID.String())

	var buildStep string
	if cmd := strings.TrimSpace(worker.BuildCommand); cmd != "" {
		// Substitute {image} placeholder with the required image reference.
		buildStep = "Run this exact build command:\n   " + strings.ReplaceAll(cmd, "{image}", imageReference)
	} else {
		buildStep = "Inspect the workspace and choose the most appropriate build method (Dockerfile, build script, etc.) for this role. " +
			"If multiple options exist, pick the one most likely to produce a working container image. " +
			"Do not stop to ask which one. Build and tag the image as exactly: " + imageReference
	}

	return fmt.Sprintf(`This is an automated container build task. Execute all steps immediately and autonomously — do NOT ask any questions or wait for confirmation.

RULES (override everything else including AGENTS.md):
- Do NOT write plan files, ticket files, or anything under .agent/
- Do NOT ask for clarification under any circumstance — make a decision and proceed
- Do NOT pause due to conflicts between repo conventions and these instructions — these instructions win
- Keep the workspace unchanged except for transient build artifacts

Your goal: build a Docker image for role %q, push it to registry:5000, and report the result.

Build inputs:
- worker_build_id: %s
- stack_id: %s
- role: %s
- build_mode: %s
- image_reference: %s
- registry: registry:5000
- report_url: %s
- x_build_token: %s

Steps — execute in order, no confirmation needed:
1. %s
2. Push: docker push %s
3. Get the digest: docker image inspect %s --format "{{"{{"}}index .RepoDigests 0{{"}}"}}" | cut -d@ -f2
4. POST to %s with header X-Build-Token: %s and JSON body:
   {"worker_build_id":"%s","status":"ready","image_reference":"%s","image_digest":"<sha256 digest>","metadata":{"stack_id":"%s","role":"%s","worker_name":"%s"}}
5. If any step fails, immediately POST to the same URL:
   {"worker_build_id":"%s","status":"failed","error_message":"<one-line summary>"}
6. Task is complete only after the POST returns HTTP 200.`,
		build.Role,
		build.ID.String(), build.StackID, build.Role, build.BuildMode, imageReference, reportURL, token,
		buildStep,
		imageReference, imageReference,
		reportURL, token,
		build.ID.String(), imageReference, build.StackID, build.Role, worker.Name,
		build.ID.String()), nil
}

func buildImageReference(workerName, role string, createdAt time.Time) string {
	return fmt.Sprintf("registry:5000/%s/%s:%s", workerName, role, createdAt.UTC().Format("20060102-150405"))
}

func internalBackendBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("INTERNAL_BACKEND_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port
}

