-- +goose Up
-- +goose StatementBegin

CREATE TABLE deployment_plans (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(user_id),
    name        TEXT NOT NULL UNIQUE,
    stack_id    TEXT NOT NULL,
    build_mode  TEXT NOT NULL
                    CHECK (build_mode IN ('fresh', 'latest')),
    status      TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'deploying', 'running', 'failed', 'stopped')),
    preview_url TEXT,
    error       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at  TIMESTAMPTZ
);

CREATE INDEX idx_deployment_plans_user_id ON deployment_plans(user_id);
CREATE INDEX idx_deployment_plans_status ON deployment_plans(status);
CREATE INDEX idx_deployment_plans_updated_at ON deployment_plans(updated_at DESC);

CREATE TABLE deployment_plan_roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL REFERENCES deployment_plans(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    worker_id       UUID NOT NULL REFERENCES workers(worker_id),
    worker_build_id UUID REFERENCES worker_builds(id),
    image_reference TEXT,
    image_digest    TEXT,
    container_name  TEXT,
    host_port       INT,
    build_status    TEXT NOT NULL DEFAULT 'pending'
                       CHECK (build_status IN ('pending', 'ready', 'failed')),
    deploy_status   TEXT NOT NULL DEFAULT 'pending'
                       CHECK (deploy_status IN ('pending', 'running', 'failed')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (plan_id, role)
);

CREATE INDEX idx_deployment_plan_roles_plan_id ON deployment_plan_roles(plan_id);
CREATE INDEX idx_deployment_plan_roles_worker_build_id ON deployment_plan_roles(worker_build_id);
CREATE INDEX idx_deployment_plan_roles_build_status ON deployment_plan_roles(build_status);

CREATE TRIGGER deployment_plans_updated_at
BEFORE UPDATE ON deployment_plans
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER deployment_plan_roles_updated_at
BEFORE UPDATE ON deployment_plan_roles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS deployment_plan_roles_updated_at ON deployment_plan_roles;
DROP TRIGGER IF EXISTS deployment_plans_updated_at ON deployment_plans;
DROP TABLE IF EXISTS deployment_plan_roles;
DROP TABLE IF EXISTS deployment_plans;
-- +goose StatementEnd
