
-- Advanced Rate Limits Table (v1.12.2+: RATE-02)
CREATE TABLE IF NOT EXISTS rate_limits (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- Scope: "global", "tenant", "user", "api_key", "model"
    scope TEXT NOT NULL DEFAULT 'global',
    
    -- Target ID: depends on scope (user_id, tenant_id, api_key_id, model_name, "all" для global)
    target_id TEXT,
    
    -- Model-specific rate limit (optional)
    model_name TEXT,
    
    -- Rate limits (NULL = no limit)
    requests_per_second INTEGER,
    requests_per_minute INTEGER,
    requests_per_hour INTEGER,
    requests_per_day INTEGER,
    
    -- Burst allowance
    burst_size INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_rate_limits_scope_target ON rate_limits(scope, target_id);
CREATE INDEX IF NOT EXISTS idx_rate_limits_model ON rate_limits(model_name) WHERE model_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rate_limits_scope ON rate_limits(scope);

-- Rate Limit Usage Tracking (for sliding window)
CREATE TABLE IF NOT EXISTS rate_limit_usage (
    id TEXT PRIMARY KEY,
    rate_limit_id TEXT NOT NULL,
    window_type TEXT NOT NULL, -- "second", "minute", "hour", "day"
    window_start TIMESTAMP NOT NULL,
    request_count INTEGER DEFAULT 0,
    last_request_at TIMESTAMP,
    
    FOREIGN KEY (rate_limit_id) REFERENCES rate_limits(id) ON DELETE CASCADE
);

-- Index for sliding window queries
CREATE INDEX IF NOT EXISTS idx_rate_limit_usage_window ON rate_limit_usage(rate_limit_id, window_type, window_start DESC);
CREATE INDEX IF NOT EXISTS idx_rate_limit_usage_cleanup ON rate_limit_usage(window_start);
    