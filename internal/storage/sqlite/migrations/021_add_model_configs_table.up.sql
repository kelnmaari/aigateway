
-- Model Configurations table for dynamic model parameters (v1.9.1)
CREATE TABLE IF NOT EXISTS model_configs (
    id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    scope TEXT NOT NULL CHECK(scope IN ('global', 'tenant', 'user')),
    tenant_id TEXT,
    user_id TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
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

-- Trigger для автоматического обновления updated_at
CREATE TRIGGER IF NOT EXISTS update_model_configs_timestamp 
AFTER UPDATE ON model_configs
FOR EACH ROW
BEGIN
    UPDATE model_configs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
	