INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.2', '2026-03-26', '## [5.1.2] - 2026-03-26

### Fixed
- **Saved Model Parameters Not Returned by API**: SavedModelResponse was missing all v5.1.0/v5.1.1 fields — params saved to disk but not returned in API, UI showed zeros on reload
- **SavedModel Missing SGLang/TGI Fields**: Added SGLangDataParallel, SGLangContextLen, SGLangChunkedPrefill, TGIMaxConcurrentReqs, TGIMaxInputLen, TGIMaxTotalTokens
- **SaveFromSpec/ToSpec Incomplete Mapping**: All fields now properly persist and restore')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
