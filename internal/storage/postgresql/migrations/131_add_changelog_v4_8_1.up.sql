-- Add changelog entry for v4.8.1
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.1',
    '2026-01-26',
    '## [4.8.1] - 2026-01-26

### Fixed

- **NPM Parser**: Fixed JSON parsing error "invalid character after top-level value" by adding content cleaning to remove BOM markers, trim whitespace, and extract JSON boundaries. Parser now handles files with encoding issues and trailing garbage.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
