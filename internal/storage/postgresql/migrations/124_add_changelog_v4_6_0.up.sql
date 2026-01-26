-- Add changelog entry for v4.6.0
INSERT INTO changelogs (version, release_date, content)
VALUES (
    '4.6.0',
    '2026-01-26',
    '## [4.6.0] - 2026-01-26

### Added

- **Tenant Access for Projects**: Regular users can now share GitLab projects within their organization/tenant, enabling group updates and joint reviews.
- **Enhanced Indexing Visibility**: Project list now shows total chunk count and last indexing timestamp for better RAG control.
- **Webhook Connection Monitoring**: Clear indication of GitLab webhook registration status directly in the user interface.

### Fixed

- **Analysis Rendering Issues**: Restored broken data display for Code Quality, Dead Code, Auto-Documentation, and Test Generation scanners in Svelte UI.
- **Improved Evidence Visibility**: Brightened code snippets in Secrets Scan results for better readability.
- **Multi-Chunk Manifest Support**: Dependency scanner now correctly handles large package.json and other manifest files split across multiple vector store chunks.'
) ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
