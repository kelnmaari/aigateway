-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ========================================
-- Users Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	full_name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_admin BOOLEAN NOT NULL DEFAULT FALSE,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	verified BOOLEAN NOT NULL DEFAULT FALSE,
	verified_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	last_login TIMESTAMP,
	preferences JSONB NOT NULL DEFAULT '{}', 
	metadata JSONB 
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- ========================================
-- Tenants Table (AUTH-05: Multi-Tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	type TEXT NOT NULL DEFAULT 'personal', -- personal, organization
	description TEXT,
	owner_id TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	settings JSONB NOT NULL DEFAULT '{}', 
	metadata JSONB, 
	FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_owner_id ON tenants(owner_id);
CREATE INDEX idx_tenants_type ON tenants(type);

-- ========================================
-- Tenant Members Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS tenant_members (
	tenant_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
	joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	left_at TIMESTAMP,
	invited_by TEXT,
	metadata JSONB, 
	PRIMARY KEY (tenant_id, user_id),
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tenant_members_user_id ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_role ON tenant_members(role);

-- ========================================
-- API Keys Table (Обновленная для multi-tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	key_hash TEXT NOT NULL UNIQUE,
	user_id TEXT, -- NULL для global keys (backward compatibility)
	tenant_id TEXT, -- NULL для personal keys
	scope TEXT NOT NULL DEFAULT 'global', -- global, personal, tenant
	models JSONB NOT NULL DEFAULT '["*"]', -- JSON array
	permissions JSONB NOT NULL DEFAULT '[]', -- JSON array
	rate_limits JSONB NOT NULL DEFAULT '{}', 
	status TEXT NOT NULL DEFAULT 'active',
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	expires_at TIMESTAMP,
	last_used_at TIMESTAMP,
	revoked_at TIMESTAMP,
	revoked_reason TEXT,
	metadata JSONB, 
	usage JSONB NOT NULL DEFAULT '{}', -- (APIKeyUsage struct)
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_scope ON api_keys(scope);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- ========================================
-- Conversations Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS conversations (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal conversations
	model TEXT NOT NULL,
	temperature REAL,
	system_prompt TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	is_archived BOOLEAN NOT NULL DEFAULT FALSE,
	is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	last_message_at TIMESTAMP,
	message_count INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	metadata JSONB, 
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_tenant_id ON conversations(tenant_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

-- ========================================
-- Messages Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	conversation_id TEXT NOT NULL,
	role TEXT NOT NULL, -- user, assistant, system, tool
	content TEXT NOT NULL,
	model TEXT,
	temperature REAL,
	tool_calls JSONB, -- JSON array
	tool_call_id TEXT,
	prompt_tokens INTEGER DEFAULT 0,
	completion_tokens INTEGER DEFAULT 0,
	total_tokens INTEGER DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	metadata JSONB, 
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_role ON messages(role);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- ========================================
-- API Usage Table (WEBUI-04: Dashboard)
-- ========================================
CREATE TABLE IF NOT EXISTS api_usage (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal usage
	api_key_id TEXT NOT NULL,
	endpoint TEXT NOT NULL,
	method TEXT NOT NULL,
	model TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	success BOOLEAN NOT NULL,
	error_message TEXT,
	prompt_tokens INTEGER DEFAULT 0,
	completion_tokens INTEGER DEFAULT 0,
	total_tokens INTEGER DEFAULT 0,
	duration_ms INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	user_agent TEXT,
	ip_address TEXT,
	conversation_id TEXT,
	metadata JSONB, 
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_api_usage_user_id ON api_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_id ON api_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_api_key_id ON api_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_created_at ON api_usage(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_endpoint ON api_usage(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_usage_model ON api_usage(model);

-- Composite indexes for better query performance (v1.4.6+)
CREATE INDEX IF NOT EXISTS idx_api_usage_user_created ON api_usage(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_created ON api_usage(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;
	