-- +goose Up
-- +goose StatementBegin

CREATE TABLE plan_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(user_id),
    worker_id        UUID NOT NULL REFERENCES workers(worker_id),
    title            TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending','completed','discarded')),
    resume_id        TEXT,
    generated_prompt TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_plan_sessions_user_id    ON plan_sessions(user_id);
CREATE INDEX idx_plan_sessions_deleted_at ON plan_sessions(deleted_at);
CREATE INDEX idx_plan_sessions_updated_at ON plan_sessions(updated_at DESC);

CREATE TABLE plan_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_session_id UUID NOT NULL REFERENCES plan_sessions(id),
    role            TEXT NOT NULL CHECK (role IN ('user','agent')),
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plan_messages_session_id ON plan_messages(plan_session_id);
CREATE INDEX idx_plan_messages_created_at ON plan_messages(created_at);

CREATE TRIGGER plan_sessions_updated_at
BEFORE UPDATE ON plan_sessions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS plan_sessions_updated_at ON plan_sessions;
DROP TABLE IF EXISTS plan_messages;
DROP TABLE IF EXISTS plan_sessions;
-- +goose StatementEnd
