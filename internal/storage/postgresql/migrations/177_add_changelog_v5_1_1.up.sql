INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.1', '2026-03-26', '## [5.1.1] - 2026-03-26

### Added
- **Tool Calling Section**: Dedicated UI subsection for vLLM, SGLang, llama.cpp with dedicated controls
- **vLLM**: enable_auto_tool_choice checkbox, tool_call_parser dropdown (10 parsers), chat_template path
- **SGLang**: tool_call_parser dropdown (7 parsers)
- **llama.cpp**: chat_template dropdown (14 templates)
- **TGI**: Informational note about API-level tool calling support

### Technical
- New fields in ModelSpec, SavedModel, API request/response structs
- Updated provider builders to emit tool calling CLI arguments')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
