-- MIGRATE-02: Create model downloads tracking table (v3.0.5+)
CREATE TABLE IF NOT EXISTS model_downloads (
    id TEXT PRIMARY KEY,                    -- Unique download ID
    model_id TEXT,                          -- FK to gguf_models (NULL if not yet created)
    huggingface_id TEXT NOT NULL,           -- Hugging Face model ID
    file_name TEXT NOT NULL,                -- File being downloaded
    status TEXT NOT NULL CHECK(status IN ('pending', 'downloading', 'paused', 'completed', 'failed', 'cancelled')),
    progress INTEGER DEFAULT 0,             -- Download progress (0-100)
    total_size INTEGER,                     -- Total file size in bytes
    downloaded_size INTEGER DEFAULT 0,      -- Downloaded bytes so far
    download_speed INTEGER,                 -- Current download speed (bytes/sec)
    eta_seconds INTEGER,                    -- Estimated time to completion (seconds)
    error_message TEXT,                     -- Error message if failed
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,                 -- When download completed
    paused_at TIMESTAMP,                    -- When download was paused
    resumed_at TIMESTAMP,                   -- When download was resumed
    retry_count INTEGER DEFAULT 0,          -- Number of retry attempts
    sha256_hash TEXT,                       -- Expected SHA256 hash
    sha256_verified BOOLEAN DEFAULT FALSE,  -- Was hash verified?
    temp_file_path TEXT,                    -- Path to temporary download file
    final_file_path TEXT,                   -- Path to final GGUF file
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (model_id) REFERENCES gguf_models(id) ON DELETE SET NULL
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_model_downloads_status ON model_downloads(status);
CREATE INDEX IF NOT EXISTS idx_model_downloads_huggingface_id ON model_downloads(huggingface_id);
CREATE INDEX IF NOT EXISTS idx_model_downloads_model_id ON model_downloads(model_id);
CREATE INDEX IF NOT EXISTS idx_model_downloads_started_at ON model_downloads(started_at DESC);

