-- Rollback: Remove changelog entry for v1.10.0
DELETE FROM changelogs WHERE version = '1.10.0';
