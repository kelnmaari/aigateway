-- Rollback: Remove changelog entry for v1.9.2
DELETE FROM changelogs WHERE version = '1.9.2';
