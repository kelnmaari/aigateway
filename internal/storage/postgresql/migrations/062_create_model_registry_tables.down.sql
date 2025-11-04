-- Rollback: Drop model_registry and model_providers tables (in correct order)

-- Drop triggers first
DROP TRIGGER IF EXISTS update_model_registry_timestamp_trigger ON model_registry;
DROP TRIGGER IF EXISTS update_model_providers_timestamp_trigger ON model_providers;

-- Drop trigger functions
DROP FUNCTION IF EXISTS update_model_registry_timestamp();
DROP FUNCTION IF EXISTS update_model_providers_timestamp();

-- Drop indexes for model_registry
DROP INDEX IF EXISTS idx_model_registry_provider;
DROP INDEX IF EXISTS idx_model_registry_status;
DROP INDEX IF EXISTS idx_model_registry_health;
DROP INDEX IF EXISTS idx_model_registry_model_id;

-- Drop model_registry table (dependent table first)
DROP TABLE IF EXISTS model_registry;

-- Drop indexes for model_providers
DROP INDEX IF EXISTS idx_model_providers_type;
DROP INDEX IF EXISTS idx_model_providers_enabled;
DROP INDEX IF EXISTS idx_model_providers_priority;

-- Drop model_providers table (parent table last)
DROP TABLE IF EXISTS model_providers;
