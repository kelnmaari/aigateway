-- Rollback: Remove changelog entry for v1.4.11
DELETE FROM changelogs WHERE version = '1.4.11';
