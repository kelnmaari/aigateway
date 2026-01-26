-- Add changelog entry for v4.7.4
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.7.4',
    '2026-01-26',
    '## [4.7.4] - 2026-01-26

### Added

- **GitLab Integration**: Expanded bulk project configuration in Discovery modal. Now you can set Embedding Model, patterns, limits, and custom review prompt for multiple projects at once.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
