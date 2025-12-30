-- Rollback: Remove scheduled scans tables
DROP INDEX IF EXISTS idx_scan_history_status;
DROP INDEX IF EXISTS idx_scan_history_started;
DROP INDEX IF EXISTS idx_scan_history_project;
DROP INDEX IF EXISTS idx_scan_history_schedule;

DROP INDEX IF EXISTS idx_scheduled_scans_next_run;
DROP INDEX IF EXISTS idx_scheduled_scans_enabled;
DROP INDEX IF EXISTS idx_scheduled_scans_integration;
DROP INDEX IF EXISTS idx_scheduled_scans_project;

DROP TABLE IF EXISTS gitlab_scan_history;
DROP TABLE IF EXISTS gitlab_scheduled_scans;

