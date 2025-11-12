-- Rollback: Drop model downloads tracking table
DROP TRIGGER IF EXISTS trigger_update_model_downloads_updated_at ON model_downloads;
DROP FUNCTION IF EXISTS update_model_downloads_updated_at();
DROP INDEX IF EXISTS idx_model_downloads_created_at;
DROP INDEX IF EXISTS idx_model_downloads_status;
DROP INDEX IF EXISTS idx_model_downloads_model_id;
DROP TABLE IF EXISTS model_downloads;

