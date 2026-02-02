INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.3', '2026-02-02', '## [4.9.3] - 2026-02-02

### Fixed

- **Jobs API UUID Parsing**: Fixed critical bug where user_id was passed with `user_` prefix but PostgreSQL expected raw UUID. Now strips prefix in getUserID() for both GitLabJobsHandler and GitLabJobSubmitHandler.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
