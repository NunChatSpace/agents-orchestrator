-- +goose Up
ALTER TABLE workers ADD COLUMN build_command TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE workers DROP COLUMN build_command;
