INSERT INTO changelogs (version, release_date, content) VALUES
('4.5.1', '2026-01-23', '## [4.5.1] - 2026-01-23

### Fixed

- **GitLab Dependency UI**: Fixed missing content (labels and versions) in the detailed dependency view due to incorrect field mapping.
- **Dependency Summary**: Fixed aggregate counters for updates and vulnerabilities in the dashboard cards.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
