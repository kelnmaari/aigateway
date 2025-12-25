-- Add index-related fields to gitlab_projects
ALTER TABLE gitlab_projects ADD COLUMN IF NOT EXISTS default_branch VARCHAR(255) DEFAULT 'main';
ALTER TABLE gitlab_projects ADD COLUMN IF NOT EXISTS index_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE gitlab_projects ADD COLUMN IF NOT EXISTS last_indexed_at TIMESTAMPTZ;

-- Add index for faster queries
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_index_status ON gitlab_projects(index_status);

COMMENT ON COLUMN gitlab_projects.default_branch IS 'Default branch for indexing (target branch)';
COMMENT ON COLUMN gitlab_projects.index_status IS 'Current indexing status: pending, in_progress, completed, failed';
COMMENT ON COLUMN gitlab_projects.last_indexed_at IS 'Timestamp of last successful indexing';

