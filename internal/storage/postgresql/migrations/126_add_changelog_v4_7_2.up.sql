-- Add changelog entry for v4.7.2
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.7.2',
    '2026-01-26',
    '## [4.7.2] - 2026-01-26

### Fixed

- **Critical: GitLab Project Listing**: Fixed a 500 error when listing GitLab projects caused by an unsupported `NULL` to string conversion for the `tenant_id` column.
- **Database Robustness**: Standardized `sql.NullString` handling across PostgreSQL storage layer for optional owner and tenant fields.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
