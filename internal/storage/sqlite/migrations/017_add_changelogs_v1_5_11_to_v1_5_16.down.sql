-- Rollback: Remove changelog entry for v1.5.16
DELETE FROM changelogs WHERE version = '1.5.16';
