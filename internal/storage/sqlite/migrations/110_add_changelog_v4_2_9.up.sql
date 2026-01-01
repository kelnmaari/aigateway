INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.9', '2025-01-01', '## [4.2.9] - 2025-01-01

### Fixed

- **Dependency Files Not Indexed**: `go.mod`, `go.sum`, `requirements.txt` and other manifest files now indexed
- **Truncated Dependency Content**: Scanner now uses longest chunk content instead of first found

### Technical

- `internal/gitlab/indexer/indexer.go`: Added `specialFiles` list for dependency manifests
- `internal/gitlab/dependencies/scanner.go`: Deduplication keeps longest content per file_path');

