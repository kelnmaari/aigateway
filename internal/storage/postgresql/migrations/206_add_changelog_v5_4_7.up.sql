INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.7', '2026-04-02', '## [5.4.7] - 2026-04-02

### Fixed
- Models UI: Target Node dropdown 401 fix (use api.get with JWT instead of raw fetch)
- Models UI: Target Node dropdown always visible when workers enabled, shows empty state with link to Workers page')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
