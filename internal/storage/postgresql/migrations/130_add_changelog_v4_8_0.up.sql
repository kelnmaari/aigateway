-- Add changelog entry for v4.8.0
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.0',
    '2026-01-26',
    '## [4.8.0] - 2026-01-26

### Fixed

- **Deep Scan JSON Parsing**: Improved LLM response parsing with automatic JSON extraction from text responses. Added stricter prompts and better error handling for models that don''t follow JSON-only instructions.
- **NPM Dependencies Parser**: Fixed parsing of Vaadin-specific package.json files with custom sections and invalid version references (like `$@package`). Parser now correctly handles `vaadin` section and filters out npm override references.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
