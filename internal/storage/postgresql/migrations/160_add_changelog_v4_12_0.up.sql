INSERT INTO changelogs (version, release_date, content) VALUES
('4.12.0', '2026-02-17', '## [4.12.0] - 2026-02-17

### Added
- **Chat UI**: /v1/models endpoint now returns models from both inference (Docker) and Model Registry (external providers)
- Models from external providers (OpenAI, DeepSeek, Anthropic, Gemini) now appear in Chat UI model dropdown

### Technical
- /v1/models aggregates inference models + active Model Registry models in a single response')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
