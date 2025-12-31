-- Rollback: Remove changelog entry for v4.2.3
DELETE FROM changelogs WHERE version = '4.2.3';

