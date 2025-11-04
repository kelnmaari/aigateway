-- Rollback: Remove changelog entry for v2.2.1
DELETE FROM changelogs WHERE version = '2.2.1';
