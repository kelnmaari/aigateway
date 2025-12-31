INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.1', '2024-12-31', '## [4.2.1] - 2024-12-31

### Added

- **Create Issue Button for All Scanners**: Added Create Issue button to all scan modals
  - Secrets Scan: Creates issue with findings summary
  - Deep Scan (LLM): Creates issue with LLM-detected secrets
  - Code Quality: Creates issue with recommendations
  - Dead Code: Creates issue with dead symbols summary
  - All buttons generate detailed markdown descriptions

### Fixed

- **Create Issue Button in Dependencies Check**: Button was silently failing
  - Added selectedProject = project in all scan handlers

- **Changelog Analysis LLM Integration**: Fixed "connection refused" for Analyze button
  - Dependencies handler now uses correct internal API key
  - Scheduled scans also use correct API key

### Technical

- Added CreateSecretsIssue, CreateQualityIssue handlers
- Updated CreateDeadCodeIssue to actually create issues via GitLab API
- Added routes for /secrets/create-issue, /quality/create-issue
- Added Create Issue buttons to all scan modals');
