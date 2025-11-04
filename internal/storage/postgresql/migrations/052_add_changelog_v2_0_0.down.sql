-- Rollback: Remove changelog entry for v2.0.0
DELETE FROM changelogs WHERE version = '2.0.0';
