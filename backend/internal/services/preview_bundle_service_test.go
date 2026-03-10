package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/domains"
	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/chatchawan/agent-orchestrator/internal/stackregistry"
	"github.com/google/uuid"
)

func TestPreviewBundleServiceCreateAndReportBuild(t *testing.T) {
	registry := mustLoadTestRegistry(t)
	repo := newFakePreviewBundleRepo()
	backendWorkerID := uuid.New()
	frontendWorkerID := uuid.New()
	workerRepo := fakeWorkerRepo{
		workers: map[uuid.UUID]*models.Worker{
			backendWorkerID: {
				WorkerID:  backendWorkerID,
				GroupName: "fi-backend",
			},
			frontendWorkerID: {
				WorkerID:  frontendWorkerID,
				GroupName: "fi-frontend",
			},
		},
	}

	svc := NewPreviewBundleService(repo, workerRepo, registry)
	userID := uuid.New()

	bundle, err := svc.Create(context.Background(), userID, domains.CreatePreviewBundleRequest{
		StackID: "fi-webapp",
		TaskID:  "task_123",
	})
	if err != nil {
		t.Fatalf("create preview bundle: %v", err)
	}
	if bundle.Status != models.PreviewBundleStatusPendingBuild {
		t.Fatalf("unexpected initial status: %s", bundle.Status)
	}
	if len(bundle.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(bundle.Roles))
	}

	bundle, err = svc.ReportBuild(context.Background(), backendWorkerID, bundle.ID, domains.ReportPreviewBuildRequest{
		Role:           "backend",
		Status:         "ready",
		ImageReference: "registry.local/fi-webapp/backend:preview",
		ImageDigest:    "sha256:backend",
		Metadata:       map[string]string{"workspace": "/workspaces/fi-backend1"},
	})
	if err != nil {
		t.Fatalf("report backend build: %v", err)
	}
	if bundle.Status != models.PreviewBundleStatusBuilding {
		t.Fatalf("unexpected bundle status after first role: %s", bundle.Status)
	}

	bundle, err = svc.ReportBuild(context.Background(), frontendWorkerID, bundle.ID, domains.ReportPreviewBuildRequest{
		Role:           "frontend",
		Status:         "ready",
		ImageReference: "registry.local/fi-webapp/frontend:preview",
		ImageDigest:    "sha256:frontend",
		Metadata:       map[string]string{"workspace": "/workspaces/fi-frontend1"},
	})
	if err != nil {
		t.Fatalf("report frontend build: %v", err)
	}
	if bundle.Status != models.PreviewBundleStatusReadyToDeploy {
		t.Fatalf("unexpected final bundle status: %s", bundle.Status)
	}
	if len(bundle.Manifests) != 2 {
		t.Fatalf("expected 2 manifests, got %d", len(bundle.Manifests))
	}
}

func TestPreviewBundleServiceRejectsWrongWorkerGroup(t *testing.T) {
	registry := mustLoadTestRegistry(t)
	repo := newFakePreviewBundleRepo()
	workerID := uuid.New()
	workerRepo := fakeWorkerRepo{
		workers: map[uuid.UUID]*models.Worker{
			workerID: {
				WorkerID:  workerID,
				GroupName: "fi-backend",
			},
		},
	}
	svc := NewPreviewBundleService(repo, workerRepo, registry)
	userID := uuid.New()

	bundle, err := svc.Create(context.Background(), userID, domains.CreatePreviewBundleRequest{
		StackID: "fi-webapp",
		TaskID:  "task_456",
	})
	if err != nil {
		t.Fatalf("create preview bundle: %v", err)
	}

	if _, err := svc.ReportBuild(context.Background(), workerID, bundle.ID, domains.ReportPreviewBuildRequest{
		Role:           "frontend",
		Status:         "ready",
		ImageReference: "registry.local/fi-webapp/frontend:preview",
		ImageDigest:    "sha256:frontend",
	}); err == nil {
		t.Fatalf("expected worker-group mismatch error")
	}
}

func mustLoadTestRegistry(t *testing.T) *stackregistry.Registry {
	t.Helper()

	dir := t.TempDir()
	content := `{
	  "stack_id": "fi-webapp",
	  "display_name": "FI Webapp",
	  "deployment_template": "subdomain-preview",
	  "roles": [
	    {
	      "role": "backend",
	      "worker_group": "fi-backend",
	      "service_name": "backend",
	      "healthcheck_path": "/healthz",
	      "required_image_labels": {
	        "preview.role": "backend"
	      }
	    },
	    {
	      "role": "frontend",
	      "worker_group": "fi-frontend",
	      "service_name": "frontend",
	      "healthcheck_path": "/",
	      "required_image_labels": {
	        "preview.role": "frontend"
	      }
	    }
	  ]
	}`
	if err := os.WriteFile(filepath.Join(dir, "fi-webapp.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write stack registry file: %v", err)
	}

	registry, err := stackregistry.LoadFromDir(dir)
	if err != nil {
		t.Fatalf("load test registry: %v", err)
	}
	return registry
}

type fakePreviewBundleRepo struct {
	bundles   map[uuid.UUID]*models.PreviewBundle
	roles     map[uuid.UUID][]*models.PreviewBundleRole
	manifests map[uuid.UUID][]*models.PreviewBuildManifest
}

func newFakePreviewBundleRepo() *fakePreviewBundleRepo {
	return &fakePreviewBundleRepo{
		bundles:   map[uuid.UUID]*models.PreviewBundle{},
		roles:     map[uuid.UUID][]*models.PreviewBundleRole{},
		manifests: map[uuid.UUID][]*models.PreviewBuildManifest{},
	}
}

func (r *fakePreviewBundleRepo) Create(_ context.Context, bundle *models.PreviewBundle, roles []*models.PreviewBundleRole) error {
	now := time.Now().UTC()
	bundle.CreatedAt = now
	bundle.UpdatedAt = now
	r.bundles[bundle.ID] = cloneBundle(bundle)
	clonedRoles := make([]*models.PreviewBundleRole, len(roles))
	for i, role := range roles {
		role.CreatedAt = now
		role.UpdatedAt = now
		clonedRoles[i] = cloneRole(role)
	}
	r.roles[bundle.ID] = clonedRoles
	return nil
}

func (r *fakePreviewBundleRepo) GetByID(_ context.Context, id uuid.UUID) (*models.PreviewBundle, []*models.PreviewBundleRole, []*models.PreviewBuildManifest, error) {
	bundle, ok := r.bundles[id]
	if !ok {
		return nil, nil, nil, os.ErrNotExist
	}
	roleCopies := make([]*models.PreviewBundleRole, len(r.roles[id]))
	for i, role := range r.roles[id] {
		roleCopies[i] = cloneRole(role)
	}
	manifestCopies := make([]*models.PreviewBuildManifest, len(r.manifests[id]))
	for i, manifest := range r.manifests[id] {
		manifestCopies[i] = cloneManifest(manifest)
	}
	return cloneBundle(bundle), roleCopies, manifestCopies, nil
}

func (r *fakePreviewBundleRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]*models.PreviewBundle, error) {
	var result []*models.PreviewBundle
	for _, bundle := range r.bundles {
		if bundle.UserID == userID {
			result = append(result, cloneBundle(bundle))
		}
	}
	return result, nil
}

func (r *fakePreviewBundleRepo) RecordBuildManifest(_ context.Context, manifest *models.PreviewBuildManifest, roleStatus models.PreviewBuildRoleStatus, workerID uuid.UUID) error {
	now := time.Now().UTC()
	manifest.CreatedAt = now
	r.manifests[manifest.PreviewBundleID] = append([]*models.PreviewBuildManifest{cloneManifest(manifest)}, r.manifests[manifest.PreviewBundleID]...)

	for _, role := range r.roles[manifest.PreviewBundleID] {
		if role.Role != manifest.Role {
			continue
		}
		workerIDText := workerID.String()
		manifestID := manifest.ID.String()
		role.AssignedWorkerID = &workerIDText
		role.BuildStatus = roleStatus
		role.LatestManifestID = &manifestID
		role.UpdatedAt = now
		if manifest.ImageReference != "" {
			ref := manifest.ImageReference
			role.ImageReference = &ref
		} else {
			role.ImageReference = nil
		}
		if manifest.ImageDigest != nil {
			digest := *manifest.ImageDigest
			role.ImageDigest = &digest
		} else {
			role.ImageDigest = nil
		}
		if manifest.ErrorMessage != nil {
			msg := *manifest.ErrorMessage
			role.ErrorMessage = &msg
		} else {
			role.ErrorMessage = nil
		}
	}
	return nil
}

func (r *fakePreviewBundleRepo) UpdateStatus(_ context.Context, id uuid.UUID, status models.PreviewBundleStatus) error {
	bundle, ok := r.bundles[id]
	if !ok {
		return os.ErrNotExist
	}
	bundle.Status = status
	bundle.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *fakePreviewBundleRepo) MarkDestroyed(_ context.Context, id uuid.UUID) error {
	bundle, ok := r.bundles[id]
	if !ok {
		return os.ErrNotExist
	}
	now := time.Now().UTC()
	bundle.Status = models.PreviewBundleStatusDestroyed
	bundle.DestroyedAt = &now
	bundle.UpdatedAt = now
	return nil
}

type fakeWorkerRepo struct {
	workers map[uuid.UUID]*models.Worker
}

func (r fakeWorkerRepo) GetByID(_ context.Context, workerID uuid.UUID) (*models.Worker, error) {
	worker, ok := r.workers[workerID]
	if !ok {
		return nil, os.ErrNotExist
	}
	copy := *worker
	return &copy, nil
}

func (r fakeWorkerRepo) GetLRUIdleByGroup(context.Context, string) (*models.Worker, error) {
	panic("unexpected call")
}

func (r fakeWorkerRepo) UpdateStatus(context.Context, uuid.UUID, models.WorkerStatus) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) UpdateLastActiveAt(context.Context, uuid.UUID) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) GetByAPIKeyHash(context.Context, string) (*models.Worker, error) {
	panic("unexpected call")
}

func (r fakeWorkerRepo) ListAll(context.Context) ([]*models.Worker, error) {
	panic("unexpected call")
}

func (r fakeWorkerRepo) GetOrCreateGroup(context.Context, string) (uuid.UUID, error) {
	panic("unexpected call")
}

func (r fakeWorkerRepo) ListGroups(context.Context) ([]*models.WorkerGroup, error) {
	panic("unexpected call")
}

func (r fakeWorkerRepo) CreateWorker(context.Context, *models.Worker) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) UpdateCLICommand(context.Context, uuid.UUID, string) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) UpdateMapPosition(context.Context, uuid.UUID, int, int) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) UpdateInstructionFields(context.Context, uuid.UUID, repository.WorkerInstructionPatch) error {
	panic("unexpected call")
}

func (r fakeWorkerRepo) DeleteWorker(context.Context, uuid.UUID) error {
	panic("unexpected call")
}

func cloneBundle(bundle *models.PreviewBundle) *models.PreviewBundle {
	copy := *bundle
	if bundle.PreviewURL != nil {
		value := *bundle.PreviewURL
		copy.PreviewURL = &value
	}
	if bundle.DestroyedAt != nil {
		value := *bundle.DestroyedAt
		copy.DestroyedAt = &value
	}
	return &copy
}

func cloneRole(role *models.PreviewBundleRole) *models.PreviewBundleRole {
	copy := *role
	if role.AssignedWorkerID != nil {
		value := *role.AssignedWorkerID
		copy.AssignedWorkerID = &value
	}
	if role.LatestManifestID != nil {
		value := *role.LatestManifestID
		copy.LatestManifestID = &value
	}
	if role.ImageReference != nil {
		value := *role.ImageReference
		copy.ImageReference = &value
	}
	if role.ImageDigest != nil {
		value := *role.ImageDigest
		copy.ImageDigest = &value
	}
	if role.ErrorMessage != nil {
		value := *role.ErrorMessage
		copy.ErrorMessage = &value
	}
	return &copy
}

func cloneManifest(manifest *models.PreviewBuildManifest) *models.PreviewBuildManifest {
	copy := *manifest
	if manifest.ImageDigest != nil {
		value := *manifest.ImageDigest
		copy.ImageDigest = &value
	}
	if manifest.ErrorMessage != nil {
		value := *manifest.ErrorMessage
		copy.ErrorMessage = &value
	}
	return &copy
}

var _ repository.PreviewBundleRepository = (*fakePreviewBundleRepo)(nil)
var _ repository.WorkerRepository = fakeWorkerRepo{}
