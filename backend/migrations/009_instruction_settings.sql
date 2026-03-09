-- +goose Up
-- +goose StatementBegin

ALTER TABLE workers
    ADD COLUMN IF NOT EXISTS job_custom_instruction TEXT,
    ADD COLUMN IF NOT EXISTS plan_chat_custom_instruction TEXT,
    ADD COLUMN IF NOT EXISTS plan_generate_custom_instruction TEXT;

CREATE TABLE IF NOT EXISTS instruction_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    job_default_instruction TEXT NOT NULL,
    plan_chat_default_instruction TEXT NOT NULL,
    plan_generate_default_instruction TEXT NOT NULL
);

INSERT INTO instruction_settings (
    id,
    job_default_instruction,
    plan_chat_default_instruction,
    plan_generate_default_instruction
)
VALUES (
    1,
    'You are the assigned worker agent for this workspace.
Follow repository instructions such as AGENTS.md, the spec, and the architecture docs when relevant.
Keep scope tight and changes easy to review.
Do not guess missing requirements; ask when needed.
Point out weak assumptions or conflicts directly.
State important assumptions, tradeoffs, and risks clearly.',
    'You are helping the user scope work before execution.
Keep the conversation focused, concrete, and efficient.
Challenge vague requirements and surface missing constraints directly.',
    'You are converting scoped discussion into an execution-ready prompt.
Be explicit, concrete, and implementation-oriented.
Do not add unnecessary scope or extra deliverables.'
)
ON CONFLICT (id) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS instruction_settings;

ALTER TABLE workers
    DROP COLUMN IF EXISTS job_custom_instruction,
    DROP COLUMN IF EXISTS plan_chat_custom_instruction,
    DROP COLUMN IF EXISTS plan_generate_custom_instruction;

-- +goose StatementEnd
