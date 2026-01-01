INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.4', '2024-12-31', '## [4.2.4] - 2024-12-31

### Fixed

- **MR Creation with Default Branch**: Fixed "invalid reference name ''main''" error
  - CreateProject now saves default_branch from GitLab API
  - UpdateProject supports updating default_branch
  - Added fallback to "main" if default branch is not set
  - Added "Default Branch" field in project edit modal (WebUI)

### Added

- **Changelog in Release Notes**: GitLab release page now shows changelog content
  - Automatically extracts version-specific changelog from CHANGELOG.md
  - Combined with install/upgrade instructions

### Technical

- internal/gitlab/storage/postgres.go: Fixed CreateProject/UpdateProject for default_branch
- internal/api/handlers/gitlab_testgen.go, gitlab_autodoc.go: Added fallback for empty DefaultBranch
- web-svelte: Added Default Branch field to project edit modal
- .gitlab-ci.yml: Release job now extracts changelog from CHANGELOG.md');

