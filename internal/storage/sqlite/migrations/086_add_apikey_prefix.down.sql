-- Rollback: Remove key_prefix column
-- Note: SQLite doesn't support DROP COLUMN directly in older versions
-- This migration is safe to leave as-is since the column is optional

-- For SQLite 3.35.0+:
-- ALTER TABLE api_keys DROP COLUMN key_prefix;

-- For older SQLite, recreate table without the column (not implemented here)
SELECT 1; -- No-op for compatibility

