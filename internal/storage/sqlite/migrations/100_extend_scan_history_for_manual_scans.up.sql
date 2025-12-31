-- Extend gitlab_scan_history to support manual scans (v4.1.2+)
-- Make schedule_id nullable and add integration_id, model_id, tokens_used

-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table
-- First, create a new table with the correct schema
CREATE TABLE IF NOT EXISTS gitlab_scan_history_new (
    id TEXT PRIMARY KEY,
    schedule_id TEXT REFERENCES gitlab_scheduled_scans(id) ON DELETE SET NULL,
    project_id TEXT NOT NULL,
    integration_id TEXT,
    scan_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    model_id TEXT,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    duration_ms INTEGER DEFAULT 0,
    error TEXT,
    
    -- Results summary
    findings_count INTEGER DEFAULT 0,
    files_affected INTEGER DEFAULT 0,
    tokens_used INTEGER DEFAULT 0,
    
    -- Legacy columns for dependencies (kept for backward compatibility)
    dependencies_checked INTEGER DEFAULT 0,
    outdated_dependencies INTEGER DEFAULT 0,
    vulnerabilities_found INTEGER DEFAULT 0,
    breaking_changes INTEGER DEFAULT 0,
    issue_created INTEGER DEFAULT 0,
    issue_url TEXT,
    
    -- Full results (JSON)
    results_json TEXT
);

-- Copy existing data if table exists
INSERT OR IGNORE INTO gitlab_scan_history_new (
    id, schedule_id, project_id, scan_type, status, started_at, completed_at, 
    duration_ms, error, dependencies_checked, outdated_dependencies, 
    vulnerabilities_found, breaking_changes, issue_created, issue_url, results_json
)
SELECT 
    id, schedule_id, project_id, scan_type, status, started_at, completed_at,
    duration_ms, error, dependencies_checked, outdated_dependencies,
    vulnerabilities_found, breaking_changes, issue_created, issue_url, results_json
FROM gitlab_scan_history;

-- Drop old table and rename new one
DROP TABLE IF EXISTS gitlab_scan_history;
ALTER TABLE gitlab_scan_history_new RENAME TO gitlab_scan_history;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_scan_history_schedule ON gitlab_scan_history(schedule_id);
CREATE INDEX IF NOT EXISTS idx_scan_history_project ON gitlab_scan_history(project_id);
CREATE INDEX IF NOT EXISTS idx_scan_history_started ON gitlab_scan_history(started_at);
CREATE INDEX IF NOT EXISTS idx_scan_history_status ON gitlab_scan_history(status);
CREATE INDEX IF NOT EXISTS idx_scan_history_scan_type ON gitlab_scan_history(scan_type);
CREATE INDEX IF NOT EXISTS idx_scan_history_integration ON gitlab_scan_history(integration_id);

