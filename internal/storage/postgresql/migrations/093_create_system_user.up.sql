-- Migration: Create system user for internal services
-- This user owns system-generated API keys (GitLab workers, etc.)
-- Password is set to impossible bcrypt hash (cannot login via password)
-- is_admin = true gives admin-level permissions

INSERT INTO users (id, email, username, full_name, password_hash, status, is_admin, is_active, verified, created_at, updated_at)
VALUES (
    'system',
    'system@internal.local',
    '_system_',
    'System (Internal Services)',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX', -- Invalid bcrypt, cannot login
    'active',
    TRUE,  -- Admin permissions
    TRUE,
    TRUE,  -- Pre-verified
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

