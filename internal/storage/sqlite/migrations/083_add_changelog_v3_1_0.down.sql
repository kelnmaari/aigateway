-- Rollback: Remove changelog entry for v3.1.0
DELETE FROM changelogs WHERE version = '3.1.0';

