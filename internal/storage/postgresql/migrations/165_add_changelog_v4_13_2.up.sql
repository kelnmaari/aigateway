INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.2', '2026-03-03', '## [4.13.2] - 2026-03-03

### Fixed
- **Svelte Frontend TEI/Rerank Filters**: Added TEI and Rerank provider filters to the SvelteKit HuggingFace model browser
- **Provider Hints**: Added provider hints for TEI and Rerank filters in search UI
- **Recommended Provider**: TEI/Rerank filters now correctly recommend TEI provider

### Technical
- HFProviderFilter type extended with tei and rerank in +page.svelte
- useHFModel() switch cases added for tei and rerank provider filters')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
