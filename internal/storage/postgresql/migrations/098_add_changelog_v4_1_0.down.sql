-- Rollback: Remove changelog entry for v4.1.0
DELETE FROM changelogs WHERE version = '4.1.0';

