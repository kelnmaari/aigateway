-- Rollback: Remove changelog entry for v1.10.3
DELETE FROM changelogs WHERE version = '1.10.3';
