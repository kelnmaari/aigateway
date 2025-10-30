
-- Create Quotas tables for Usage Quotas System (Version 1.11.7+)

-- Quotas table: defines limits for users or tenants
CREATE TABLE IF NOT EXISTS quotas (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    scope TEXT NOT NULL, -- 'user', 'tenant'
    target_id TEXT NOT NULL, -- user_id or tenant_id
    
    -- Token limits
    tokens_per_day INTEGER,
    tokens_per_month INTEGER,
    
    -- Request limits
    requests_per_day INTEGER,
    requests_per_month INTEGER,
    max_concurrent INTEGER,
    
    -- Storage limits
    max_storage_bytes INTEGER,
    max_conversations INTEGER,
    max_file_size INTEGER,
    
    -- Model restrictions (JSON array of model names)
    allowed_models TEXT,
    
    -- Metadata
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(scope, target_id)
);

-- Quota usage table: tracks current usage against quotas
CREATE TABLE IF NOT EXISTS quota_usage (
    id TEXT PRIMARY KEY,
    quota_id TEXT NOT NULL,
    target_id TEXT NOT NULL, -- user_id or tenant_id (matches quota)
    
    -- Current usage counters
    tokens_used_today INTEGER NOT NULL DEFAULT 0,
    tokens_used_month INTEGER NOT NULL DEFAULT 0,
    requests_today INTEGER NOT NULL DEFAULT 0,
    requests_month INTEGER NOT NULL DEFAULT 0,
    current_concurrent INTEGER NOT NULL DEFAULT 0,
    storage_used_bytes INTEGER NOT NULL DEFAULT 0,
    conversations_count INTEGER NOT NULL DEFAULT 0,
    
    -- Reset timestamps
    last_daily_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_monthly_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (quota_id) REFERENCES quotas(id) ON DELETE CASCADE,
    UNIQUE(quota_id, target_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_quotas_target ON quotas(scope, target_id) WHERE enabled = 1;
CREATE INDEX IF NOT EXISTS idx_quotas_scope ON quotas(scope);
CREATE INDEX IF NOT EXISTS idx_quota_usage_target ON quota_usage(target_id);
CREATE INDEX IF NOT EXISTS idx_quota_usage_quota ON quota_usage(quota_id);
	