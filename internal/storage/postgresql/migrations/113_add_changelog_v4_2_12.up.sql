INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.12', '2026-01-05', '## [4.2.12] - 2026-01-05

### Added

- **Test Generation Review Warning**: MR descriptions now include a checklist for mandatory code review
  - Warning about placeholder import paths (`yourapp/...`)
  - Checklist: verify imports, test logic, dependencies, run tests
  - Clear "REVIEW REQUIRED" banner at the top

### Changed

- **Improved Test Generation Prompts**: AI now instructed not to use placeholder paths
  - Go tests: explicit instruction to avoid `yourapp/...` placeholders
  - Suggests using TODO comments for module-specific imports

### Technical

- `internal/api/handlers/gitlab_testgen.go`: Added review checklist to MR description
- `internal/gitlab/testgen/generator.go`: Updated Go test prompt with import path instructions')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

