INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.8', '2026-03-25', '## [4.13.8] - 2026-03-25

### Added
- **vLLM Extra Args**: New `vllm_extra_args` parameter for passing additional CLI flags to vLLM container (e.g. `--enable-auto-tool-choice --tool-call-parser hermes`)

### Fixed
- **Web Search 400 Error with vLLM**: Fixed tool calling error when using web search — vLLM containers now support extra args for `--enable-auto-tool-choice` and `--tool-call-parser`

### Technical
- Added VLLMExtraArgs to all inference model structs and SvelteKit UI forms
- Same pattern as LlamaExtraArgs: space-separated CLI flags appended to vLLM command')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
