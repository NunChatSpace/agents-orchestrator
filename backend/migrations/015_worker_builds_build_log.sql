-- +goose Up
ALTER TABLE worker_builds ADD COLUMN build_log TEXT;

-- +goose Down
ALTER TABLE worker_builds DROP COLUMN build_log;
