
-- Model Configurations table for dynamic model parameters (v1.9.1)
CREATE TABLE IF NOT EXISTS model_configs (
    id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    scope TEXT NOT NULL CHECK(scope IN ('global', 'tenant', 'user')),
    tenant_id TEXT,
    user_id TEXT,
    created_by TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    parameters TEXT NOT NULL, -- JSON: ModelParameters
    
    -- Foreign keys
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (
        (scope = 'global' AND tenant_id IS NULL AND user_id IS NULL) OR
        (scope = 'tenant' AND tenant_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    ),
    
    -- Unique constraint для предотвращения дубликатов
    UNIQUE(model_name, scope, tenant_id, user_id)
);

-- Indexes для быстрого поиска config по scope
CREATE INDEX IF NOT EXISTS idx_model_configs_model ON model_configs(model_name);
CREATE INDEX IF NOT EXISTS idx_model_configs_scope ON model_configs(scope);
CREATE INDEX IF NOT EXISTS idx_model_configs_tenant ON model_configs(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_model_configs_user ON model_configs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_model_configs_lookup ON model_configs(model_name, scope, tenant_id, user_id);

-- Trigger function для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_model_configs_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger для автоматического обновления updated_at
DROP TRIGGER IF EXISTS update_model_configs_timestamp_trigger ON model_configs;
CREATE TRIGGER update_model_configs_timestamp_trigger
    BEFORE UPDATE ON model_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_model_configs_timestamp();
	