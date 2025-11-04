-- Rollback: Drop quotas table
DROP INDEX IF EXISTS idx_quotas_target;
DROP INDEX IF EXISTS idx_quotas_scope;
DROP INDEX IF EXISTS idx_quota_usage_target;
DROP INDEX IF EXISTS idx_quota_usage_quota;
DROP TABLE IF EXISTS quotas;
