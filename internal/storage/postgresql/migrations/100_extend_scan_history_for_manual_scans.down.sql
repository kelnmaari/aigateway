-- Rollback: Remove new columns and restore NOT NULL constraint
-- Warning: This will delete manual scan records (schedule_id IS NULL)

DROP INDEX IF EXISTS idx_scan_history_scan_type;
DROP INDEX IF EXISTS idx_scan_history_integration;

ALTER TABLE gitlab_scan_history DROP COLUMN IF EXISTS integration_id;
ALTER TABLE gitlab_scan_history DROP COLUMN IF EXISTS model_id;
ALTER TABLE gitlab_scan_history DROP COLUMN IF EXISTS findings_count;
ALTER TABLE gitlab_scan_history DROP COLUMN IF EXISTS files_affected;
ALTER TABLE gitlab_scan_history DROP COLUMN IF EXISTS tokens_used;

-- Delete manual scans and restore NOT NULL constraint
DELETE FROM gitlab_scan_history WHERE schedule_id IS NULL;
ALTER TABLE gitlab_scan_history ALTER COLUMN schedule_id SET NOT NULL;

