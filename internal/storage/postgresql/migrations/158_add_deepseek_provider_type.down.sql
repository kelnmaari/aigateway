-- Remove 'deepseek' from valid_provider_type CHECK constraint
DELETE FROM model_registry WHERE provider_id IN (
    SELECT id FROM model_providers WHERE provider_type = 'deepseek'
);
DELETE FROM model_providers WHERE provider_type = 'deepseek';

ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('vllm', 'openai', 'anthropic', 'gemini', 'custom'));
