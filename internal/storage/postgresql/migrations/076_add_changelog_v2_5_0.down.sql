-- Rollback: Remove changelog entry for v2.5.0
DELETE FROM changelogs WHERE version = '2.5.0';

