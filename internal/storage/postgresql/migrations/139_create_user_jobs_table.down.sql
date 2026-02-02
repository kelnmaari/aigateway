-- Drop user_jobs table
DROP INDEX IF EXISTS idx_user_jobs_project_created;
DROP INDEX IF EXISTS idx_user_jobs_user_status;
DROP INDEX IF EXISTS idx_user_jobs_created_at;
DROP INDEX IF EXISTS idx_user_jobs_job_type;
DROP INDEX IF EXISTS idx_user_jobs_status;
DROP INDEX IF EXISTS idx_user_jobs_integration_id;
DROP INDEX IF EXISTS idx_user_jobs_project_id;
DROP INDEX IF EXISTS idx_user_jobs_user_id;
DROP TABLE IF EXISTS user_jobs;
