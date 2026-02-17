-- Migration 152: Remove Ollama provider, add Gemini provider type
-- Part of AIGateway transition from Ollama to multi-provider architecture

-- Remove the default Ollama provider (seeded in migration 063)
DELETE FROM model_registry WHERE provider_id IN (
    SELECT id FROM model_providers WHERE provider_type = 'ollama'
);
DELETE FROM model_providers WHERE provider_type = 'ollama';

-- Drop and recreate the CHECK constraint to remove 'ollama' and add 'gemini'
ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('vllm', 'openai', 'anthropic', 'gemini', 'custom'));
