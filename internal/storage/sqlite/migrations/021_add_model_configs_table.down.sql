-- Rollback: Drop model_configs table
DROP INDEX IF EXISTS idx_model_configs_model;
DROP INDEX IF EXISTS idx_model_configs_scope;
DROP INDEX IF EXISTS idx_model_configs_tenant;
DROP INDEX IF EXISTS idx_model_configs_user;
DROP INDEX IF EXISTS idx_model_configs_lookup;
DROP TABLE IF EXISTS model_configs;
