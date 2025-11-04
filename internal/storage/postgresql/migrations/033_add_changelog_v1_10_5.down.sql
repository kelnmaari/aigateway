-- Rollback: Remove changelog entry for v1.10.5
DELETE FROM changelogs WHERE version = '1.10.5';
