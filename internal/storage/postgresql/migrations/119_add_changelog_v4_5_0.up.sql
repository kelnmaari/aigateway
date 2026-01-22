INSERT INTO changelogs (version, release_date, content) VALUES
('4.5.0', '2026-01-22', '## [4.5.0] - 2026-01-22

### Added

- **GitLab Webhook Auto-Registration**: Users can now automatically setup project webhooks directly from the UI.
- **Enhanced Dependency Analysis UI**: Ported full-featured, multi-ecosystem dependency view from admin panel to regular users.
  - Ecosystem tabs (Go, Node.js, Python, etc.)
  - AI-powered changelog analysis for upgrade risks
- **Default Project Settings**: New projects and the edit form now default to optimal settings (Russian language, specific patterns, high token limits).

### Fixed

- **GitLab Indexing Authorization**: Fixed 403 Forbidden error preventing regular users from indexing their own repositories.
- **Detect Duplication Rendering**: Resolved issue where duplication analysis results were not displaying correctly.
- **Backend Stability**: Fixed unused imports and standardized context timeouts in GitLab handlers.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
