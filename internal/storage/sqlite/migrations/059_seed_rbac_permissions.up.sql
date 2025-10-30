
-- Seed RBAC Permissions (v2.2.0+)
INSERT OR IGNORE INTO permissions (id, name, description, resource, action, scope, created_at) VALUES
-- API Keys permissions
('perm_api_keys_create', 'api_keys:create', 'Create API keys', 'api_keys', 'create', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_read', 'api_keys:read', 'View API keys', 'api_keys', 'read', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_update', 'api_keys:update', 'Update API keys', 'api_keys', 'update', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_delete', 'api_keys:delete', 'Delete API keys', 'api_keys', 'delete', 'global', CURRENT_TIMESTAMP),

-- Users permissions
('perm_users_create', 'users:create', 'Create users', 'users', 'create', 'global', CURRENT_TIMESTAMP),
('perm_users_read', 'users:read', 'View users', 'users', 'read', 'global', CURRENT_TIMESTAMP),
('perm_users_update', 'users:update', 'Update users', 'users', 'update', 'global', CURRENT_TIMESTAMP),
('perm_users_delete', 'users:delete', 'Delete users', 'users', 'delete', 'global', CURRENT_TIMESTAMP),

-- Tenants permissions
('perm_tenants_create', 'tenants:create', 'Create tenants', 'tenants', 'create', 'global', CURRENT_TIMESTAMP),
('perm_tenants_read', 'tenants:read', 'View tenants', 'tenants', 'read', 'global', CURRENT_TIMESTAMP),
('perm_tenants_update', 'tenants:update', 'Update tenants', 'tenants', 'update', 'global', CURRENT_TIMESTAMP),
('perm_tenants_delete', 'tenants:delete', 'Delete tenants', 'tenants', 'delete', 'global', CURRENT_TIMESTAMP),

-- Chat / Models permissions
('perm_chat_use', 'chat:use', 'Use chat interface', 'chat', 'use', 'global', CURRENT_TIMESTAMP),
('perm_models_read', 'models:read', 'View available models', 'models', 'read', 'global', CURRENT_TIMESTAMP),
('perm_models_manage', 'models:manage', 'Manage models', 'models', 'manage', 'global', CURRENT_TIMESTAMP),

-- System permissions
('perm_system_config', 'system:config', 'Configure system settings', 'system', 'config', 'global', CURRENT_TIMESTAMP),
('perm_system_backup', 'system:backup', 'Create system backups', 'system', 'backup', 'global', CURRENT_TIMESTAMP),
('perm_system_logs', 'system:logs', 'View system logs', 'system', 'logs', 'global', CURRENT_TIMESTAMP),
('perm_audit_read', 'audit:read', 'View audit logs', 'audit', 'read', 'global', CURRENT_TIMESTAMP),

-- Files permissions
('perm_files_upload', 'files:upload', 'Upload files', 'files', 'upload', 'global', CURRENT_TIMESTAMP),
('perm_files_read', 'files:read', 'View files', 'files', 'read', 'global', CURRENT_TIMESTAMP),
('perm_files_delete', 'files:delete', 'Delete files', 'files', 'delete', 'global', CURRENT_TIMESTAMP),

-- Conversations permissions
('perm_conversations_create', 'conversations:create', 'Create conversations', 'conversations', 'create', 'global', CURRENT_TIMESTAMP),
('perm_conversations_read', 'conversations:read', 'View conversations', 'conversations', 'read', 'global', CURRENT_TIMESTAMP),
('perm_conversations_update', 'conversations:update', 'Update conversations', 'conversations', 'update', 'global', CURRENT_TIMESTAMP),
('perm_conversations_delete', 'conversations:delete', 'Delete conversations', 'conversations', 'delete', 'global', CURRENT_TIMESTAMP),

-- Invitations permissions (v2.2.0+)
('perm_invitations_create', 'invitations:create', 'Create invitations', 'invitations', 'create', 'global', CURRENT_TIMESTAMP),
('perm_invitations_read', 'invitations:read', 'View invitations', 'invitations', 'read', 'global', CURRENT_TIMESTAMP),
('perm_invitations_revoke', 'invitations:revoke', 'Revoke invitations', 'invitations', 'revoke', 'global', CURRENT_TIMESTAMP),

-- RBAC permissions
('perm_roles_create', 'roles:create', 'Create roles', 'roles', 'create', 'global', CURRENT_TIMESTAMP),
('perm_roles_read', 'roles:read', 'View roles', 'roles', 'read', 'global', CURRENT_TIMESTAMP),
('perm_roles_update', 'roles:update', 'Update roles', 'roles', 'update', 'global', CURRENT_TIMESTAMP),
('perm_roles_delete', 'roles:delete', 'Delete roles', 'roles', 'delete', 'global', CURRENT_TIMESTAMP),
('perm_roles_assign', 'roles:assign', 'Assign roles to users', 'roles', 'assign', 'global', CURRENT_TIMESTAMP),

-- Wildcard permission (Super Admin)
('perm_all', '*:*', 'All permissions (Super Admin)', '*', '*', 'global', CURRENT_TIMESTAMP);
	