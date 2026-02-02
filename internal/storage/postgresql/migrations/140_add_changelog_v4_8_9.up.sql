INSERT INTO changelogs (version, release_date, content) VALUES
('4.8.9', '2026-02-02', '## [4.8.9] - 2026-02-02

### Added

- **Background Jobs System**: Introduced a comprehensive background job system for GitLab operations. Users can now trigger scans and other operations that continue running even if they close the browser.
  - New `user_jobs` table for tracking job status, progress, and results
  - Job types: secrets scan, deep secrets scan, SAST scan, dependency scan, quality scan, dead code scan, autodocs scan, test generation, project indexing, MR creation
  - Real-time progress tracking with percentage and status messages
  - Job cancellation support

- **User Jobs API**: New endpoints for managing background jobs:
  - `GET /api/gitlab/jobs` - List user''s jobs with filtering
  - `GET /api/gitlab/jobs/active` - Get currently running jobs
  - `GET /api/gitlab/jobs/:id` - Get job details
  - `POST /api/gitlab/jobs/:id/cancel` - Cancel a running job
  - `GET /api/gitlab/jobs/types` - List available job types
  - `GET /api/gitlab/jobs/statuses` - List job statuses

- **Deep Scan Improvements**:
  - Enhanced hallucination detection for numeric garbage patterns
  - Added SSE streaming mode with real-time progress updates
  - Test file filtering to skip test directories
  - Documentation file filtering to reduce false positives
  - Improved few-shot examples in prompts for better accuracy
  - Post-validation filtering for placeholders and environment variables
  - Reduced batch size from 5 to 3 for better quality

### Technical

- New packages: internal/gitlab/jobs (service, executors)
- New storage: internal/gitlab/storage/postgres_user_jobs.go
- New handlers: internal/api/handlers/gitlab_jobs.go
- Migration 139: user_jobs table with indexes for efficient querying')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
