-- Add index-related fields to gitlab_projects
ALTER TABLE gitlab_projects ADD COLUMN default_branch TEXT DEFAULT 'main';
ALTER TABLE gitlab_projects ADD COLUMN index_status TEXT DEFAULT 'pending';
ALTER TABLE gitlab_projects ADD COLUMN last_indexed_at DATETIME;

-- Add index for faster queries
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_index_status ON gitlab_projects(index_status);

