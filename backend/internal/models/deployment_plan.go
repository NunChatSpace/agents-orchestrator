package models

import (
	"time"

	"github.com/google/uuid"
)

type DeploymentPlanStatus string

const (
	DeploymentPlanStatusPending   DeploymentPlanStatus = "pending"
	DeploymentPlanStatusDeploying DeploymentPlanStatus = "deploying"
	DeploymentPlanStatusRunning   DeploymentPlanStatus = "running"
	DeploymentPlanStatusFailed    DeploymentPlanStatus = "failed"
	DeploymentPlanStatusStopped   DeploymentPlanStatus = "stopped"
)

func IsDeploymentPlanStatus(value string) bool {
	switch DeploymentPlanStatus(value) {
	case DeploymentPlanStatusPending,
		DeploymentPlanStatusDeploying,
		DeploymentPlanStatusRunning,
		DeploymentPlanStatusFailed,
		DeploymentPlanStatusStopped:
		return true
	default:
		return false
	}
}

type DeploymentPlanRoleBuildStatus string

const (
	DeploymentPlanRoleBuildStatusPending DeploymentPlanRoleBuildStatus = "pending"
	DeploymentPlanRoleBuildStatusReady   DeploymentPlanRoleBuildStatus = "ready"
	DeploymentPlanRoleBuildStatusFailed  DeploymentPlanRoleBuildStatus = "failed"
)

func IsDeploymentPlanRoleBuildStatus(value string) bool {
	switch DeploymentPlanRoleBuildStatus(value) {
	case DeploymentPlanRoleBuildStatusPending,
		DeploymentPlanRoleBuildStatusReady,
		DeploymentPlanRoleBuildStatusFailed:
		return true
	default:
		return false
	}
}

type DeploymentPlanRoleDeployStatus string

const (
	DeploymentPlanRoleDeployStatusPending DeploymentPlanRoleDeployStatus = "pending"
	DeploymentPlanRoleDeployStatusRunning DeploymentPlanRoleDeployStatus = "running"
	DeploymentPlanRoleDeployStatusFailed  DeploymentPlanRoleDeployStatus = "failed"
)

func IsDeploymentPlanRoleDeployStatus(value string) bool {
	switch DeploymentPlanRoleDeployStatus(value) {
	case DeploymentPlanRoleDeployStatusPending,
		DeploymentPlanRoleDeployStatusRunning,
		DeploymentPlanRoleDeployStatusFailed:
		return true
	default:
		return false
	}
}

type DeploymentPlan struct {
	ID         uuid.UUID            `db:"id"`
	UserID     uuid.UUID            `db:"user_id"`
	Name       string               `db:"name"`
	StackID    string               `db:"stack_id"`
	BuildMode  WorkerBuildMode      `db:"build_mode"`
	Status     DeploymentPlanStatus `db:"status"`
	PreviewURL *string              `db:"preview_url"`
	Error      *string              `db:"error"`
	CreatedAt  time.Time            `db:"created_at"`
	UpdatedAt  time.Time            `db:"updated_at"`
	StoppedAt  *time.Time           `db:"stopped_at"`
}

type DeploymentPlanRole struct {
	ID                uuid.UUID                      `db:"id"`
	PlanID            uuid.UUID                      `db:"plan_id"`
	Role              string                         `db:"role"`
	WorkerID          uuid.UUID                      `db:"worker_id"`
	WorkerBuildID     *uuid.UUID                     `db:"worker_build_id"`
	ImageReference    *string                        `db:"image_reference"`
	ImageDigest       *string                        `db:"image_digest"`
	ContainerName     *string                        `db:"container_name"`
	HostPort          *int                           `db:"host_port"`
	BuildStatus       DeploymentPlanRoleBuildStatus  `db:"build_status"`
	DeployStatus      DeploymentPlanRoleDeployStatus `db:"deploy_status"`
	CreatedAt         time.Time                      `db:"created_at"`
	UpdatedAt         time.Time                      `db:"updated_at"`
	// WorkerBuildStatus is populated via JOIN when reading — never written.
	WorkerBuildStatus *string `db:"worker_build_status"`
}
