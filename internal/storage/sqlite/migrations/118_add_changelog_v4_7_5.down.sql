-- Rollback migration 118
DELETE FROM changelogs WHERE version = '4.7.5';
