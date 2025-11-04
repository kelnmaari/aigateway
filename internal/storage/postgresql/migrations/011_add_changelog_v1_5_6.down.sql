-- Rollback: Remove changelog entry for v1.5.6
DELETE FROM changelogs WHERE version = '1.5.6';
