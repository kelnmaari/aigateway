-- Rollback: Remove changelog entry for v4.1.1
DELETE FROM changelogs WHERE version = '4.1.1';

