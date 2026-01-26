ALTER TABLE gitlab_projects ADD COLUMN IF NOT EXISTS tenant_id TEXT;
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_tenant_id ON gitlab_projects(tenant_id);
