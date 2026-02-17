-- Add 'deepseek' to valid_provider_type CHECK constraint
ALTER TABLE model_providers DROP CONSTRAINT IF EXISTS valid_provider_type;
ALTER TABLE model_providers ADD CONSTRAINT valid_provider_type
    CHECK (provider_type IN ('vllm', 'openai', 'anthropic', 'gemini', 'deepseek', 'custom'));
