-- Rollback migration 129
DELETE FROM system_changelog WHERE version = '4.7.5';
