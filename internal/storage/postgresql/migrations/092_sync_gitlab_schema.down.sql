-- Rollback: Remove added columns and triggers
DROP TRIGGER IF EXISTS gitlab_jobs_updated_at ON gitlab_analysis_jobs;
DROP TRIGGER IF EXISTS gitlab_reviews_updated_at ON gitlab_mr_reviews;

ALTER TABLE gitlab_integrations DROP COLUMN IF EXISTS owner_id;
ALTER TABLE gitlab_integrations DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE gitlab_analysis_jobs DROP COLUMN IF EXISTS updated_at;
ALTER TABLE gitlab_mr_reviews DROP COLUMN IF EXISTS max_retries;
ALTER TABLE gitlab_mr_reviews DROP COLUMN IF EXISTS updated_at;

