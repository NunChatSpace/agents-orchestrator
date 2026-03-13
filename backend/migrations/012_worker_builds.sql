-- +goose Up
-- +goose StatementBegin

CREATE TABLE worker_builds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id       UUID NOT NULL REFERENCES workers(worker_id),
    stack_id        TEXT NOT NULL,
    role            TEXT NOT NULL,
    build_mode      TEXT NOT NULL DEFAULT 'fresh'
                        CHECK (build_mode IN ('fresh', 'latest')),
    status          TEXT NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'building', 'ready', 'failed')),
    callback_token  UUID NOT NULL DEFAULT gen_random_uuid(),
    image_reference TEXT,
    image_digest    TEXT,
    error_message   TEXT,
    triggered_by    UUID REFERENCES users(user_id),
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_worker_builds_worker_id ON worker_builds(worker_id);
CREATE INDEX idx_worker_builds_status ON worker_builds(status);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS worker_builds;
-- +goose StatementEnd
