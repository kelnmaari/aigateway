-- Rollback: Remove changelog entry for v3.2.0
DELETE FROM changelogs WHERE version = '3.2.0';

