-- Add changelog entry for v4.7.5
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.7.5',
    '2026-01-26',
    '## [4.7.5] - 2026-01-26

### Added

- **GitLab Project Configuration**: Added ability to select specific branches for repository indexing in the project settings. This allows users to create search indexes for feature branches in addition to the default branch.
- **GitLab User API**: New endpoint to fetch project branches via integrated GitLab credentials.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
