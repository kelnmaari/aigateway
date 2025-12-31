INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.0', '2024-12-31', '## [4.2.0] - 2024-12-31

### Added

- **Scan History UI**: Human-readable scan results display
  - Summary cards with severity breakdown (Critical/High/Medium/Low)
  - Category breakdown visualization
  - Findings list with file paths and line numbers
  - Collapsible raw JSON for debugging
  - Support for all scan types (Secrets, Quality, DeadCode, etc.)

### Fixed

- **AI Code Review Score**: Score no longer shows 0/100 when no issues found
  - Returns 100 when analysis finds no issues
  - Smart fallback scoring based on issue count
  - Model name now displayed in review comments

- **Index Status after Server Restart**: Status properly persisted and restored
  - GetStatus now falls back to database when cache is empty
  - Interrupted indexing (server restart during process) marked as "Failed"
  - Proper status synchronization between Redis/memory/DB

- **Data Race in Orchestrator**: Fixed concurrent access to ModelInstance fields
  - Added updateInstance() method for thread-safe field modifications
  - All status/handle/error updates now protected by mutex
  - Fixed race between StartModel and ListModels

- **Data Race in WorkerPool Tests**: Fixed atomic operations in tests

- **TypeScript Errors in WebUI**: Fixed 28 TypeScript errors
  - Added query parameter support to apiRequest
  - Fixed type definitions and event target casting

### Changed

- **CI/CD Pipeline Optimization**: Reduced test execution time from 10+ min to ~1-2 min
  - Split into test:fast (all commits) and test:race (MR only)
  - Race detector only runs on merge requests to main

### Technical

- internal/gitlab/indexer/indexer.go: GetStatus returns nil when no cache, handler checks DB
- internal/inference/orchestrator.go: Added updateInstance() for thread-safe updates
- internal/gitlab/processor/processor.go: Score defaults to 100 when no issues
- .gitlab-ci.yml: Optimized test pipeline with fast/race split')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

