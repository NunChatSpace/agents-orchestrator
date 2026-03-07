-- +goose Up
-- +goose StatementBegin
ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_kind_check;
ALTER TABLE messages
    ADD CONSTRAINT messages_kind_check
    CHECK (kind IN ('instruction','question','answer','summary','system','thinking','file_change'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_kind_check;
ALTER TABLE messages
    ADD CONSTRAINT messages_kind_check
    CHECK (kind IN ('instruction','question','answer','summary','system','thinking'));
-- +goose StatementEnd
