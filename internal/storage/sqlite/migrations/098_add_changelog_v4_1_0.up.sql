INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.1.0', '2024-12-30', '## [4.1.0] - 2024-12-30

### Added
- **Auto-Doc: Bulk Apply + MR Creation**: Apply documentation to multiple files and create GitLab MR
  - POST `/api/admin/gitlab/projects/:id/bulk-apply-docs`
  - POST `/api/admin/gitlab/projects/:id/create-docs-mr`
  - Auto-generates commit messages and MR descriptions

- **Test Gen: Download as File**: Download generated tests as single file or ZIP archive
  - POST `/api/admin/gitlab/projects/:id/download-tests`
  - Supports single file or ZIP format
  - Auto-generates file headers for each language

- **Architecture Diagram Generator**: Automatic architecture visualization
  - Module dependency graph (Mermaid)
  - Call graph visualization
  - Package structure diagrams
  - Data flow diagrams with LLM enhancement
  - POST `/api/admin/gitlab/projects/:id/scan-architecture`
  - POST `/api/admin/gitlab/projects/:id/generate-diagram`
  - GET `/api/admin/gitlab/projects/:id/architecture`

- **SAST Security Scanner**: Static Application Security Testing
  - SQL injection patterns detection
  - XSS vulnerability patterns
  - Path traversal detection
  - Command injection patterns
  - Hardcoded IPs/URLs detection
  - Insecure cryptography (MD5, SHA1, DES)
  - Insecure random detection
  - Open redirect patterns
  - SSRF patterns
  - XXE vulnerability detection
  - Insecure deserialization patterns
  - Hardcoded credentials detection
  - CWE and OWASP classification
  - POST `/api/admin/gitlab/projects/:id/sast-scan`

- **Analytics Dashboard**: Complete analytics for projects and teams
  - Code review statistics (avg time, issues found)
  - Dependency health metrics (outdated %)
  - Security score tracking
  - Team productivity metrics
  - Model usage comparison
  - Export to JSON/CSV
  - GET `/api/admin/gitlab/analytics/dashboard`
  - GET `/api/admin/gitlab/analytics/models`
  - GET `/api/admin/gitlab/analytics/security`
  - GET `/api/admin/gitlab/analytics/dependencies`
  - GET `/api/admin/gitlab/projects/:id/analytics`
  - GET `/api/admin/gitlab/analytics/export`

### Changed
- Updated ROADMAP.md with completed features
- All major analysis features from roadmap now implemented

### Technical
- `internal/gitlab/architecture/` - Architecture diagram generation
- `internal/gitlab/scanner/sast.go` - SAST vulnerability patterns
- `internal/gitlab/analytics/service.go` - Extended analytics metrics
- `internal/api/handlers/gitlab_architecture.go` - Architecture API handlers
- `internal/api/handlers/gitlab_analytics.go` - Analytics API handlers
- Frontend types for all new features in `web-svelte/src/lib/api/gitlab.ts`');

