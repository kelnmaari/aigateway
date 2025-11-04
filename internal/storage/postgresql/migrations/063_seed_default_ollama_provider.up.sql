
-- Seed default Ollama provider (local)
-- Используем фиксированный ID для идемпотентности
INSERT INTO model_providers (
    id, name, provider_type, base_url, 
    enabled, priority, config, 
    health_status, created_at, updated_at
) VALUES (
    'ollama-local-default',
    'ollama-local',
    'ollama',
    'http://localhost:11434',
    TRUE,
    100,
    '{}',
    'unknown',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (id) DO NOTHING;
	