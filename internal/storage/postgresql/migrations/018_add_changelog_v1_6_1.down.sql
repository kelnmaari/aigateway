-- Rollback: Remove changelog entry for v1.6.1
DELETE FROM changelogs WHERE version = '1.6.1';
