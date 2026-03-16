package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type workerRepo struct {
	db *sqlx.DB
}

func NewWorkerRepo(db *sqlx.DB) WorkerRepository {
	return &workerRepo{db: db}
}

func (r *workerRepo) GetByID(ctx context.Context, workerID uuid.UUID) (*models.Worker, error) {
	var w models.Worker
	err := r.db.GetContext(ctx, &w, `
		SELECT w.*, g.name AS group_name
		FROM workers w
		JOIN worker_groups g ON g.group_id = w.group_id
		WHERE w.worker_id=$1 AND w.deleted_at IS NULL`, workerID)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) GetLRUIdleByGroup(ctx context.Context, groupName string) (*models.Worker, error) {
	var w models.Worker
	err := r.db.GetContext(ctx, &w, `
		SELECT w.* FROM workers w
		JOIN worker_groups g ON g.group_id = w.group_id
		WHERE g.name = $1
		  AND w.status = 'idle'
		  AND w.deleted_at IS NULL
		ORDER BY w.last_active_at ASC NULLS FIRST
		LIMIT 1
		FOR UPDATE SKIP LOCKED`, groupName)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) UpdateStatus(ctx context.Context, workerID uuid.UUID, status models.WorkerStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET status=$1 WHERE worker_id=$2`, status, workerID)
	return err
}

func (r *workerRepo) UpdateLastActiveAt(ctx context.Context, workerID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET last_active_at=NOW() WHERE worker_id=$1`, workerID)
	return err
}

func (r *workerRepo) GetByAPIKeyHash(ctx context.Context, hash string) (*models.Worker, error) {
	var w models.Worker
	err := r.db.GetContext(ctx, &w,
		`SELECT * FROM workers WHERE api_key_hash=$1 AND deleted_at IS NULL`, hash)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *workerRepo) ListAll(ctx context.Context) ([]*models.Worker, error) {
	var workers []*models.Worker
	err := r.db.SelectContext(ctx, &workers, `
		SELECT w.*, g.name AS group_name
		FROM workers w
		JOIN worker_groups g ON g.group_id = w.group_id
		WHERE w.deleted_at IS NULL
		ORDER BY w.name`)
	return workers, err
}

func (r *workerRepo) ListGroups(ctx context.Context) ([]*models.WorkerGroup, error) {
	var groups []*models.WorkerGroup
	err := r.db.SelectContext(ctx, &groups,
		`SELECT group_id, name, created_at FROM worker_groups ORDER BY name`)
	return groups, err
}

func (r *workerRepo) GetOrCreateGroup(ctx context.Context, groupName string) (uuid.UUID, error) {
	var groupID uuid.UUID
	err := r.db.GetContext(ctx, &groupID,
		`SELECT group_id FROM worker_groups WHERE name=$1`, groupName)
	if err == nil {
		return groupID, nil
	}
	groupID = uuid.New()
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO worker_groups (group_id, name) VALUES ($1, $2)`, groupID, groupName)
	return groupID, err
}

func (r *workerRepo) CreateWorker(ctx context.Context, worker *models.Worker) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workers
		  (worker_id, group_id, name, callback_url, api_key_hash, workspace, cli_command, git_repo_url, instruction_job, instruction_plan, instruction_discuss, map_x, map_y, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		worker.WorkerID, worker.GroupID, worker.Name, worker.CallbackURL,
		worker.APIKeyHash, worker.Workspace, worker.CLICommand, worker.GitRepoURL,
		worker.InstructionJob, worker.InstructionPlan, worker.InstructionDiscuss,
		worker.MapX, worker.MapY, worker.Status)
	return err
}

func (r *workerRepo) UpdateCLICommand(ctx context.Context, workerID uuid.UUID, cliCommand string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET cli_command=$1 WHERE worker_id=$2`, cliCommand, workerID)
	return err
}


func (r *workerRepo) UpdateMapPosition(ctx context.Context, workerID uuid.UUID, mapX, mapY int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET map_x=$1, map_y=$2 WHERE worker_id=$3`, mapX, mapY, workerID)
	return err
}

func (r *workerRepo) UpdateInstructionFields(ctx context.Context, workerID uuid.UUID, patch WorkerInstructionPatch) error {
	setParts := make([]string, 0, 3)
	args := make([]any, 0, 4)
	idx := 1

	if patch.HasInstructionJob {
		setParts = append(setParts, "instruction_job = $"+itoa(idx))
		args = append(args, patch.InstructionJob)
		idx++
	}
	if patch.HasInstructionPlan {
		setParts = append(setParts, "instruction_plan = $"+itoa(idx))
		args = append(args, patch.InstructionPlan)
		idx++
	}
	if patch.HasInstructionDiscuss {
		setParts = append(setParts, "instruction_discuss = $"+itoa(idx))
		args = append(args, patch.InstructionDiscuss)
		idx++
	}
	if len(setParts) == 0 {
		return nil
	}

	args = append(args, workerID)
	query := `UPDATE workers SET ` + strings.Join(setParts, ", ") + ` WHERE worker_id = $` + itoa(idx)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *workerRepo) UpdatePreviewCommand(ctx context.Context, workerID uuid.UUID, cmd *string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET preview_command=$1 WHERE worker_id=$2`, cmd, workerID)
	return err
}

func (r *workerRepo) DeleteWorker(ctx context.Context, workerID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE workers SET deleted_at=NOW() WHERE worker_id=$1 AND deleted_at IS NULL`, workerID)
	return err
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
