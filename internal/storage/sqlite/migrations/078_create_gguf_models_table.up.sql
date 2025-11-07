-- MIGRATE-02: Create GGUF models tracking table (v3.0.5+)
CREATE TABLE IF NOT EXISTS gguf_models (
    id TEXT PRIMARY KEY,                    -- Unique model ID (sha256 of file path or HF ID)
    name TEXT NOT NULL,                     -- Human-readable model name
    file_path TEXT NOT NULL UNIQUE,         -- Absolute path to GGUF file
    file_size INTEGER NOT NULL,             -- File size in bytes
    architecture TEXT,                      -- Model architecture: llama, qwen, gemma, mistral, etc.
    quantization TEXT,                      -- Quantization level: Q4_K_M, Q8_0, F16, etc.
    parameters_count INTEGER,               -- Number of parameters (e.g., 7000000000 for 7B)
    context_size INTEGER,                   -- Context window size
    is_vlm BOOLEAN DEFAULT FALSE,           -- Is this a Vision Language Model?
    mmproj_path TEXT,                       -- Path to mmproj file (for VLMs)
    huggingface_id TEXT,                    -- Original Hugging Face model ID (e.g., "Qwen/Qwen2.5-VL-3B-GGUF")
    huggingface_file TEXT,                  -- Original HF file name (e.g., "qwen2.5-vl-3b-q4_k_m.gguf")
    downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  -- When model was downloaded
    is_active BOOLEAN DEFAULT TRUE,         -- Is model available (not deleted)
    last_loaded_at TIMESTAMP,               -- When model was last loaded into memory
    load_count INTEGER DEFAULT 0,           -- How many times model was loaded
    metadata TEXT,                          -- JSON metadata from GGUF file
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_gguf_models_name ON gguf_models(name);
CREATE INDEX IF NOT EXISTS idx_gguf_models_architecture ON gguf_models(architecture);
CREATE INDEX IF NOT EXISTS idx_gguf_models_is_vlm ON gguf_models(is_vlm);
CREATE INDEX IF NOT EXISTS idx_gguf_models_is_active ON gguf_models(is_active);
CREATE INDEX IF NOT EXISTS idx_gguf_models_huggingface_id ON gguf_models(huggingface_id);

