ALTER TABLE gitlab_projects ADD COLUMN IF NOT EXISTS index_chunks_count BIGINT DEFAULT 0;
