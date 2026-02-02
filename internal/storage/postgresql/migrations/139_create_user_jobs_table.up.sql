-- Create user_jobs table for background task tracking
CREATE TABLE IF NOT EXISTS user_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID,
    project_id UUID NOT NULL,
    integration_id UUID NOT NULL,
    job_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',

    -- Job configuration (JSON)
    config JSONB DEFAULT '{}',

    -- Progress tracking
    progress INTEGER DEFAULT 0,
    progress_msg TEXT,

    -- Timing
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Results
    result_id UUID,
    result_type VARCHAR(50),
    result_url TEXT,
    error TEXT,

    -- Foreign keys
    CONSTRAINT fk_user_jobs_project FOREIGN KEY (project_id) REFERENCES gitlab_projects(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_jobs_integration FOREIGN KEY (integration_id) REFERENCES gitlab_integrations(id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX idx_user_jobs_user_id ON user_jobs(user_id);
CREATE INDEX idx_user_jobs_project_id ON user_jobs(project_id);
CREATE INDEX idx_user_jobs_integration_id ON user_jobs(integration_id);
CREATE INDEX idx_user_jobs_status ON user_jobs(status);
CREATE INDEX idx_user_jobs_job_type ON user_jobs(job_type);
CREATE INDEX idx_user_jobs_created_at ON user_jobs(created_at DESC);

-- Composite index for user's active jobs
CREATE INDEX idx_user_jobs_user_status ON user_jobs(user_id, status);

-- Composite index for project's jobs history
CREATE INDEX idx_user_jobs_project_created ON user_jobs(project_id, created_at DESC);

COMMENT ON TABLE user_jobs IS 'Background jobs initiated by users (scans, indexing, generation tasks)';
COMMENT ON COLUMN user_jobs.job_type IS 'Type: secrets_scan, deep_secrets_scan, sast_scan, dependency_scan, quality_scan, deadcode_scan, autodocs_scan, testgen_scan, project_index, generate_docs, generate_tests, create_mr';
COMMENT ON COLUMN user_jobs.status IS 'Status: pending, running, completed, failed, cancelled';
COMMENT ON COLUMN user_jobs.progress IS 'Progress percentage 0-100';
COMMENT ON COLUMN user_jobs.result_id IS 'Reference to scan_history or other result table';
COMMENT ON COLUMN user_jobs.result_type IS 'Type of result: scan_result, merge_request, index';
