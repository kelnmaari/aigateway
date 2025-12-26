-- Rollback: Remove changelog entry for v4.0.2
DELETE FROM changelogs WHERE version = '4.0.2';

