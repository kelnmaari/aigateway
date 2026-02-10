INSERT INTO changelogs (version, release_date, content) VALUES
('4.10.1', '2026-02-10', '## [4.10.1] - 2026-02-10

### Fixed
- **llama.cpp Jinja Setting**: Fixed `--jinja` checkbox not persisting when editing saved model configuration
- `openEditSavedModal()` was not reading `llama_jinja` from saved model into the edit form
- Added missing `llama_jinja` to TypeScript type annotation of `editSavedForm`')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
