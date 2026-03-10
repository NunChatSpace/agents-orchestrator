-- +goose Up
-- +goose StatementBegin

CREATE TABLE preview_bundles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(user_id),
    stack_id     TEXT NOT NULL,
    task_id      TEXT NOT NULL,
    status       TEXT NOT NULL
                     CHECK (status IN ('pending_build','building','ready_to_deploy','deploying','healthy','failed','destroyed')),
    preview_url  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    destroyed_at TIMESTAMPTZ
);

CREATE INDEX idx_preview_bundles_user_id ON preview_bundles(user_id);
CREATE INDEX idx_preview_bundles_status ON preview_bundles(status);
CREATE INDEX idx_preview_bundles_updated_at ON preview_bundles(updated_at DESC);

CREATE TABLE preview_bundle_roles (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preview_bundle_id  UUID NOT NULL REFERENCES preview_bundles(id) ON DELETE CASCADE,
    role               TEXT NOT NULL,
    worker_group       TEXT NOT NULL,
    assigned_worker_id UUID REFERENCES workers(worker_id),
    build_status       TEXT NOT NULL CHECK (build_status IN ('requested','ready','failed')),
    latest_manifest_id UUID,
    image_reference    TEXT,
    image_digest       TEXT,
    error_message      TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (preview_bundle_id, role)
);

CREATE INDEX idx_preview_bundle_roles_bundle_id ON preview_bundle_roles(preview_bundle_id);
CREATE INDEX idx_preview_bundle_roles_build_status ON preview_bundle_roles(build_status);

CREATE TABLE preview_build_manifests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preview_bundle_id UUID NOT NULL REFERENCES preview_bundles(id) ON DELETE CASCADE,
    role              TEXT NOT NULL,
    worker_id         UUID NOT NULL REFERENCES workers(worker_id),
    status            TEXT NOT NULL CHECK (status IN ('ready','failed')),
    image_reference   TEXT NOT NULL DEFAULT '',
    image_digest      TEXT,
    metadata_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message     TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_preview_build_manifests_bundle_id ON preview_build_manifests(preview_bundle_id);
CREATE INDEX idx_preview_build_manifests_role ON preview_build_manifests(role);
CREATE INDEX idx_preview_build_manifests_created_at ON preview_build_manifests(created_at DESC);

CREATE TRIGGER preview_bundles_updated_at
BEFORE UPDATE ON preview_bundles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER preview_bundle_roles_updated_at
BEFORE UPDATE ON preview_bundle_roles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS preview_bundle_roles_updated_at ON preview_bundle_roles;
DROP TRIGGER IF EXISTS preview_bundles_updated_at ON preview_bundles;
DROP TABLE IF EXISTS preview_build_manifests;
DROP TABLE IF EXISTS preview_bundle_roles;
DROP TABLE IF EXISTS preview_bundles;
-- +goose StatementEnd
