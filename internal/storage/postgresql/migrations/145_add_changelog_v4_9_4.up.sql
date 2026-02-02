INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.4', '2026-02-02', '## [4.9.4] - 2026-02-02

### Fixed

- **Jobs Tab UI**: Fixed Jobs tab on GitLab integration page - changed from external link to proper inline tab with embedded job list and real-time progress cards

### Changed

- Jobs are now displayed directly on the integration page instead of redirecting to a separate page
- Added pagination support for jobs list within the tab
- Added refresh button for jobs list')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
