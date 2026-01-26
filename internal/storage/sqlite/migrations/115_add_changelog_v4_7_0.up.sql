-- Add changelog entry for v4.7.0
INSERT INTO changelog (version, title, description, changes, created_at)
VALUES (
    '4.7.0',
    'GitLab Project Discovery & Bulk Import',
    'This release introduces a powerful way to discover and bulk-add projects from GitLab, alongside API client improvements for better stability.',
    '{"added": ["GitLab Project Discovery (scan your whole instance)", "Bulk Project Import with shared configuration", "Dynamic model selector in discovery modals"], "changed": ["Standardized GitLab User API methods", "Improved TypeScript API client robustness and type safety"]}',
    CURRENT_TIMESTAMP
) ON CONFLICT (version) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    changes = EXCLUDED.changes;
