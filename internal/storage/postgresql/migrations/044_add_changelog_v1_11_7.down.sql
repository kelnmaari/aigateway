-- Rollback: Remove changelog entry for v1.11.7
DELETE FROM changelogs WHERE version = '1.11.7';
