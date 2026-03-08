-- +goose Up
-- +goose StatementBegin

ALTER TABLE workers
    ADD COLUMN IF NOT EXISTS map_x INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS map_y INTEGER NOT NULL DEFAULT 0;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE workers
    DROP COLUMN IF EXISTS map_x,
    DROP COLUMN IF EXISTS map_y;

-- +goose StatementEnd
