-- Migration: Create system user for internal services
-- This user owns system-generated API keys (GitLab workers, etc.)
-- Password is set to impossible bcrypt hash (cannot login via password)
-- is_admin = 1 gives admin-level permissions

INSERT OR IGNORE INTO users (id, email, username, full_name, password_hash, status, is_admin, is_active, verified, created_at, updated_at)
VALUES (
    'system',
    'system@internal.local',
    '_system_',
    'System (Internal Services)',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX', -- Invalid bcrypt, cannot login
    'active',
    1,  -- Admin permissions (SQLite uses 1/0 for boolean)
    1,
    1,  -- Pre-verified
    datetime('now'),
    datetime('now')
);

