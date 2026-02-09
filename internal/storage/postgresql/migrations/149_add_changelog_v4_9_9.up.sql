INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.9', '2026-02-09', '## [4.9.9] - 2026-02-09

### Added

- **llama.cpp**: Added `--jinja` parameter support for Jinja template processing in chat templates
- **UI**: Added Jinja checkbox in model configuration forms for llama.cpp provider')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
