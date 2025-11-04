-- Enable required PostgreSQL extensions (if not already enabled)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ========================================
-- Files Table (FILE-STORAGE-01: v1.10.0+)
-- ========================================
CREATE TABLE IF NOT EXISTS files (
	id TEXT PRIMARY KEY DEFAULT (encode(gen_random_bytes(16), 'hex')),
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal files
	
	-- File information
	filename TEXT NOT NULL,
	original_filename TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	size_bytes INTEGER NOT NULL,
	checksum_sha256 TEXT, -- For deduplication
	
	-- Storage
	storage_backend TEXT NOT NULL, -- 'local', 's3'
	storage_path TEXT NOT NULL, -- Path within backend
	storage_bucket TEXT, -- S3 bucket name
	
	-- Extracted content (for simple chat integration)
	extracted_text TEXT, -- Full text for simple use cases
	extraction_status TEXT DEFAULT 'pending', -- 'pending', 'completed', 'failed'
	extraction_error TEXT,
	
	-- Metadata (JSON)
	metadata JSONB, -- page_count, author, keywords, etc.
	
	-- Document info
	page_count INTEGER,
	word_count INTEGER,
	language TEXT,
	
	-- Access control
	is_public INTEGER DEFAULT 0, -- Boolean
	shared_with JSONB, --  array of user IDs
	
	-- Usage tracking
	download_count INTEGER DEFAULT 0,
	last_accessed_at TIMESTAMP,
	
	-- Timestamps
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
	deleted_at TIMESTAMP, -- Soft delete
	
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_tenant_id ON files(tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_mime_type ON files(mime_type);
CREATE INDEX IF NOT EXISTS idx_files_extraction_status ON files(extraction_status);
CREATE INDEX IF NOT EXISTS idx_files_checksum ON files(checksum_sha256);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_storage_backend ON files(storage_backend);

-- ========================================
-- File Access Logs Table (optional, для audit)
-- ========================================
CREATE TABLE IF NOT EXISTS file_access_logs (
	id BIGSERIAL PRIMARY KEY,
	file_id TEXT NOT NULL,
	user_id TEXT,
	action TEXT NOT NULL, -- 'upload', 'download', 'delete', 'view'
	ip_address TEXT,
	user_agent TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_file_logs_file_id ON file_access_logs(file_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_user_id ON file_access_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_created_at ON file_access_logs(created_at DESC);
	