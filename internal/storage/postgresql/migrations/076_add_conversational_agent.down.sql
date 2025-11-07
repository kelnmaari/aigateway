-- Rollback: Remove Conversational Agent support (v2.5.1)

-- Drop agent mode index
DROP INDEX IF EXISTS idx_conversations_agent_mode;

-- Remove agent columns from conversations
-- PostgreSQL supports DROP COLUMN directly
ALTER TABLE conversations DROP COLUMN IF EXISTS agent_context;
ALTER TABLE conversations DROP COLUMN IF EXISTS agent_mode;

-- Note: Messages table role column is unchanged (VARCHAR allows any value)
-- Agent-specific roles (agent_thinking, agent_action, etc.) will simply not be used after rollback

