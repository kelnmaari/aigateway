INSERT INTO changelogs (version, release_date, content) VALUES
('4.4.0', '2026-01-22', '## [4.4.0] - 2026-01-22

### Added

- **GitLab Project Management (User-Level)**: Regular users can now fully manage their integrated projects
  - Added "Edit Project" modal with granular control over analysis and embedding models
  - New settings for auto-review, inclusion/exclusion patterns, and scanner parameters
  - Support for advanced settings: `MaxReviewTokens`, `PerFileReview`, `SkipBots`, and `ReviewLanguage`
  - Redesigned project configuration UI for better usability

### Fixed

- **GitLab Access Control**: Resolved 403 Forbidden errors when regular users attempted to index their projects
- **Backend Stability**: Fixed compilation error caused by duplicate `AnalyzeProject` function
- **API Parity**: Ensured all project settings are correctly synchronized between frontend and backend

### Technical

- **Svelte 5 Migration**: Migrated GitLab UI components to new Svelte event syntax (`onclick`, `onsubmit`, `onkeydown`)
- **API Extension**: Added `updateMyProject` to user API service
- **Route Registration**: Added `PUT /api/v1/gitlab/projects/:id` for user-level project updates')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
