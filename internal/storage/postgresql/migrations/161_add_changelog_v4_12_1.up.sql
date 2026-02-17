INSERT INTO changelogs (version, release_date, content) VALUES
('4.12.1', '2026-02-17', '## [4.12.1] - 2026-02-17

### Added
- **Unified Chat Routing**: /v1/chat/completions and /api/chat/completions now route to both inference (Docker) and external providers (Model Registry)
- Chat UI can now send messages to external provider models (DeepSeek, OpenAI, Anthropic, Gemini) directly

### Technical
- unifiedChatCompletions() handler: checks inference first, then Model Registry, delegates to appropriate handler
- Both inference and external proxy coexist on the same endpoint')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
