-- Rollback: Drop settings table
DROP INDEX IF EXISTS idx_settings_updated_at;
DROP INDEX IF EXISTS idx_settings_requires_restart;
DROP INDEX IF EXISTS idx_settings_is_migrated;
DROP INDEX IF EXISTS idx_settings_is_editable;
DROP INDEX IF EXISTS idx_settings_category;
DROP TABLE IF EXISTS settings;

