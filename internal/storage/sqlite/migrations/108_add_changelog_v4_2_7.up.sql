INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.7', '2025-01-01', '## [4.2.7] - 2025-01-01

### Added

- **Multi-Ecosystem Dependency Scanning**: Full monorepo support
  - Scans ALL dependency files (go.mod, package.json, requirements.txt)
  - UI tabs for switching between ecosystems (Go | Node.js | Python)
  - Aggregated total summary across all ecosystems
  - Per-ecosystem breakdown with individual file paths and durations
  - Combined issue creation with dependencies from all ecosystems

### Technical

- internal/gitlab/dependencies/types.go: Added MultiEcosystemScanResult type
- internal/gitlab/dependencies/scanner.go: New ScanProjectAllEcosystems() method
- internal/api/handlers/gitlab_dependencies.go: Uses multi-ecosystem scanner
- web-svelte UI: Ecosystem tabs for multi-ecosystem display');

