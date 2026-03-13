package domains

import "time"

type CreateDeploymentPlanRequest struct {
	Name      string                    `json:"name"`
	StackID   string                    `json:"stack_id"`
	BuildMode string                    `json:"build_mode"`
	Roles     []DeploymentPlanRoleInput `json:"roles"`
}

type DeploymentPlanRoleInput struct {
	Role     string `json:"role"`
	WorkerID string `json:"worker_id"`
}

type DeploymentPlanRoleResponse struct {
	ID                string  `json:"id"`
	Role              string  `json:"role"`
	WorkerID          string  `json:"worker_id"`
	WorkerBuildID     *string `json:"worker_build_id,omitempty"`
	ImageReference    *string `json:"image_reference,omitempty"`
	ImageDigest       *string `json:"image_digest,omitempty"`
	ContainerName     *string `json:"container_name,omitempty"`
	BuildStatus       string  `json:"build_status"`
	DeployStatus      string  `json:"deploy_status"`
	WorkerBuildStatus *string `json:"worker_build_status,omitempty"`
}

type DeploymentPlanResponse struct {
	ID         string                       `json:"id"`
	Name       string                       `json:"name"`
	StackID    string                       `json:"stack_id"`
	BuildMode  string                       `json:"build_mode"`
	Status     string                       `json:"status"`
	PreviewURL *string                      `json:"preview_url,omitempty"`
	Error      *string                      `json:"error,omitempty"`
	Roles      []DeploymentPlanRoleResponse `json:"roles"`
	CreatedAt  time.Time                    `json:"created_at"`
	UpdatedAt  time.Time                    `json:"updated_at"`
	StoppedAt  *time.Time                   `json:"stopped_at,omitempty"`
}
