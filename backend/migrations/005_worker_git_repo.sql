-- +goose Up
-- +goose StatementBegin

ALTER TABLE workers
    ADD COLUMN IF NOT EXISTS git_repo_url TEXT NOT NULL DEFAULT '';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE workers
    DROP COLUMN IF EXISTS git_repo_url;

-- +goose StatementEnd
