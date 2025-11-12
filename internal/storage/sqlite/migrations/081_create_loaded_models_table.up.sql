-- Migration: Create loaded_models table for persistence (v3.0.6+)
-- This table stores currently loaded GGUF models for auto-loading on server restart

CREATE TABLE IF NOT EXISTS loaded_models (
    id TEXT PRIMARY KEY,
    model_path TEXT NOT NULL UNIQUE,
    alias TEXT NOT NULL UNIQUE,
    loaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    auto_load BOOLEAN NOT NULL DEFAULT 1,
    context_size INTEGER,
    batch_size INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast lookup by alias
CREATE INDEX IF NOT EXISTS idx_loaded_models_alias ON loaded_models(alias);

-- Index for auto_load flag
CREATE INDEX IF NOT EXISTS idx_loaded_models_auto_load ON loaded_models(auto_load);

