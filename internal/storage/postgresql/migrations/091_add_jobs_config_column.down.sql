-- Rollback: Remove config column
ALTER TABLE gitlab_analysis_jobs DROP COLUMN IF EXISTS config;
ALTER TABLE gitlab_webhook_events ALTER COLUMN action SET NOT NULL;

