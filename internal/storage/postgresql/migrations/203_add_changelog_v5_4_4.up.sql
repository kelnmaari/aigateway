INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.4', '2026-04-02', '## [5.4.4] - 2026-04-02

### Fixed
- WebUI: refreshed embedded Svelte build with Workers page JS chunks (fixed MIME type mismatch on admin tab navigation)')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
