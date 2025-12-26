-- Rollback: Remove system user
-- WARNING: This will cascade delete all API keys owned by system user

DELETE FROM users WHERE id = 'system';

