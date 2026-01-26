-- Add changelog entry for v4.7.3
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.7.3',
    '2026-01-26',
    '## [4.7.3] - 2026-01-26

### Added

- **Tenant Management**: Restored "Add Member" and "Change Role" functionality in Svelte-based Organizations UI.

### Fixed

- **API**: Fixed incorrect user search endpoint in Svelte-based UI.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
