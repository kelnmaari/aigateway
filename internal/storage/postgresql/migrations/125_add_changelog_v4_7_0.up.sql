-- Add changelog entry for v4.7.0
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.7.0',
    '2026-01-26',
    '## [4.7.0] - 2026-01-26

### Added

- **GitLab Project Discovery**: Users can now search their entire GitLab instance for repositories directly from the UI.
- **Bulk Project Import**: Support for selecting multiple discovered repositories and importing them with a shared configuration in one click.
- **Enhanced Analysis Selection**: Discovery and Add Project modals now feature a dynamic model selector with provider information.

### Changed

- **Standardized GitLab User API**: Improved consistency between admin and user-level API methods for project discovery and management.
- **Improved API Client Robustness**: Fixed various formatting and parameter mapping issues in the GitLab TypeScript clients.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
