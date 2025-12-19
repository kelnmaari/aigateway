-- Rollback: Remove payload and status columns from gitlab_webhook_events
ALTER TABLE gitlab_webhook_events DROP COLUMN IF EXISTS payload;
ALTER TABLE gitlab_webhook_events DROP COLUMN IF EXISTS status;

