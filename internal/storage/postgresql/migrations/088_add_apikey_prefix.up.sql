-- Migration 088: Add key_prefix column to api_keys table
-- This stores the first ~25 characters of the API key for display purposes

ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS key_prefix TEXT DEFAULT '';

