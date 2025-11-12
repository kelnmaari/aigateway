-- Rollback: Drop loaded_models table
-- This removes persistence of loaded models

DROP INDEX IF EXISTS idx_loaded_models_auto_load;
DROP INDEX IF EXISTS idx_loaded_models_alias;
DROP TABLE IF EXISTS loaded_models;

