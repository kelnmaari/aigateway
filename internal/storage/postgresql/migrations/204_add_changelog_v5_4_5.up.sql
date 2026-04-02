INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.5', '2026-04-02', '## [5.4.5] - 2026-04-02

### Fixed
- Chat UI: fixed think tag parsing for multiline markdown content inside thinking blocks (replaced single regex with two-step index-based extraction)')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
