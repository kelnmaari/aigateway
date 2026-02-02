INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.1', '2026-02-02', '## [4.9.1] - 2026-02-02

### Fixed

- **Jobs API**: Fixed UUID parsing error in GitLabJobsHandler - user_id was incorrectly extracted with `user_` prefix causing "invalid input syntax for type uuid" errors')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
