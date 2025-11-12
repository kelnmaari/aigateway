-- Create GGUF models tracking table
CREATE TABLE IF NOT EXISTS gguf_models (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT,
    architecture TEXT,      -- llama, qwen, gemma, phi
    quantization TEXT,       -- Q4_K_M, Q8_0, F16
    parameters_count BIGINT, -- 7B, 13B, 70B (in millions)
    context_size INTEGER,    -- 2048, 4096, 8192, 32768
    is_vlm BOOLEAN DEFAULT FALSE,
    mmproj_path TEXT,        -- Path to multimodal projector for VLMs
    huggingface_id TEXT,     -- Original Hugging Face model ID (e.g., "Qwen/Qwen2.5-VL-7B-Instruct-GGUF")
    huggingface_file TEXT,   -- Specific file from HF (e.g., "qwen2.5-vl-7b-instruct-q4_k_m.gguf")
    downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_loaded_at TIMESTAMP,
    load_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_gguf_models_name ON gguf_models(name);
CREATE INDEX IF NOT EXISTS idx_gguf_models_architecture ON gguf_models(architecture);
CREATE INDEX IF NOT EXISTS idx_gguf_models_is_vlm ON gguf_models(is_vlm);
CREATE INDEX IF NOT EXISTS idx_gguf_models_huggingface_id ON gguf_models(huggingface_id);
CREATE INDEX IF NOT EXISTS idx_gguf_models_is_active ON gguf_models(is_active);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_gguf_models_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_gguf_models_updated_at
    BEFORE UPDATE ON gguf_models
    FOR EACH ROW
    EXECUTE FUNCTION update_gguf_models_updated_at();

