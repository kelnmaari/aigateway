-- Rollback: Drop model downloads tracking table
DROP INDEX IF EXISTS idx_model_downloads_started_at;
DROP INDEX IF EXISTS idx_model_downloads_model_id;
DROP INDEX IF EXISTS idx_model_downloads_huggingface_id;
DROP INDEX IF EXISTS idx_model_downloads_status;
DROP TABLE IF EXISTS model_downloads;

