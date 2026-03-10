package repository

import (
	"context"
	"database/sql"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type previewBundleRepo struct {
	db *sqlx.DB
}

func NewPreviewBundleRepo(db *sqlx.DB) PreviewBundleRepository {
	return &previewBundleRepo{db: db}
}

func (r *previewBundleRepo) Create(ctx context.Context, bundle *models.PreviewBundle, roles []*models.PreviewBundleRole) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO preview_bundles (id, user_id, stack_id, task_id, status, preview_url, destroyed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		bundle.ID, bundle.UserID, bundle.StackID, bundle.TaskID, bundle.Status, bundle.PreviewURL, bundle.DestroyedAt,
	); err != nil {
		return err
	}

	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO preview_bundle_roles
			  (id, preview_bundle_id, role, worker_group, assigned_worker_id, build_status, latest_manifest_id, image_reference, image_digest, error_message)
			VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, $6, NULLIF($7, '')::uuid, $8, $9, $10)`,
			role.ID, role.PreviewBundleID, role.Role, role.WorkerGroup, strPtrValue(role.AssignedWorkerID),
			role.BuildStatus, strPtrValue(role.LatestManifestID), role.ImageReference, role.ImageDigest, role.ErrorMessage,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *previewBundleRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.PreviewBundle, []*models.PreviewBundleRole, []*models.PreviewBuildManifest, error) {
	var bundle models.PreviewBundle
	if err := r.db.GetContext(ctx, &bundle, `
		SELECT id, user_id, stack_id, task_id, status, preview_url, created_at, updated_at, destroyed_at
		FROM preview_bundles
		WHERE id=$1`, id); err != nil {
		return nil, nil, nil, err
	}

	var roles []*models.PreviewBundleRole
	if err := r.db.SelectContext(ctx, &roles, `
		SELECT id,
		       preview_bundle_id,
		       role,
		       worker_group,
		       assigned_worker_id::text AS assigned_worker_id,
		       build_status,
		       latest_manifest_id::text AS latest_manifest_id,
		       image_reference,
		       image_digest,
		       error_message,
		       created_at,
		       updated_at
		FROM preview_bundle_roles
		WHERE preview_bundle_id=$1
		ORDER BY role ASC`, id); err != nil {
		return nil, nil, nil, err
	}

	var manifests []*models.PreviewBuildManifest
	if err := r.db.SelectContext(ctx, &manifests, `
		SELECT id,
		       preview_bundle_id,
		       role,
		       worker_id,
		       status,
		       image_reference,
		       image_digest,
		       metadata_json::text AS metadata_json,
		       error_message,
		       created_at
		FROM preview_build_manifests
		WHERE preview_bundle_id=$1
		ORDER BY created_at DESC`, id); err != nil {
		return nil, nil, nil, err
	}

	return &bundle, roles, manifests, nil
}

func (r *previewBundleRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.PreviewBundle, error) {
	var bundles []*models.PreviewBundle
	if err := r.db.SelectContext(ctx, &bundles, `
		SELECT id, user_id, stack_id, task_id, status, preview_url, created_at, updated_at, destroyed_at
		FROM preview_bundles
		WHERE user_id=$1
		ORDER BY updated_at DESC`, userID); err != nil {
		return nil, err
	}
	return bundles, nil
}

func (r *previewBundleRepo) RecordBuildManifest(ctx context.Context, manifest *models.PreviewBuildManifest, roleStatus models.PreviewBuildRoleStatus, workerID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO preview_build_manifests
		  (id, preview_bundle_id, role, worker_id, status, image_reference, image_digest, metadata_json, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)`,
		manifest.ID, manifest.PreviewBundleID, manifest.Role, manifest.WorkerID, manifest.Status,
		manifest.ImageReference, manifest.ImageDigest, manifest.MetadataJSON, manifest.ErrorMessage,
	); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE preview_bundle_roles
		SET assigned_worker_id = $1,
		    build_status = $2,
		    latest_manifest_id = $3,
		    image_reference = $4,
		    image_digest = $5,
		    error_message = $6
		WHERE preview_bundle_id = $7 AND role = $8`,
		workerID, roleStatus, manifest.ID, manifest.ImageReference, manifest.ImageDigest, manifest.ErrorMessage, manifest.PreviewBundleID, manifest.Role,
	)
	if err != nil {
		return err
	}
	if rows, err := result.RowsAffected(); err != nil {
		return err
	} else if rows == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

func (r *previewBundleRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.PreviewBundleStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE preview_bundles SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (r *previewBundleRepo) MarkDestroyed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE preview_bundles
		SET status = 'destroyed',
		    destroyed_at = NOW()
		WHERE id = $1`, id)
	return err
}

func rollback(tx *sqlx.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func strPtrValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

var _ PreviewBundleRepository = (*previewBundleRepo)(nil)
