INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.0', '2026-02-02', '## [4.9.0] - 2026-02-02

### Added

- **Jobs Monitoring Tab**: Added "Jobs" tab to GitLab integration page for quick access to background jobs filtered by integration
- **Integration Filter**: Jobs page now supports filtering by integration_id via URL parameter

### Changed

- Moved job-related navigation from hidden to visible tab in integration UI')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
