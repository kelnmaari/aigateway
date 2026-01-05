INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.10', '2025-01-01', '## [4.2.10] - 2025-01-01

### Fixed

- **Chunked Manifest Files**: Dependency scanner now concatenates ALL chunks for manifest files (go.mod, package.json)
  - Previously only one chunk was used, missing `require` blocks in go.mod
  - Now collects all chunks and sorts by `chunk_index` before concatenation

### Technical

- `internal/gitlab/dependencies/scanner.go`: `findAllDependencyFiles()` collects all chunks per file and concatenates them');

