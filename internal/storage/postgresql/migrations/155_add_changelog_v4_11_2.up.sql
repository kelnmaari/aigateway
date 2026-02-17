INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.2', '2026-02-17', '## [4.11.2] - 2026-02-17

### Fixed
- **HuggingFace**: Use HTTP 502 Bad Gateway instead of 401 for expired upstream token — prevents false user session logout')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
