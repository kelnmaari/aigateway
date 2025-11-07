-- Migration 076: Add Conversational Agent support to conversations and messages
-- Version: 2.5.1
-- Description: Enables agent mode in chat conversations with reasoning, actions, and observations

-- ========================================
-- Conversations: Add Agent Mode
-- ========================================

-- Enable agent mode flag for conversations
ALTER TABLE conversations ADD COLUMN agent_mode BOOLEAN NOT NULL DEFAULT FALSE;

-- Store agent context (ReAct loop state, tools used, etc.)
ALTER TABLE conversations ADD COLUMN agent_context JSONB;

-- Index for querying agent-enabled conversations
CREATE INDEX idx_conversations_agent_mode ON conversations(agent_mode) WHERE agent_mode = TRUE;

-- ========================================
-- Messages: Extend roles for Agent
-- ========================================

-- Messages table already has TEXT/VARCHAR role column, so we just document new valid values:
-- Existing roles: user, assistant, system, tool
-- NEW Agent roles (v2.5.1+):
--   - agent_thinking: Agent's reasoning/thought process (collapsible in UI)
--   - agent_action: Agent tool execution (file operations, terminal commands)
--   - agent_observation: Agent's analysis of action results
--   - agent_approval: Agent requesting user approval for dangerous operations

-- No schema change needed for messages table - role is already VARCHAR
-- New roles will be validated at application layer

-- ========================================
-- Notes:
-- ========================================
-- 1. agent_mode=TRUE enables conversational agent for the conversation
-- 2. agent_context stores ReAct loop state as JSONB:
--    {
--      "current_task": "...",
--      "thought_history": [...],
--      "tools_used": [...],
--      "pending_approvals": [...]
--    }
-- 3. Message roles extended but schema unchanged (VARCHAR allows any value)
-- 4. JSONB used for efficient querying of agent context

