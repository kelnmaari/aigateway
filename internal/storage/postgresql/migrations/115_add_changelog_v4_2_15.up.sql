INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.15', '2026-01-22', '## [4.2.15] - 2026-01-22

### Added

- **User-Level GitLab Integration**: Regular users can now manage their own GitLab integrations
  - Create, edit, and delete personal GitLab integrations
  - Add and manage projects within integrations
  - View code review history for own projects
  - Full ownership isolation (users can only access their own data)
  - Edit integration settings including access token updates

### Technical

- `internal/api/handlers/gitlab_user.go`: User-level GitLab handlers with ownership checks
- `internal/api/router/router.go`: User routes registration in `setupGitLabRoutes()`
- `web-svelte/src/lib/api/gitlab-user.ts`: Frontend API client for user operations
- `web-svelte/src/routes/(protected)/gitlab/+page.svelte`: User GitLab management UI')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

