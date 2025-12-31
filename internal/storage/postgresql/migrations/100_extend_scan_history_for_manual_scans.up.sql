-- Extend gitlab_scan_history to support manual scans (v4.1.2+)
-- Make schedule_id nullable and add integration_id, model_id, tokens_used

-- Make schedule_id nullable
ALTER TABLE gitlab_scan_history ALTER COLUMN schedule_id DROP NOT NULL;

-- Add new columns with correct types matching PostgreSQL schema
-- integration_id: UUID to match gitlab_integrations.id (but nullable, no FK for manual scans)
ALTER TABLE gitlab_scan_history ADD COLUMN IF NOT EXISTS integration_id UUID;
ALTER TABLE gitlab_scan_history ADD COLUMN IF NOT EXISTS model_id TEXT;
ALTER TABLE gitlab_scan_history ADD COLUMN IF NOT EXISTS findings_count INTEGER DEFAULT 0;
ALTER TABLE gitlab_scan_history ADD COLUMN IF NOT EXISTS files_affected INTEGER DEFAULT 0;
ALTER TABLE gitlab_scan_history ADD COLUMN IF NOT EXISTS tokens_used BIGINT DEFAULT 0;

-- Create new indexes
CREATE INDEX IF NOT EXISTS idx_scan_history_scan_type ON gitlab_scan_history(scan_type);
CREATE INDEX IF NOT EXISTS idx_scan_history_integration ON gitlab_scan_history(integration_id);

