
-- Web Fetches Table for storing fetched web pages
CREATE TABLE IF NOT EXISTS web_fetches (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    
    -- URL info
    url TEXT NOT NULL,
    url_hash TEXT NOT NULL,
    domain TEXT NOT NULL,
    
    -- Content
    title TEXT,
    content TEXT NOT NULL,
    html_content TEXT,
    
    -- Metadata (JSON)
    metadata TEXT,
    
    -- Links
    links TEXT,
    
    -- LLM processing
    summary TEXT,
    summary_model TEXT,
    
    -- Fetch info
    fetch_status TEXT DEFAULT 'success',
    fetch_time_ms INTEGER,
    status_code INTEGER,
    content_type TEXT,
    content_length INTEGER,
    fetch_error TEXT,
    
    -- Language detection
    language TEXT,
    word_count INTEGER,
    
    -- Cache
    expires_at TIMESTAMP,
    is_cached INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Indexes for web_fetches
CREATE INDEX IF NOT EXISTS idx_web_fetches_url_hash ON web_fetches(url_hash);
CREATE INDEX IF NOT EXISTS idx_web_fetches_user ON web_fetches(user_id);
CREATE INDEX IF NOT EXISTS idx_web_fetches_tenant ON web_fetches(tenant_id);
CREATE INDEX IF NOT EXISTS idx_web_fetches_domain ON web_fetches(domain);
CREATE INDEX IF NOT EXISTS idx_web_fetches_expires ON web_fetches(expires_at);
CREATE INDEX IF NOT EXISTS idx_web_fetches_created ON web_fetches(created_at DESC);

-- Web Fetch Rate Limits Table
CREATE TABLE IF NOT EXISTS web_fetch_rate_limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    requests_per_minute INTEGER DEFAULT 10,
    last_request_at TIMESTAMP,
    request_count INTEGER DEFAULT 0,
    blocked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_domain ON web_fetch_rate_limits(domain);
	