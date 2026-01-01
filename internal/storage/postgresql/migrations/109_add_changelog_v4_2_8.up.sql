INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.8', '2025-01-01', '## [4.2.8] - 2025-01-01

### Fixed

- **Dependency Scanner Deduplication**: Files with multiple chunks no longer scanned multiple times
- **Changelog Analysis Null Error**: Fixed "Cannot read properties of null" when arrays are null

### Technical

- `internal/gitlab/dependencies/scanner.go`: Deduplication by file_path in `findAllDependencyFiles()`
- `web-svelte`: Added null checks (`?.`) for changelog result arrays')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;

