-- Rollback: Remove changelog entry for v2.4.4
DELETE FROM changelogs WHERE version = '2.4.4';
