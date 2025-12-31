-- Rollback: This is a destructive migration that cannot be fully reversed
-- The new nullable schedule_id and additional columns cannot be removed without data loss

-- Drop the new indexes
DROP INDEX IF EXISTS idx_scan_history_scan_type;
DROP INDEX IF EXISTS idx_scan_history_integration;

-- Note: Full rollback would require recreating the table with NOT NULL schedule_id
-- which would delete all manual scan entries. This is intentionally left as-is.

