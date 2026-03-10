package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/stackregistry"
	"github.com/google/uuid"
)

type PreviewBundleService interface {
	ListStacks(ctx context.Context) ([]domains.PreviewStackResponse, error)
	Create(ctx context.Context, userID uuid.UUID, req domains.CreatePreviewBundleRequest) (*domains.PreviewBundleResponse, error)
	List(ctx context.Context, userID uuid.UUID) ([]*domains.PreviewBundleResponse, error)
	Get(ctx context.Context, userID uuid.UUID, bundleID uuid.UUID) (*domains.PreviewBundleResponse, error)
	Destroy(ctx context.Context, userID uuid.UUID, bundleID uuid.UUID) (*domains.PreviewBundleResponse, error)
	ReportBuild(ctx context.Context, workerID uuid.UUID, bundleID uuid.UUID, req domains.ReportPreviewBuildRequest) (*domains.PreviewBundleResponse, error)
}

type previewBundleService struct {
	repo       repository.PreviewBundleRepository
	workerRepo repository.WorkerRepository
	registry   *stackregistry.Registry
}

func NewPreviewBundleService(
	repo repository.PreviewBundleRepository,
	workerRepo repository.WorkerRepository,
	registry *stackregistry.Registry,
) PreviewBundleService {
	return &previewBundleService{repo: repo, workerRepo: workerRepo, registry: registry}
}

func (s *previewBundleService) ListStacks(ctx context.Context) ([]domains.PreviewStackResponse, error) {
	defs := s.registry.List()
	result := make([]domains.PreviewStackResponse, len(defs))
	for i, def := range defs {
		result[i] = stackResponse(def)
	}
	return result, nil
}

func (s *previewBundleService) Create(ctx context.Context, userID uuid.UUID, req domains.CreatePreviewBundleRequest) (*domains.PreviewBundleResponse, error) {
	if req.StackID == "" {
		return nil, fmt.Errorf("stack_id is required")
	}
	if req.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	def, ok := s.registry.Get(req.StackID)
	if !ok {
		return nil, fmt.Errorf("stack not found")
	}

	bundle := &models.PreviewBundle{
		ID:      uuid.New(),
		UserID:  userID,
		StackID: req.StackID,
		TaskID:  req.TaskID,
		Status:  models.PreviewBundleStatusPendingBuild,
	}

	roles := make([]*models.PreviewBundleRole, len(def.Roles))
	for i, role := range def.Roles {
		roles[i] = &models.PreviewBundleRole{
			ID:              uuid.New(),
			PreviewBundleID: bundle.ID,
			Role:            role.Role,
			WorkerGroup:     role.WorkerGroup,
			BuildStatus:     models.PreviewBuildRoleStatusRequested,
		}
	}

	if err := s.repo.Create(ctx, bundle, roles); err != nil {
		return nil, fmt.Errorf("create preview bundle: %w", err)
	}

	return s.Get(ctx, userID, bundle.ID)
}

func (s *previewBundleService) List(ctx context.Context, userID uuid.UUID) ([]*domains.PreviewBundleResponse, error) {
	bundles, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*domains.PreviewBundleResponse, 0, len(bundles))
	for _, bundle := range bundles {
		detail, err := s.Get(ctx, userID, bundle.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, detail)
	}

	return result, nil
}

func (s *previewBundleService) Get(ctx context.Context, userID uuid.UUID, bundleID uuid.UUID) (*domains.PreviewBundleResponse, error) {
	bundle, roles, manifests, err := s.repo.GetByID(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("preview bundle not found")
	}
	if bundle.UserID != userID {
		return nil, fmt.Errorf("preview bundle not found")
	}
	return toPreviewBundleResponse(bundle, roles, manifests)
}

func (s *previewBundleService) Destroy(ctx context.Context, userID uuid.UUID, bundleID uuid.UUID) (*domains.PreviewBundleResponse, error) {
	bundle, _, _, err := s.repo.GetByID(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("preview bundle not found")
	}
	if bundle.UserID != userID {
		return nil, fmt.Errorf("preview bundle not found")
	}
	if err := s.repo.MarkDestroyed(ctx, bundleID); err != nil {
		return nil, fmt.Errorf("destroy preview bundle: %w", err)
	}
	return s.Get(ctx, userID, bundleID)
}

func (s *previewBundleService) ReportBuild(ctx context.Context, workerID uuid.UUID, bundleID uuid.UUID, req domains.ReportPreviewBuildRequest) (*domains.PreviewBundleResponse, error) {
	if req.Role == "" {
		return nil, fmt.Errorf("role is required")
	}
	if !models.IsPreviewBuildRoleStatus(req.Status) || req.Status == string(models.PreviewBuildRoleStatusRequested) {
		return nil, fmt.Errorf("status must be ready or failed")
	}

	worker, err := s.workerRepo.GetByID(ctx, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker not found")
	}

	bundle, roles, _, err := s.repo.GetByID(ctx, bundleID)
	if err != nil {
		return nil, fmt.Errorf("preview bundle not found")
	}
	if bundle.Status == models.PreviewBundleStatusDestroyed {
		return nil, fmt.Errorf("preview bundle is destroyed")
	}

	roleRow := findPreviewRole(roles, req.Role)
	if roleRow == nil {
		return nil, fmt.Errorf("role not found in preview bundle")
	}
	if worker.GroupName != roleRow.WorkerGroup {
		return nil, fmt.Errorf("worker group %q cannot report role %q", worker.GroupName, req.Role)
	}

	roleStatus := models.PreviewBuildRoleStatus(req.Status)
	if roleStatus == models.PreviewBuildRoleStatusReady {
		if req.ImageReference == "" || req.ImageDigest == "" {
			return nil, fmt.Errorf("image_reference and image_digest are required for ready status")
		}
	} else if req.ErrorMessage == "" {
		return nil, fmt.Errorf("error_message is required for failed status")
	}

	metadataJSON, err := marshalMetadata(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("encode metadata: %w", err)
	}

	imageDigest := nullableString(req.ImageDigest)
	errorMessage := nullableString(req.ErrorMessage)

	manifest := &models.PreviewBuildManifest{
		ID:              uuid.New(),
		PreviewBundleID: bundleID,
		Role:            req.Role,
		WorkerID:        workerID,
		Status:          roleStatus,
		ImageReference:  req.ImageReference,
		ImageDigest:     imageDigest,
		MetadataJSON:    metadataJSON,
		ErrorMessage:    errorMessage,
	}
	if err := s.repo.RecordBuildManifest(ctx, manifest, roleStatus, workerID); err != nil {
		return nil, fmt.Errorf("record build manifest: %w", err)
	}

	bundleStatus, err := s.nextBundleStatus(ctx, bundleID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, bundleID, bundleStatus); err != nil {
		return nil, fmt.Errorf("update preview bundle status: %w", err)
	}

	return s.Get(ctx, bundle.UserID, bundleID)
}

func (s *previewBundleService) nextBundleStatus(ctx context.Context, bundleID uuid.UUID) (models.PreviewBundleStatus, error) {
	bundle, roles, _, err := s.repo.GetByID(ctx, bundleID)
	if err != nil {
		return "", fmt.Errorf("preview bundle not found")
	}
	if bundle.Status == models.PreviewBundleStatusDestroyed {
		return models.PreviewBundleStatusDestroyed, nil
	}

	anyReady := false
	allReady := len(roles) > 0
	for _, role := range roles {
		switch role.BuildStatus {
		case models.PreviewBuildRoleStatusFailed:
			return models.PreviewBundleStatusFailed, nil
		case models.PreviewBuildRoleStatusReady:
			anyReady = true
		case models.PreviewBuildRoleStatusRequested:
			allReady = false
		default:
			allReady = false
		}
	}

	if len(roles) == 0 {
		return models.PreviewBundleStatusPendingBuild, nil
	}
	if allReady {
		return models.PreviewBundleStatusReadyToDeploy, nil
	}
	if anyReady {
		return models.PreviewBundleStatusBuilding, nil
	}
	return models.PreviewBundleStatusPendingBuild, nil
}

func findPreviewRole(roles []*models.PreviewBundleRole, role string) *models.PreviewBundleRole {
	for _, item := range roles {
		if item.Role == role {
			return item
		}
	}
	return nil
}

func marshalMetadata(values map[string]string) (string, error) {
	if values == nil {
		values = map[string]string{}
	}
	bytes, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stackResponse(def stackregistry.Definition) domains.PreviewStackResponse {
	roles := make([]domains.PreviewStackRoleResponse, len(def.Roles))
	for i, role := range def.Roles {
		labels := make(map[string]string, len(role.RequiredImageLabels))
		for key, value := range role.RequiredImageLabels {
			labels[key] = value
		}
		roles[i] = domains.PreviewStackRoleResponse{
			Role:                role.Role,
			WorkerGroup:         role.WorkerGroup,
			ServiceName:         role.ServiceName,
			HealthcheckPath:     role.HealthcheckPath,
			RequiredImageLabels: labels,
		}
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Role < roles[j].Role
	})
	return domains.PreviewStackResponse{
		StackID:            def.StackID,
		DisplayName:        def.DisplayName,
		DeploymentTemplate: def.DeploymentTemplate,
		Roles:              roles,
	}
}

func toPreviewBundleResponse(
	bundle *models.PreviewBundle,
	roles []*models.PreviewBundleRole,
	manifests []*models.PreviewBuildManifest,
) (*domains.PreviewBundleResponse, error) {
	resp := &domains.PreviewBundleResponse{
		ID:          bundle.ID,
		StackID:     bundle.StackID,
		TaskID:      bundle.TaskID,
		Status:      bundle.Status,
		PreviewURL:  bundle.PreviewURL,
		CreatedAt:   bundle.CreatedAt,
		UpdatedAt:   bundle.UpdatedAt,
		DestroyedAt: bundle.DestroyedAt,
		Roles:       make([]domains.PreviewBundleRoleResponse, len(roles)),
	}

	for i, role := range roles {
		resp.Roles[i] = domains.PreviewBundleRoleResponse{
			ID:               role.ID,
			Role:             role.Role,
			WorkerGroup:      role.WorkerGroup,
			AssignedWorkerID: role.AssignedWorkerID,
			BuildStatus:      role.BuildStatus,
			LatestManifestID: role.LatestManifestID,
			ImageReference:   role.ImageReference,
			ImageDigest:      role.ImageDigest,
			ErrorMessage:     role.ErrorMessage,
			CreatedAt:        role.CreatedAt,
			UpdatedAt:        role.UpdatedAt,
		}
	}

	if len(manifests) > 0 {
		resp.Manifests = make([]domains.PreviewBuildManifestResponse, len(manifests))
		for i, manifest := range manifests {
			var metadata map[string]string
			if manifest.MetadataJSON != "" {
				if err := json.Unmarshal([]byte(manifest.MetadataJSON), &metadata); err != nil {
					return nil, fmt.Errorf("decode manifest metadata: %w", err)
				}
			}
			if metadata == nil {
				metadata = map[string]string{}
			}
			resp.Manifests[i] = domains.PreviewBuildManifestResponse{
				ID:             manifest.ID,
				Role:           manifest.Role,
				WorkerID:       manifest.WorkerID,
				Status:         manifest.Status,
				ImageReference: manifest.ImageReference,
				ImageDigest:    manifest.ImageDigest,
				Metadata:       metadata,
				ErrorMessage:   manifest.ErrorMessage,
				CreatedAt:      manifest.CreatedAt,
			}
		}
	}

	sort.Slice(resp.Roles, func(i, j int) bool {
		return resp.Roles[i].Role < resp.Roles[j].Role
	})

	return resp, nil
}
