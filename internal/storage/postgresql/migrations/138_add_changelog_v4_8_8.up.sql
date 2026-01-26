-- Add changelog entry for v4.8.8
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.8',
    '2026-01-26',
    '## [4.8.8] - 2026-01-26

### Fixed

- **Analysis Timeouts**: Increased timeouts for all heavy LLM-based analysis components (Quality, Test Generation, Autodoc, Dead Code, Deep Scan) to 15-30 minutes. This prevents "context canceled" errors during long-running tasks.
- **Inference Proxy**: Increased generation timeout to 15 minutes in the inference proxy.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
