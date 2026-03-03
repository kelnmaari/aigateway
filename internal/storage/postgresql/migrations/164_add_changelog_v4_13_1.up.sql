INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.1', '2026-03-03', '## [4.13.1] - 2026-03-03

### Added
- **HuggingFace TEI Filter**: Provider filter for searching TEI-compatible models (embeddings & reranking)
- **HuggingFace Rerank Filter**: Provider filter for searching reranking models
- **Provider Dropdown**: HuggingFace model browser UI now includes Provider filter dropdown (All, vLLM/SGLang/TGI, llama.cpp, TEI, Embedding, Rerank)

### Technical
- GetModelsSearch() and GetPopularModels() extended with tei and rerank provider filter cases')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
