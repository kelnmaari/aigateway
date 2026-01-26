-- Add changelog entry for v4.8.6
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.6',
    '2026-01-26',
    '## [4.8.6] - 2026-01-26

### Fixed

- **Changelog Fetcher**: Configured custom HTTP transport with aggressive timeouts (5s connection, 5s TLS handshake, 5s response headers) to prevent slow DNS or connection issues. Total timeout reduced to 15 seconds.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
