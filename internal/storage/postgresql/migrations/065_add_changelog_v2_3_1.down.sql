-- Rollback: Remove changelog entry for v2.3.1
DELETE FROM changelogs WHERE version = '2.3.1';
