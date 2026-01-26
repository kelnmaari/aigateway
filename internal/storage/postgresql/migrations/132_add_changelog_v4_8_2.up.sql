-- Add changelog entry for v4.8.2
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.2',
    '2026-01-26',
    '## [4.8.2] - 2026-01-26

### Fixed

- **Dependencies Scanner**: Fixed chunk concatenation when reading dependency files from vector store. Added newline separators between chunks to prevent JSON parsing errors caused by chunks being joined without whitespace.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
