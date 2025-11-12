-- Create model downloads tracking table
CREATE TABLE IF NOT EXISTS model_downloads (
    id TEXT PRIMARY KEY,
    model_id TEXT NOT NULL, -- References gguf_models.id
    huggingface_url TEXT NOT NULL,
    status TEXT NOT NULL,   -- pending, downloading, paused, completed, failed, cancelled
    progress INTEGER DEFAULT 0, -- 0-100
    total_size BIGINT,
    downloaded_size BIGINT DEFAULT 0,
    download_speed BIGINT,  -- bytes per second
    eta_seconds INTEGER,    -- estimated time remaining
    temp_file_path TEXT,    -- for resume support
    sha256_hash TEXT,       -- expected SHA256 checksum
    sha256_verified BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    started_at TIMESTAMP,
    paused_at TIMESTAMP,
    resumed_at TIMESTAMP,
    completed_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES gguf_models(id) ON DELETE CASCADE
);

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_model_downloads_model_id ON model_downloads(model_id);
CREATE INDEX IF NOT EXISTS idx_model_downloads_status ON model_downloads(status);
CREATE INDEX IF NOT EXISTS idx_model_downloads_created_at ON model_downloads(created_at);

-- Trigger to auto-update updated_at
CREATE OR REPLACE FUNCTION update_model_downloads_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_model_downloads_updated_at
    BEFORE UPDATE ON model_downloads
    FOR EACH ROW
    EXECUTE FUNCTION update_model_downloads_updated_at();

