
-- ========================================
-- Fix API Usage Table: Nullable api_key_id (Migration v6: BUG-03 v1.5.12)
-- ========================================
-- Problem: FOREIGN KEY constraint failed при использовании WebUI чата
-- Solution: Сделать api_key_id nullable с ON DELETE SET NULL

-- Шаг 1: Создаем временную таблицу с правильной структурой
CREATE TABLE api_usage_new (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal usage
	api_key_id TEXT, -- ✅ NULLABLE для JWT auth (было NOT NULL)
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
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	user_agent TEXT,
	ip_address TEXT,
	conversation_id TEXT,
	metadata TEXT, -- JSON
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL, -- ✅ SET NULL вместо CASCADE
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

-- Шаг 2: Копируем существующие данные
-- Фильтруем записи с несуществующими api_key_id ("jwt_auth", "unknown")
INSERT INTO api_usage_new 
SELECT 
	id,
	user_id,
	tenant_id,
	CASE 
		WHEN api_key_id IN ('jwt_auth', 'unknown') THEN NULL
		ELSE api_key_id
	END as api_key_id,
	endpoint,
	method,
	model,
	status_code,
	success,
	error_message,
	prompt_tokens,
	completion_tokens,
	total_tokens,
	duration_ms,
	created_at,
	user_agent,
	ip_address,
	conversation_id,
	metadata
FROM api_usage;

-- Шаг 3: Удаляем старую таблицу
DROP TABLE api_usage;

-- Шаг 4: Переименовываем новую таблицу
ALTER TABLE api_usage_new RENAME TO api_usage;

-- Шаг 5: Пересоздаем индексы
CREATE INDEX IF NOT EXISTS idx_api_usage_user_id ON api_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_id ON api_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_api_key_id ON api_usage(api_key_id) WHERE api_key_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_api_usage_created_at ON api_usage(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_endpoint ON api_usage(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_usage_model ON api_usage(model);
CREATE INDEX IF NOT EXISTS idx_api_usage_user_created ON api_usage(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_created ON api_usage(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;
	