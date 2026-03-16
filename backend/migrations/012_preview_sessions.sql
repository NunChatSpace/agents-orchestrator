-- Add preview_command to workers (shell command the agent runs to start the preview server)
ALTER TABLE workers ADD COLUMN IF NOT EXISTS preview_command TEXT;

-- One preview session groups one or more agents previewing together
CREATE TABLE preview_sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by   UUID NOT NULL REFERENCES users(user_id),
    status       TEXT NOT NULL DEFAULT 'starting',  -- starting | running | stopped | failed
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at   TIMESTAMPTZ
);

CREATE INDEX idx_preview_sessions_created_by ON preview_sessions(created_by);
CREATE INDEX idx_preview_sessions_status     ON preview_sessions(status);

-- One row per agent role within a session
CREATE TABLE preview_session_roles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id    UUID NOT NULL REFERENCES preview_sessions(id) ON DELETE CASCADE,
    worker_id     UUID NOT NULL REFERENCES workers(worker_id),
    role          TEXT NOT NULL,
    port          INT,
    status        TEXT NOT NULL DEFAULT 'starting',  -- starting | running | stopped | failed
    error_message TEXT,
    preview_url   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_preview_session_roles_session_id ON preview_session_roles(session_id);
CREATE INDEX idx_preview_session_roles_worker_id  ON preview_session_roles(worker_id);
