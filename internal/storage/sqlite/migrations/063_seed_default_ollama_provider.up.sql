
-- Seed default Ollama provider (local)
-- Используем фиксированный ID для идемпотентности
INSERT OR IGNORE INTO model_providers (
    id, name, provider_type, base_url, 
    enabled, priority, config, 
    health_status, created_at, updated_at
) VALUES (
    'ollama-local-default',
    'ollama-local',
    'ollama',
    'http://localhost:11434',
    1,
    100,
    '{}',
    'unknown',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
);
	