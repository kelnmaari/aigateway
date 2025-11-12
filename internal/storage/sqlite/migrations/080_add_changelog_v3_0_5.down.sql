-- Rollback: Remove changelog entry for v3.0.5
DELETE FROM changelogs WHERE version = '3.0.5';

