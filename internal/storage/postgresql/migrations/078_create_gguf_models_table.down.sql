-- Rollback: Drop GGUF models tracking table
DROP TRIGGER IF EXISTS trigger_update_gguf_models_updated_at ON gguf_models;
DROP FUNCTION IF EXISTS update_gguf_models_updated_at();
DROP INDEX IF EXISTS idx_gguf_models_is_active;
DROP INDEX IF EXISTS idx_gguf_models_huggingface_id;
DROP INDEX IF EXISTS idx_gguf_models_is_vlm;
DROP INDEX IF EXISTS idx_gguf_models_architecture;
DROP INDEX IF EXISTS idx_gguf_models_name;
DROP TABLE IF EXISTS gguf_models;

