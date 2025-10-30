-- Rollback: Remove changelog entry for v2.3.0
DELETE FROM changelogs WHERE version = '2.3.0';
