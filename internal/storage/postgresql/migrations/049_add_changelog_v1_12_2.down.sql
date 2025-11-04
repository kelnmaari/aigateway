-- Rollback: Remove changelog entry for v1.12.2
DELETE FROM changelogs WHERE version = '1.12.2';
