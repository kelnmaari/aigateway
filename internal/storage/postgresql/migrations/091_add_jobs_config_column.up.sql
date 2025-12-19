-- Add missing config column to gitlab_analysis_jobs table
-- Migration: 091_add_jobs_config_column

ALTER TABLE gitlab_analysis_jobs ADD COLUMN IF NOT EXISTS config JSONB;

-- Make action column nullable (webhooks may not have action)
ALTER TABLE gitlab_webhook_events ALTER COLUMN action DROP NOT NULL;

