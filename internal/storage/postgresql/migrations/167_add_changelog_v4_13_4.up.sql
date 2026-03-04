INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.4', '2026-03-04', '## [4.13.4] - 2026-03-04

### Fixed
- **Chat Tools Handler**: Fixed `json: cannot unmarshal array into Go struct field ChatMessage.messages.content of type string` error when clients (e.g. OpenClaw) send `content` as array (multimodal format per OpenAI API spec) instead of plain string

### Technical
- Changed `Content` field type from `string` to `interface{}` in `ChatMessage` and `llmResponse` structs in `chat_tools_handler.go` to support both string and array content formats')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
