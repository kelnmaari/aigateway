-- Scheduled scans table for dependency scanning (v4.1.0)
CREATE TABLE IF NOT EXISTS gitlab_scheduled_scans (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES gitlab_projects(id) ON DELETE CASCADE,
    integration_id TEXT NOT NULL REFERENCES gitlab_integrations(id) ON DELETE CASCADE,
    scan_type TEXT NOT NULL DEFAULT 'dependencies',
    frequency TEXT NOT NULL DEFAULT 'daily',
    cron_expr TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Notification settings
    notify_email TEXT,
    create_issue BOOLEAN NOT NULL DEFAULT FALSE,
    only_breaking BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Execution tracking
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_run_status TEXT,
    last_run_error TEXT,
    last_run_duration_ms BIGINT DEFAULT 0,
    
    -- Statistics
    total_runs INTEGER NOT NULL DEFAULT 0,
    successful_runs INTEGER NOT NULL DEFAULT 0,
    failed_runs INTEGER NOT NULL DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Scan history table for tracking individual scan executions
CREATE TABLE IF NOT EXISTS gitlab_scan_history (
    id TEXT PRIMARY KEY,
    schedule_id TEXT NOT NULL REFERENCES gitlab_scheduled_scans(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL,
    scan_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms BIGINT DEFAULT 0,
    error TEXT,
    
    -- Results summary
    dependencies_checked INTEGER DEFAULT 0,
    outdated_dependencies INTEGER DEFAULT 0,
    vulnerabilities_found INTEGER DEFAULT 0,
    breaking_changes INTEGER DEFAULT 0,
    issue_created BOOLEAN DEFAULT FALSE,
    issue_url TEXT,
    
    -- Full results (JSON)
    results_json JSONB
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_scheduled_scans_project ON gitlab_scheduled_scans(project_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_scans_integration ON gitlab_scheduled_scans(integration_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_scans_enabled ON gitlab_scheduled_scans(enabled);
CREATE INDEX IF NOT EXISTS idx_scheduled_scans_next_run ON gitlab_scheduled_scans(next_run_at);

CREATE INDEX IF NOT EXISTS idx_scan_history_schedule ON gitlab_scan_history(schedule_id);
CREATE INDEX IF NOT EXISTS idx_scan_history_project ON gitlab_scan_history(project_id);
CREATE INDEX IF NOT EXISTS idx_scan_history_started ON gitlab_scan_history(started_at);
CREATE INDEX IF NOT EXISTS idx_scan_history_status ON gitlab_scan_history(status);

