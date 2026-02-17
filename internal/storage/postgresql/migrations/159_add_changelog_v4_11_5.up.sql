INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.5', '2026-02-17', '## [4.11.5] - 2026-02-17

### Fixed
- **DeepSeek**: Added DB migration to include deepseek in valid_provider_type CHECK constraint — creating DeepSeek provider no longer returns 500')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
