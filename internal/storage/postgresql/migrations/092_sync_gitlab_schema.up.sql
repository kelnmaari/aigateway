-- Synchronize GitLab tables schema with models
-- Migration: 092_sync_gitlab_schema

-- ============================================================================
-- gitlab_integrations: Add missing columns
-- ============================================================================
ALTER TABLE gitlab_integrations ADD COLUMN IF NOT EXISTS owner_id TEXT;
ALTER TABLE gitlab_integrations ADD COLUMN IF NOT EXISTS tenant_id TEXT;

-- ============================================================================
-- gitlab_analysis_jobs: Add missing columns
-- ============================================================================
ALTER TABLE gitlab_analysis_jobs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add trigger for updated_at
DROP TRIGGER IF EXISTS gitlab_jobs_updated_at ON gitlab_analysis_jobs;
CREATE TRIGGER gitlab_jobs_updated_at
    BEFORE UPDATE ON gitlab_analysis_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_gitlab_updated_at();

-- ============================================================================
-- gitlab_mr_reviews: Add missing columns
-- ============================================================================
ALTER TABLE gitlab_mr_reviews ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 3;
ALTER TABLE gitlab_mr_reviews ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT NOW();

-- Add trigger for updated_at
DROP TRIGGER IF EXISTS gitlab_reviews_updated_at ON gitlab_mr_reviews;
CREATE TRIGGER gitlab_reviews_updated_at
    BEFORE UPDATE ON gitlab_mr_reviews
    FOR EACH ROW
    EXECUTE FUNCTION update_gitlab_updated_at();

-- ============================================================================
-- gitlab_webhook_events: Ensure payload field is properly typed
-- ============================================================================
-- Already added in migration 090, but ensure JSONB type
DO $$
BEGIN
    -- Only change type if it's TEXT
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'gitlab_webhook_events' 
        AND column_name = 'payload' 
        AND data_type = 'text'
    ) THEN
        ALTER TABLE gitlab_webhook_events ALTER COLUMN payload TYPE JSONB USING payload::JSONB;
    END IF;
END $$;

