INSERT INTO changelogs (version, release_date, content) VALUES
('5.2.3', '2026-03-30', '## [5.2.3] - 2026-03-30

### Fixed
- **Tool calls broken through AIGateway proxy**: ChatToolsHandler was dropping tools, tool_choice, and all non-basic parameters before forwarding to vLLM. Raw request body is now preserved; only the model field is rewritten.
- **Missing Transfer-Encoding: chunked header** in ChatToolsHandler streaming path.

### Technical
- chat_tools_handler.go — forwardRawRequest + rewriteModelInJSON via map[string]json.RawMessage')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
