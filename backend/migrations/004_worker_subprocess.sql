-- +goose Up
-- +goose StatementBegin

ALTER TABLE workers
    ADD COLUMN IF NOT EXISTS workspace    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cli_command  TEXT NOT NULL DEFAULT 'claude';

-- Set workspace paths (mounted from host at /workspaces/ in the backend container)
-- and cli_command per worker.
UPDATE workers SET workspace = '/workspaces/fi-backend1',  cli_command = 'claude' WHERE name = 'fi-backend1';
UPDATE workers SET workspace = '/workspaces/fi-backend2',  cli_command = 'claude' WHERE name = 'fi-backend2';
UPDATE workers SET workspace = '/workspaces/fi-frontend1', cli_command = 'claude' WHERE name = 'fi-frontend1';
UPDATE workers SET workspace = '/workspaces/fi-frontend2', cli_command = 'claude' WHERE name = 'fi-frontend2';
UPDATE workers SET workspace = '/workspaces/ib-kha',       cli_command = 'claude' WHERE name = 'ib-kha';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE workers
    DROP COLUMN IF EXISTS workspace,
    DROP COLUMN IF EXISTS cli_command;
-- +goose StatementEnd
