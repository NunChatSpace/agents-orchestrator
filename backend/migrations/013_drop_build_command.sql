-- Drop build_command column — replaced by process-based preview (preview_command)
ALTER TABLE workers DROP COLUMN IF EXISTS build_command;
