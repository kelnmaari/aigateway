-- Rollback: Remove changelog entry for v4.2.0
DELETE FROM changelogs WHERE version = '4.2.0';

