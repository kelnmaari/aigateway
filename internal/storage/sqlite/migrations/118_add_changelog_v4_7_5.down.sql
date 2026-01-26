-- Rollback migration 118
DELETE FROM system_changelog WHERE version = '4.7.5';
