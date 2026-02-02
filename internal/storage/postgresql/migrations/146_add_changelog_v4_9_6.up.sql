INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.6', '2026-02-02', '## [4.9.6] - 2026-02-02

### Added

- **Model Refresh/Update**: Added ability to refresh/re-download saved models in Model Registry
  - New 🔄 button in Saved Models list for HuggingFace models
  - Automatically re-downloads missing or corrupted files (like `config.json`)
  - New API endpoint `POST /api/system/inference/refresh-saved?alias=...`

### Technical

- New `PostRefreshSaved` handler in inference API
- Uses existing `DownloadRepository` which intelligently skips already-downloaded files')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
