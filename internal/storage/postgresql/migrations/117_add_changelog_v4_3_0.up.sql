INSERT INTO changelogs (version, release_date, content) VALUES
('4.3.0', '2026-01-22', '## [4.3.0] - 2026-01-22

### Added

- **GitLab User-Level Analysis & Scanners**: Regular users can now perform deep analysis on their owned projects
  - Added comprehensive Security Scans (Secrets, Deep Scan, SAST)
  - Added Code Quality Analysis and Duplication Detection
  - Added Dependency Check and Changelog Analysis
  - Added Dead Code, Auto-Documentation, and Test Generation
- **Enhanced GitLab UI**: New "Analyze" interactive modal in project details
  - Grouped scanners by category (Security, Quality, DevOps & Tooling)
  - Real-time progress indicators for analysis tasks
  - Interactive summary results with severity breakdown
  - JSON results viewer for detailed findings

### Fixed

- **GitLab Access Control**: Restored repository indexing for regular users with proper ownership verification (fixing 403 Forbidden errors)
- **Scanner Security**: Implemented strict ownership checks across all GitLab module handlers to prevent cross-user data access

### Technical

- **Route Refactoring**: Centralized GitLab route registration in `gitlab_routes.go` for better maintainability and code reuse
- **API Client**: Extended `gitlab-user.ts` frontend service with 15+ new scanner and analysis methods')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
