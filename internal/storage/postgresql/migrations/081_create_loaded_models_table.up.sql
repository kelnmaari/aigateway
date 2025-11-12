-- Migration: Create loaded_models table for persistence (v3.0.6+)
-- This table stores currently loaded GGUF models for auto-loading on server restart

CREATE TABLE IF NOT EXISTS loaded_models (
    id TEXT PRIMARY KEY,
    model_path TEXT NOT NULL UNIQUE,
    alias TEXT NOT NULL UNIQUE,
    loaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    auto_load BOOLEAN NOT NULL DEFAULT TRUE,
    context_size INTEGER,
    batch_size INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookup by alias
CREATE INDEX IF NOT EXISTS idx_loaded_models_alias ON loaded_models(alias);

-- Index for auto_load flag
CREATE INDEX IF NOT EXISTS idx_loaded_models_auto_load ON loaded_models(auto_load);

-- Comment for documentation
COMMENT ON TABLE loaded_models IS 'Stores loaded yzma GGUF models for auto-loading on server restart (v3.0.6+)';
COMMENT ON COLUMN loaded_models.model_path IS 'Relative or absolute path to GGUF model file';
COMMENT ON COLUMN loaded_models.alias IS 'Short alias for the model (e.g., llama-3.2-3b-instruct-q8_0)';
COMMENT ON COLUMN loaded_models.auto_load IS 'If true, model will be loaded automatically on server startup';

