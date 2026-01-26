-- Remove tenant_id index and column from gitlab_projects
DROP INDEX IF EXISTS idx_gitlab_projects_tenant_id;
ALTER TABLE gitlab_projects DROP COLUMN IF EXISTS tenant_id;
