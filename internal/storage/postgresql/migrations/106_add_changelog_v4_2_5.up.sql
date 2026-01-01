INSERT INTO changelogs (version, release_date, content) VALUES
('4.2.5', '2025-01-01', '## [4.2.5] - 2025-01-01

### Fixed

- **Test Generation MR**: Fixed "A file with this name already exists" error
  - Now checks if test file exists in target branch before commit
  - Uses update action for existing files, create for new files

### Technical

- internal/api/handlers/gitlab_testgen.go: Check file existence via GitLab API before commit')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

