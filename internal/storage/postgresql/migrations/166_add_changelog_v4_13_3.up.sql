INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.3', '2026-03-03', '## [4.13.3] - 2026-03-03

### Fixed

- **TEI Rerank Adapter**: Fixed 404 error when using `/v1/rerank` with TEI inference provider — TEI uses `/rerank` endpoint (not `/v1/rerank`) with a different request/response format
- **Dify Integration**: Rerank model validation now works correctly with TEI-based models (e.g., `bge-reranker-v2-m3`)

### Added

- **TEI Request/Response Translation**: Automatic format conversion between OpenAI-compatible rerank API and TEI native format
- **Provider-aware Path Mapping**: `mapProviderPath()` translates OpenAI paths to provider-specific paths for inference containers
- **Debug Logging**: `unifiedPassthrough` now logs incoming requests with model name and routing decisions

### Technical

- `handleTEIRerank()` — full format adapter: OpenAI rerank ↔ TEI rerank with `top_n` and `return_documents` support
- `mapProviderPath()` — extensible path translation for provider-specific endpoints
- Enhanced logging in `unifiedPassthrough` for debugging routing issues')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
