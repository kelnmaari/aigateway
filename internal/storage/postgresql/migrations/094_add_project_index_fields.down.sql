-- Remove index-related fields from gitlab_projects
DROP INDEX IF EXISTS idx_gitlab_projects_index_status;
ALTER TABLE gitlab_projects DROP COLUMN IF EXISTS last_indexed_at;
ALTER TABLE gitlab_projects DROP COLUMN IF EXISTS index_status;
ALTER TABLE gitlab_projects DROP COLUMN IF EXISTS default_branch;

