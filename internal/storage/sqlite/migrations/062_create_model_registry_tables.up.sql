
-- Model Providers Table
-- Хранит конфигурацию для каждого model provider (Ollama, vLLM, OpenAI, etc)
CREATE TABLE IF NOT EXISTS model_providers (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT UNIQUE NOT NULL,              -- "ollama-local", "vllm-primary", "openai"
    provider_type TEXT NOT NULL,            -- "ollama", "vllm", "openai", "anthropic", "custom"
    base_url TEXT NOT NULL,                 -- "http://localhost:11434"
    api_key TEXT,                           -- Encrypted (для OpenAI, Anthropic)
    
    -- Configuration
    enabled BOOLEAN DEFAULT 1,              -- Provider включен/выключен
    priority INTEGER DEFAULT 100,           -- Higher = preferred (для fallback)
    config TEXT DEFAULT '{}',               -- JSON: Provider-specific config
    
    -- Health tracking
    health_status TEXT DEFAULT 'unknown',   -- "healthy", "unhealthy", "unknown"
    last_health_check TIMESTAMP,
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT valid_provider_type CHECK (provider_type IN ('ollama', 'vllm', 'openai', 'anthropic', 'custom'))
);

CREATE INDEX idx_model_providers_type ON model_providers(provider_type);
CREATE INDEX idx_model_providers_enabled ON model_providers(enabled);
CREATE INDEX idx_model_providers_priority ON model_providers(priority DESC);

-- Model Registry Table
-- Универсальный реестр всех моделей из всех providers
CREATE TABLE IF NOT EXISTS model_registry (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    
    -- Model identification
    model_id TEXT UNIQUE NOT NULL,          -- "llama2:7b", "mistral-7b-instruct", "gpt-4"
    model_name TEXT NOT NULL,               -- Display name
    provider_id TEXT NOT NULL,              -- FK to model_providers
    
    -- Capabilities (JSON array)
    capabilities TEXT DEFAULT '[]',         -- ["chat", "embeddings", "vision", "function-calling"]
    parameters TEXT DEFAULT '{}',           -- JSON: Model-specific parameters (max_tokens, etc)
    
    -- Requirements
    requires_gpu BOOLEAN DEFAULT 0,
    min_vram_gb INTEGER,
    context_length INTEGER,
    
    -- Status
    status TEXT DEFAULT 'active',           -- "active", "inactive", "loading", "error"
    health_status TEXT DEFAULT 'unknown',   -- "healthy", "unhealthy", "unknown"
    last_health_check TIMESTAMP,
    
    -- Metadata
    description TEXT,
    tags TEXT,                              -- Comma-separated: "opensource,7b,instruct"
    
    -- Performance metrics (updated periodically)
    avg_latency_ms REAL,
    tokens_per_second REAL,
    total_requests INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    FOREIGN KEY (provider_id) REFERENCES model_providers(id) ON DELETE CASCADE,
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'loading', 'error')),
    CONSTRAINT valid_health CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
);

CREATE INDEX idx_model_registry_provider ON model_registry(provider_id);
CREATE INDEX idx_model_registry_status ON model_registry(status);
CREATE INDEX idx_model_registry_health ON model_registry(health_status);
CREATE INDEX idx_model_registry_model_id ON model_registry(model_id);

-- Triggers для auto-update updated_at
CREATE TRIGGER update_model_providers_timestamp 
AFTER UPDATE ON model_providers
FOR EACH ROW
BEGIN
    UPDATE model_providers SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_model_registry_timestamp 
AFTER UPDATE ON model_registry
FOR EACH ROW
BEGIN
    UPDATE model_registry SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
	