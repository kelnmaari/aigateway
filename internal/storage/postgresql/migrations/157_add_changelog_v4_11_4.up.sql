INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.4', '2026-02-17', '## [4.11.4] - 2026-02-17

### Added
- **Providers**: DeepSeek provider — OpenAI-compatible backend for DeepSeek API (deepseek-chat, deepseek-reasoner, etc.)

### Technical
- DeepSeekProvider implementation with HealthCheck, ListModels, GetModelInfo
- Proxy routing uses proxyOpenAICompatible (same as OpenAI/vLLM)
- Frontend: DeepSeek in provider type dropdown with default URL https://api.deepseek.com')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
