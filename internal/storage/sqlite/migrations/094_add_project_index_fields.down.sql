-- SQLite doesn't support DROP COLUMN, so this is a placeholder
-- For full rollback, recreate the table without these columns
DROP INDEX IF EXISTS idx_gitlab_projects_index_status;

