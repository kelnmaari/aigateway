INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.0', '2026-03-03', '## [4.13.0] - 2026-03-03

### Added
- **Full OpenAI-compatible API**: All major OpenAI API endpoints through unified passthrough routing
- **Embeddings API**: POST /v1/embeddings
- **Rerank API**: POST /v1/rerank (Cohere/Jina/TEI-compatible)
- **Audio API**: POST /v1/audio/transcriptions, /v1/audio/translations, /v1/audio/speech
- **Images API**: POST /v1/images/generations, /v1/images/edits, /v1/images/variations
- **Moderations API**: POST /v1/moderations
- **Model Details**: GET /v1/models/:model
- **Dify Integration**: Full support for all OpenAI-API-compatible model types

### Technical
- Generic unifiedPassthrough(path) handler with JSON and multipart/form-data support
- ExtractModelFromRequest helper for model field extraction from any content type
- HandlePassthrough on ExternalProxyHandler and InferenceProxyHandler')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
