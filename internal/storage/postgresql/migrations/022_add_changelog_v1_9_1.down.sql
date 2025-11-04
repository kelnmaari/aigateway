-- Rollback: Remove changelog entry for v1.9.1
DELETE FROM changelogs WHERE version = '1.9.1';
