-- Add changelog entry for v4.8.3
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.3',
    '2026-01-26',
    '## [4.8.3] - 2026-01-26

### Fixed

- **Changelog Analysis**: Added detailed debug logging to help diagnose analysis completion issues. Logs now show analysis start, completion status, and response metadata.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
