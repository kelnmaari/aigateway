-- Rollback: Remove Conversational Agent support (v2.5.1)

-- Drop agent mode index
DROP INDEX IF EXISTS idx_conversations_agent_mode;

-- Remove agent columns from conversations
-- Note: SQLite doesn't support DROP COLUMN directly in older versions
-- For newer SQLite (3.35.0+), use:
-- ALTER TABLE conversations DROP COLUMN agent_context;
-- ALTER TABLE conversations DROP COLUMN agent_mode;

-- Workaround for older SQLite: Recreate table without agent columns
-- This is safe if agent_mode was not used yet, or data loss is acceptable
PRAGMA foreign_keys=OFF;

BEGIN TRANSACTION;

-- Create new table without agent columns
CREATE TABLE conversations_new (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    model TEXT NOT NULL,
    temperature REAL,
    system_prompt TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    is_archived BOOLEAN NOT NULL DEFAULT 0,
    is_pinned BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_message_at TIMESTAMP,
    message_count INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Copy data (excluding agent columns)
INSERT INTO conversations_new
SELECT 
    id, title, user_id, tenant_id, model, temperature, system_prompt,
    status, is_archived, is_pinned, created_at, updated_at, last_message_at,
    message_count, total_tokens, metadata
FROM conversations;

-- Drop old table
DROP TABLE conversations;

-- Rename new table
ALTER TABLE conversations_new RENAME TO conversations;

-- Recreate indexes
CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_tenant_id ON conversations(tenant_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

COMMIT;

PRAGMA foreign_keys=ON;

-- Note: Messages table role column is unchanged (TEXT allows any value)
-- Agent-specific roles (agent_thinking, agent_action, etc.) will simply not be used after rollback

