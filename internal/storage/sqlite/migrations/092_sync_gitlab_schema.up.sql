-- Synchronize GitLab tables schema with models
-- Migration: 092_sync_gitlab_schema

-- gitlab_integrations
ALTER TABLE gitlab_integrations ADD COLUMN owner_id TEXT;
ALTER TABLE gitlab_integrations ADD COLUMN tenant_id TEXT;

-- gitlab_analysis_jobs  
ALTER TABLE gitlab_analysis_jobs ADD COLUMN updated_at TEXT DEFAULT CURRENT_TIMESTAMP;

-- gitlab_mr_reviews
ALTER TABLE gitlab_mr_reviews ADD COLUMN max_retries INTEGER DEFAULT 3;
ALTER TABLE gitlab_mr_reviews ADD COLUMN updated_at TEXT DEFAULT CURRENT_TIMESTAMP;

