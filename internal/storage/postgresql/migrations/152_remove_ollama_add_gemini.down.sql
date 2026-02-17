-- Rollback migration 152: Restore Ollama provider type

-- Restore the CHECK constraint with ollama, remove gemini
ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('ollama', 'vllm', 'openai', 'anthropic', 'custom'));

-- Re-seed default Ollama provider
INSERT INTO model_providers (id, name, provider_type, base_url, is_active, health_status, created_at, updated_at)
VALUES (
    'default-ollama',
    'Local Ollama',
    'ollama',
    'http://localhost:11434',
    true,
    'unknown',
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
