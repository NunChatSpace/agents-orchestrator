-- +goose Up
-- +goose StatementBegin

CREATE TABLE users (
    user_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username     TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ
);

CREATE TABLE sessions (
    session_id TEXT PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE TABLE worker_groups (
    group_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workers (
    worker_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id       UUID NOT NULL REFERENCES worker_groups(group_id),
    name           TEXT NOT NULL UNIQUE,
    callback_url   TEXT NOT NULL,
    api_key_hash   TEXT NOT NULL,
    status         TEXT NOT NULL DEFAULT 'offline'
                       CHECK (status IN ('busy','pending_user','idle','offline')),
    last_active_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);
CREATE INDEX idx_workers_group_id       ON workers(group_id);
CREATE INDEX idx_workers_status         ON workers(status);
CREATE INDEX idx_workers_last_active_at ON workers(last_active_at);

CREATE TABLE jobs (
    job_id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID NOT NULL REFERENCES users(user_id),
    title                  TEXT NOT NULL,
    prompt                 TEXT NOT NULL,
    target_group           TEXT NOT NULL,
    assigned_worker_id     UUID REFERENCES workers(worker_id),
    manual_worker_override UUID REFERENCES workers(worker_id),
    status                 TEXT NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft','queued','assigned','busy','pending_user','done','failed','cancelled')),
    resume_id              TEXT,
    failure_reason         TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at             TIMESTAMPTZ
);
CREATE INDEX idx_jobs_user_id         ON jobs(user_id);
CREATE INDEX idx_jobs_status          ON jobs(status);
CREATE INDEX idx_jobs_target_group    ON jobs(target_group);
CREATE INDEX idx_jobs_updated_at      ON jobs(updated_at DESC);
CREATE INDEX idx_jobs_assigned_worker ON jobs(assigned_worker_id);

CREATE TABLE messages (
    message_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     UUID NOT NULL REFERENCES jobs(job_id),
    role       TEXT NOT NULL CHECK (role IN ('user','oagent','worker')),
    kind       TEXT NOT NULL CHECK (kind IN ('instruction','question','answer','summary','system')),
    content    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_messages_job_id     ON messages(job_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Auto-update updated_at on jobs
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER jobs_updated_at
BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS jobs_updated_at ON jobs;
DROP FUNCTION IF EXISTS set_updated_at();
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS workers;
DROP TABLE IF EXISTS worker_groups;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
