-- Rollback: Remove changelog entry for v2.2.0
DELETE FROM changelogs WHERE version = '2.2.0';
