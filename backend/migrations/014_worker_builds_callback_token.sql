-- +goose Up
-- +goose StatementBegin
ALTER TABLE worker_builds
    ADD COLUMN callback_token UUID NOT NULL DEFAULT gen_random_uuid();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE worker_builds DROP COLUMN IF EXISTS callback_token;
-- +goose StatementEnd
