package domains

import (
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
)

type CreatePreviewBundleRequest struct {
	StackID string `json:"stack_id"`
	TaskID  string `json:"task_id"`
}

type ReportPreviewBuildRequest struct {
	Role           string            `json:"role"`
	Status         string            `json:"status"`
	ImageReference string            `json:"image_reference,omitempty"`
	ImageDigest    string            `json:"image_digest,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ErrorMessage   string            `json:"error_message,omitempty"`
}

type PreviewStackResponse struct {
	StackID            string                     `json:"stack_id"`
	DisplayName        string                     `json:"display_name"`
	DeploymentTemplate string                     `json:"deployment_template"`
	Roles              []PreviewStackRoleResponse `json:"roles"`
}

type PreviewStackRoleResponse struct {
	Role                string            `json:"role"`
	WorkerGroup         string            `json:"worker_group"`
	ServiceName         string            `json:"service_name"`
	HealthcheckPath     string            `json:"healthcheck_path"`
	RequiredImageLabels map[string]string `json:"required_image_labels"`
}

type PreviewBundleResponse struct {
	ID          uuid.UUID                      `json:"id"`
	StackID     string                         `json:"stack_id"`
	TaskID      string                         `json:"task_id"`
	Status      models.PreviewBundleStatus     `json:"status"`
	PreviewURL  *string                        `json:"preview_url,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
	DestroyedAt *time.Time                     `json:"destroyed_at,omitempty"`
	Roles       []PreviewBundleRoleResponse    `json:"roles"`
	Manifests   []PreviewBuildManifestResponse `json:"manifests,omitempty"`
}

type PreviewBundleRoleResponse struct {
	ID               uuid.UUID                     `json:"id"`
	Role             string                        `json:"role"`
	WorkerGroup      string                        `json:"worker_group"`
	AssignedWorkerID *string                       `json:"assigned_worker_id,omitempty"`
	BuildStatus      models.PreviewBuildRoleStatus `json:"build_status"`
	LatestManifestID *string                       `json:"latest_manifest_id,omitempty"`
	ImageReference   *string                       `json:"image_reference,omitempty"`
	ImageDigest      *string                       `json:"image_digest,omitempty"`
	ErrorMessage     *string                       `json:"error_message,omitempty"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
}

type PreviewBuildManifestResponse struct {
	ID             uuid.UUID                     `json:"id"`
	Role           string                        `json:"role"`
	WorkerID       uuid.UUID                     `json:"worker_id"`
	Status         models.PreviewBuildRoleStatus `json:"status"`
	ImageReference string                        `json:"image_reference"`
	ImageDigest    *string                       `json:"image_digest,omitempty"`
	Metadata       map[string]string             `json:"metadata"`
	ErrorMessage   *string                       `json:"error_message,omitempty"`
	CreatedAt      time.Time                     `json:"created_at"`
}
