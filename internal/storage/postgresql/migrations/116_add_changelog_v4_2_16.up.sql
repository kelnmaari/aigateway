INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.16', '2026-01-22', '## [4.2.16] - 2026-01-22

### Fixed

- **GitLab Integration Visibility**: Fixed an issue where new integrations created by regular users were not visible in their personal list
  - Added `owner_id` to the integration creation process in the database
  - Ensured `owner_id` is properly retrieved when fetching integration details
  - This ensures that integrations appear correctly in the user-level dashboard while remaining accessible to administrators

### Technical

- `internal/gitlab/storage/postgres.go`: Added `owner_id` field to `CreateIntegration`, `GetIntegration`, and `List` methods')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
