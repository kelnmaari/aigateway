-- Add changelog entry for v4.8.4
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.4',
    '2026-01-26',
    '## [4.8.4] - 2026-01-26

### Fixed

- **Changelog Analysis**: Increased request timeout from 2 to 5 minutes to prevent "context deadline exceeded" errors when analyzing large changelogs or using slower LLM models.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
