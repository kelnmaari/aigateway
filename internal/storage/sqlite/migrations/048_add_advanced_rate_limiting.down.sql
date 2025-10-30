-- Rollback: Drop rate_limits table
DROP INDEX IF EXISTS idx_rate_limits_scope_target;
DROP INDEX IF EXISTS idx_rate_limits_model;
DROP INDEX IF EXISTS idx_rate_limits_scope;
DROP INDEX IF EXISTS idx_rate_limit_usage_window;
DROP INDEX IF EXISTS idx_rate_limit_usage_cleanup;
DROP TABLE IF EXISTS rate_limits;
