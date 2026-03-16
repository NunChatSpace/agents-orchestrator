-- Add log_output column to preview_session_roles for streaming terminal output
ALTER TABLE preview_session_roles ADD COLUMN IF NOT EXISTS log_output TEXT NOT NULL DEFAULT '';
