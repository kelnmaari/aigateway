-- Remove index_chunks_count column from gitlab_projects
ALTER TABLE gitlab_projects DROP COLUMN IF EXISTS index_chunks_count;
