-- GitLab Integration Tables
-- Migration: 085_create_gitlab_tables

-- GitLab Integrations (connections to GitLab instances)
CREATE TABLE IF NOT EXISTS gitlab_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    base_url VARCHAR(500) NOT NULL,
    access_token TEXT NOT NULL,  -- Encrypted
    webhook_secret VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    last_sync_at TIMESTAMPTZ,
    last_error TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gitlab_integrations_status ON gitlab_integrations(status);
CREATE INDEX IF NOT EXISTS idx_gitlab_integrations_name ON gitlab_integrations(name);

-- GitLab Projects (repositories configured for review)
CREATE TABLE IF NOT EXISTS gitlab_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES gitlab_integrations(id) ON DELETE CASCADE,
    gitlab_project_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    path_with_namespace VARCHAR(500) NOT NULL,
    webhook_id BIGINT,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    auto_review BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Model Configuration
    analysis_model_id VARCHAR(255) NOT NULL,
    embedding_model_id VARCHAR(255) NOT NULL,
    
    -- Review Configuration
    review_prompt TEXT,
    settings JSONB DEFAULT '{}',
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(integration_id, gitlab_project_id)
);

CREATE INDEX IF NOT EXISTS idx_gitlab_projects_integration ON gitlab_projects(integration_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_status ON gitlab_projects(status);
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_analysis_model ON gitlab_projects(analysis_model_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_projects_embedding_model ON gitlab_projects(embedding_model_id);

-- GitLab MR Reviews (analysis results)
CREATE TABLE IF NOT EXISTS gitlab_mr_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES gitlab_projects(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES gitlab_integrations(id) ON DELETE CASCADE,
    
    -- MR Information
    mr_iid INTEGER NOT NULL,
    mr_title TEXT NOT NULL,
    mr_author VARCHAR(255) NOT NULL,
    mr_author_id BIGINT NOT NULL,
    source_branch VARCHAR(255) NOT NULL,
    target_branch VARCHAR(255) NOT NULL,
    mr_url TEXT NOT NULL,
    
    -- Review Status
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'analyzing', 'completed', 'failed', 'cancelled', 'skipped')),
    priority VARCHAR(50) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    
    -- Analysis Results
    files_analyzed INTEGER NOT NULL DEFAULT 0,
    lines_changed INTEGER NOT NULL DEFAULT 0,
    issues_found INTEGER NOT NULL DEFAULT 0,
    review_result JSONB,
    
    -- GitLab Note
    note_id BIGINT,
    discussion_id VARCHAR(255),
    
    -- Performance Metrics
    processing_time_ms BIGINT NOT NULL DEFAULT 0,
    tokens_used INTEGER NOT NULL DEFAULT 0,
    model_used VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Error handling
    error TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_project ON gitlab_mr_reviews(project_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_integration ON gitlab_mr_reviews(integration_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_status ON gitlab_mr_reviews(status);
CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_mr_iid ON gitlab_mr_reviews(project_id, mr_iid);
CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_created ON gitlab_mr_reviews(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gitlab_reviews_priority_status ON gitlab_mr_reviews(priority, status);

-- GitLab Analysis Jobs (queue)
CREATE TABLE IF NOT EXISTS gitlab_analysis_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES gitlab_mr_reviews(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES gitlab_projects(id) ON DELETE CASCADE,
    integration_id UUID NOT NULL REFERENCES gitlab_integrations(id) ON DELETE CASCADE,
    
    -- Job Info
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled', 'retrying')),
    priority VARCHAR(50) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    
    -- MR Info (denormalized)
    mr_iid INTEGER NOT NULL,
    mr_title TEXT NOT NULL,
    
    -- Worker Info
    worker_id VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Retry Info
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_status ON gitlab_analysis_jobs(status);
CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_priority ON gitlab_analysis_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_review ON gitlab_analysis_jobs(review_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_worker ON gitlab_analysis_jobs(worker_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_next_retry ON gitlab_analysis_jobs(next_retry_at) WHERE next_retry_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_gitlab_jobs_pending ON gitlab_analysis_jobs(priority DESC, created_at ASC) WHERE status = 'pending';

-- GitLab Webhook Events (for deduplication)
CREATE TABLE IF NOT EXISTS gitlab_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    integration_id UUID NOT NULL REFERENCES gitlab_integrations(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL,  -- GitLab project ID (not our ID)
    mr_iid INTEGER NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    action VARCHAR(50) NOT NULL,
    object_id BIGINT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    deduplicated BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_gitlab_events_dedup ON gitlab_webhook_events(integration_id, project_id, mr_iid, object_id);
CREATE INDEX IF NOT EXISTS idx_gitlab_events_received ON gitlab_webhook_events(received_at DESC);

-- Function to update updated_at
CREATE OR REPLACE FUNCTION update_gitlab_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers
DROP TRIGGER IF EXISTS gitlab_integrations_updated_at ON gitlab_integrations;
CREATE TRIGGER gitlab_integrations_updated_at
    BEFORE UPDATE ON gitlab_integrations
    FOR EACH ROW
    EXECUTE FUNCTION update_gitlab_updated_at();

DROP TRIGGER IF EXISTS gitlab_projects_updated_at ON gitlab_projects;
CREATE TRIGGER gitlab_projects_updated_at
    BEFORE UPDATE ON gitlab_projects
    FOR EACH ROW
    EXECUTE FUNCTION update_gitlab_updated_at();

