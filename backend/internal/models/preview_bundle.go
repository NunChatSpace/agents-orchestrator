package models

import (
	"time"

	"github.com/google/uuid"
)

type PreviewBundleStatus string

const (
	PreviewBundleStatusPendingBuild  PreviewBundleStatus = "pending_build"
	PreviewBundleStatusBuilding      PreviewBundleStatus = "building"
	PreviewBundleStatusReadyToDeploy PreviewBundleStatus = "ready_to_deploy"
	PreviewBundleStatusDeploying     PreviewBundleStatus = "deploying"
	PreviewBundleStatusHealthy       PreviewBundleStatus = "healthy"
	PreviewBundleStatusFailed        PreviewBundleStatus = "failed"
	PreviewBundleStatusDestroyed     PreviewBundleStatus = "destroyed"
)

func IsPreviewBundleStatus(value string) bool {
	switch PreviewBundleStatus(value) {
	case PreviewBundleStatusPendingBuild,
		PreviewBundleStatusBuilding,
		PreviewBundleStatusReadyToDeploy,
		PreviewBundleStatusDeploying,
		PreviewBundleStatusHealthy,
		PreviewBundleStatusFailed,
		PreviewBundleStatusDestroyed:
		return true
	default:
		return false
	}
}

type PreviewBuildRoleStatus string

const (
	PreviewBuildRoleStatusRequested PreviewBuildRoleStatus = "requested"
	PreviewBuildRoleStatusReady     PreviewBuildRoleStatus = "ready"
	PreviewBuildRoleStatusFailed    PreviewBuildRoleStatus = "failed"
)

func IsPreviewBuildRoleStatus(value string) bool {
	switch PreviewBuildRoleStatus(value) {
	case PreviewBuildRoleStatusRequested,
		PreviewBuildRoleStatusReady,
		PreviewBuildRoleStatusFailed:
		return true
	default:
		return false
	}
}

type PreviewBundle struct {
	ID          uuid.UUID           `db:"id"`
	UserID      uuid.UUID           `db:"user_id"`
	StackID     string              `db:"stack_id"`
	TaskID      string              `db:"task_id"`
	Status      PreviewBundleStatus `db:"status"`
	PreviewURL  *string             `db:"preview_url"`
	CreatedAt   time.Time           `db:"created_at"`
	UpdatedAt   time.Time           `db:"updated_at"`
	DestroyedAt *time.Time          `db:"destroyed_at"`
}

type PreviewBundleRole struct {
	ID               uuid.UUID              `db:"id"`
	PreviewBundleID  uuid.UUID              `db:"preview_bundle_id"`
	Role             string                 `db:"role"`
	WorkerGroup      string                 `db:"worker_group"`
	AssignedWorkerID *string                `db:"assigned_worker_id"`
	BuildStatus      PreviewBuildRoleStatus `db:"build_status"`
	LatestManifestID *string                `db:"latest_manifest_id"`
	ImageReference   *string                `db:"image_reference"`
	ImageDigest      *string                `db:"image_digest"`
	ErrorMessage     *string                `db:"error_message"`
	CreatedAt        time.Time              `db:"created_at"`
	UpdatedAt        time.Time              `db:"updated_at"`
}

type PreviewBuildManifest struct {
	ID              uuid.UUID              `db:"id"`
	PreviewBundleID uuid.UUID              `db:"preview_bundle_id"`
	Role            string                 `db:"role"`
	WorkerID        uuid.UUID              `db:"worker_id"`
	Status          PreviewBuildRoleStatus `db:"status"`
	ImageReference  string                 `db:"image_reference"`
	ImageDigest     *string                `db:"image_digest"`
	MetadataJSON    string                 `db:"metadata_json"`
	ErrorMessage    *string                `db:"error_message"`
	CreatedAt       time.Time              `db:"created_at"`
}
