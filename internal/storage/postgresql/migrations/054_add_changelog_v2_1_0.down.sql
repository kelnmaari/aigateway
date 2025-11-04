-- Rollback: Remove changelog entry for v2.1.0
DELETE FROM changelogs WHERE version = '2.1.0';
