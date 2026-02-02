INSERT INTO changelogs (version, release_date, content) VALUES
('4.9.2', '2026-02-02', '## [4.9.2] - 2026-02-02

### Added

- **Quality Scan Executor**: Background job executor for code quality analysis
- **Dependency Scan Executor**: Background job executor for vulnerability checking across all ecosystems (Go, npm, pip, Maven, Gradle)
- **Dead Code Scan Executor**: Background job executor for detecting unused code

### Fixed

- **Jobs API UUID**: Fixed UUID parsing error - user_id was extracted with `user_` prefix but DB expects raw UUID (strips prefix in getUserID)
- **JobService nil check**: Added null check for jobService in SetupGitLabUserJobsRoutes to prevent panic
- **Rate limiting**: Added limit of 5 concurrent jobs per user to prevent abuse
- **SSE heartbeat**: Added 15-second heartbeat to prevent proxy timeout on idle SSE connections

### Technical

- New executors: QualityScanExecutor, DependencyScanExecutor, DeadCodeScanExecutor
- All 6 scan types now have working background executors
- Improved SSE streaming reliability')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
