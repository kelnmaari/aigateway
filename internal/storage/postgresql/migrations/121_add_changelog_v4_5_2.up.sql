INSERT INTO changelogs (version, release_date, content) VALUES
('4.5.2', '2026-01-23', '## [4.5.2] - 2026-01-23

### Fixed

- **GitLab Changelog Analysis**: Fixed 400 Bad Request error when triggering AI changelog analysis due to incorrect field mapping in the frontend.
- **Dependency Tracking**: Corrected property mapping for package names and versions in the analysis request payload.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
