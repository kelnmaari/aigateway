-- Rollback: Remove changelog entry for v3.3.0
DELETE FROM changelogs WHERE version = '3.3.0';

