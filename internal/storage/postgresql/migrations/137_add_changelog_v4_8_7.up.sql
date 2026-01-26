-- Add changelog entry for v4.8.7
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.8.7',
    '2026-01-26',
    '## [4.8.7] - 2026-01-26

### Fixed

- **Deep Scan**: Improved robustness against LLM hallucinations and garbage output. Added garbage detection in output parsing, refined system prompts for better JSON compliance, and set temperature to 0.0 for maximum determinism.
- **Deep Scan**: Increased HTTP client timeout to 3 minutes to handle complex batch analysis.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
