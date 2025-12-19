-- Rollback: Remove key_prefix column
ALTER TABLE api_keys DROP COLUMN IF EXISTS key_prefix;

