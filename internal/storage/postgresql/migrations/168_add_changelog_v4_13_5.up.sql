INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.5', '2026-03-04', '## [4.13.5] - 2026-03-04

### Added
- **llama.cpp Cache Reuse**: New cache_reuse parameter for llama-server --cache-reuse flag (0=default, -1=disable for SWA models)
- **llama.cpp Extra Args**: New extra_args parameter for passing arbitrary CLI flags to llama-server

### Technical
- Added LlamaCacheReuse and LlamaExtraArgs to all inference model structs
- Updated BuildLlamaCPPRequest and SvelteKit UI')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
