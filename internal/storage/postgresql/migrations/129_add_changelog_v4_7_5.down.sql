-- Rollback migration 129
DELETE FROM changelogs WHERE version = '4.7.5';
