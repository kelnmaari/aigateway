-- Rollback: Remove changelog entry for v4.0.0
DELETE FROM changelogs WHERE version = '4.0.0';

