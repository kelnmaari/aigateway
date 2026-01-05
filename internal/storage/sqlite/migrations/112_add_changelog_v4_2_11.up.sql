INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('4.2.11', '2026-01-05', '## [4.2.11] - 2026-01-05

### Fixed

- **Index Panic on New Projects**: Fixed nil pointer dereference when indexing newly added GitLab projects
  - `GetStatus()` returns nil for projects without prior indexing status
  - Added nil check before accessing status fields

### Technical

- `internal/gitlab/indexer/indexer.go`: Added nil check in `IndexBranch()` for `GetStatus()` return value');

