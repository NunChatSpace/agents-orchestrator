package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DeploymentPlanRepository interface {
	Create(ctx context.Context, plan *models.DeploymentPlan, roles []*models.DeploymentPlanRole) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.DeploymentPlan, []*models.DeploymentPlanRole, error)
	GetByName(ctx context.Context, name string) (*models.DeploymentPlan, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.DeploymentPlan, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.DeploymentPlanStatus, previewURL, errMsg *string) error
	UpdateRole(ctx context.Context, role *models.DeploymentPlanRole) error
	Delete(ctx context.Context, id uuid.UUID) error
	MarkStopped(ctx context.Context, id uuid.UUID, stoppedAt time.Time) error
	ListRunning(ctx context.Context) ([]*models.DeploymentPlan, error)
	ListRolesByWorkerBuildID(ctx context.Context, workerBuildID uuid.UUID) ([]*models.DeploymentPlanRole, error)
	AllocateNextHostPort(ctx context.Context, roleID uuid.UUID, startPort int) (int, error)
}

type deploymentPlanRepo struct {
	db *sqlx.DB
}

func NewDeploymentPlanRepo(db *sqlx.DB) DeploymentPlanRepository {
	return &deploymentPlanRepo{db: db}
}

func (r *deploymentPlanRepo) Create(ctx context.Context, plan *models.DeploymentPlan, roles []*models.DeploymentPlanRole) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackDeploymentPlan(tx)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO deployment_plans
		  (id, user_id, name, stack_id, build_mode, status, preview_url, error, stopped_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		plan.ID, plan.UserID, plan.Name, plan.StackID, plan.BuildMode, plan.Status,
		plan.PreviewURL, plan.Error, plan.StoppedAt, plan.CreatedAt, plan.UpdatedAt,
	); err != nil {
		return err
	}

	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO deployment_plan_roles
			  (id, plan_id, role, worker_id, worker_build_id, image_reference, image_digest, container_name, host_port, build_status, deploy_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
			role.ID, role.PlanID, role.Role, role.WorkerID, role.WorkerBuildID, role.ImageReference,
			role.ImageDigest, role.ContainerName, role.HostPort, role.BuildStatus, role.DeployStatus,
			role.CreatedAt, role.UpdatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *deploymentPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.DeploymentPlan, []*models.DeploymentPlanRole, error) {
	var plan models.DeploymentPlan
	if err := r.db.GetContext(ctx, &plan, `
		SELECT id, user_id, name, stack_id, build_mode, status, preview_url, error, created_at, updated_at, stopped_at
		FROM deployment_plans
		WHERE id = $1`, id); err != nil {
		return nil, nil, err
	}

	var roles []*models.DeploymentPlanRole
	if err := r.db.SelectContext(ctx, &roles, `
		SELECT dpr.id,
		       dpr.plan_id,
		       dpr.role,
		       dpr.worker_id,
		       dpr.worker_build_id,
		       dpr.image_reference,
		       dpr.image_digest,
		       dpr.container_name,
		       dpr.host_port,
		       dpr.build_status,
		       dpr.deploy_status,
		       dpr.created_at,
		       dpr.updated_at,
		       wb.status AS worker_build_status
		FROM deployment_plan_roles dpr
		LEFT JOIN worker_builds wb ON wb.id = dpr.worker_build_id
		WHERE dpr.plan_id = $1
		ORDER BY dpr.role ASC`, id); err != nil {
		return nil, nil, err
	}

	return &plan, roles, nil
}

func (r *deploymentPlanRepo) GetByName(ctx context.Context, name string) (*models.DeploymentPlan, error) {
	var plan models.DeploymentPlan
	err := r.db.GetContext(ctx, &plan, `
		SELECT id, user_id, name, stack_id, build_mode, status, preview_url, error, created_at, updated_at, stopped_at
		FROM deployment_plans
		WHERE name = $1`, name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (r *deploymentPlanRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.DeploymentPlan, error) {
	var plans []*models.DeploymentPlan
	err := r.db.SelectContext(ctx, &plans, `
		SELECT id, user_id, name, stack_id, build_mode, status, preview_url, error, created_at, updated_at, stopped_at
		FROM deployment_plans
		WHERE user_id = $1
		ORDER BY updated_at DESC, created_at DESC`, userID)
	return plans, err
}

func (r *deploymentPlanRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.DeploymentPlanStatus, previewURL, errMsg *string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE deployment_plans
		SET status = $2,
		    preview_url = $3,
		    error = $4
		WHERE id = $1`, id, status, previewURL, errMsg)
	return err
}

func (r *deploymentPlanRepo) UpdateRole(ctx context.Context, role *models.DeploymentPlanRole) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE deployment_plan_roles
		SET worker_build_id = $2,
		    image_reference = $3,
		    image_digest = $4,
		    container_name = $5,
		    host_port = $6,
		    build_status = $7,
		    deploy_status = $8
		WHERE id = $1`,
		role.ID, role.WorkerBuildID, role.ImageReference, role.ImageDigest,
		role.ContainerName, role.HostPort, role.BuildStatus, role.DeployStatus,
	)
	return err
}

func (r *deploymentPlanRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM deployment_plans WHERE id = $1`, id)
	return err
}

func (r *deploymentPlanRepo) MarkStopped(ctx context.Context, id uuid.UUID, stoppedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE deployment_plans
		SET status = 'stopped',
		    preview_url = NULL,
		    error = NULL,
		    stopped_at = $2
		WHERE id = $1`, id, stoppedAt)
	return err
}

func (r *deploymentPlanRepo) ListRunning(ctx context.Context) ([]*models.DeploymentPlan, error) {
	var plans []*models.DeploymentPlan
	err := r.db.SelectContext(ctx, &plans, `
		SELECT id, user_id, name, stack_id, build_mode, status, preview_url, error, created_at, updated_at, stopped_at
		FROM deployment_plans
		WHERE status = 'running'
		ORDER BY name ASC`)
	return plans, err
}

func (r *deploymentPlanRepo) ListRolesByWorkerBuildID(ctx context.Context, workerBuildID uuid.UUID) ([]*models.DeploymentPlanRole, error) {
	var roles []*models.DeploymentPlanRole
	err := r.db.SelectContext(ctx, &roles, `
		SELECT id,
		       plan_id,
		       role,
		       worker_id,
		       worker_build_id,
		       image_reference,
		       image_digest,
		       container_name,
		       host_port,
		       build_status,
		       deploy_status,
		       created_at,
		       updated_at
		FROM deployment_plan_roles
		WHERE worker_build_id = $1
		ORDER BY created_at ASC`, workerBuildID)
	return roles, err
}

func (r *deploymentPlanRepo) AllocateNextHostPort(ctx context.Context, roleID uuid.UUID, startPort int) (int, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollbackDeploymentPlan(tx)

	if _, err := tx.ExecContext(ctx, `LOCK TABLE deployment_plan_roles IN EXCLUSIVE MODE`); err != nil {
		return 0, err
	}

	var usedPorts []int
	if err := tx.SelectContext(ctx, &usedPorts, `
		SELECT host_port
		FROM deployment_plan_roles
		WHERE host_port IS NOT NULL
		ORDER BY host_port ASC`); err != nil {
		return 0, err
	}

	used := make(map[int]struct{}, len(usedPorts))
	for _, port := range usedPorts {
		used[port] = struct{}{}
	}

	port := startPort
	for {
		if _, exists := used[port]; !exists {
			break
		}
		port++
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE deployment_plan_roles
		SET host_port = $2
		WHERE id = $1`, roleID, port)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rowsAffected == 0 {
		return 0, sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return port, nil
}

func rollbackDeploymentPlan(tx *sqlx.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

var _ DeploymentPlanRepository = (*deploymentPlanRepo)(nil)
