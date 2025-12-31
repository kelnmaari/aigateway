-- Rollback: Remove changelog entry for v4.2.1
DELETE FROM changelogs WHERE version = '4.2.1';

