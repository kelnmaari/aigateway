-- Add changelog entry for v4.6.0
INSERT INTO changelog (version, title, description, changes, created_at)
VALUES (
    '4.6.0',
    'GitLab User UI Refinement & Tenant Access',
    'This release improves the GitLab integration experience for regular users and enables project-level sharing within organizations.',
    '{"added": ["Tenant Access for GitLab projects", "Webhook registration status monitoring", "Granular indexing stats (chunk count, timestamp)"], "fixed": ["Analysis data rendering in Svelte UI (Quality, DeadCode, etc.)", "Secrets Scan evidence brightness", "Improved large package.json support (multi-chunk collation)"]}',
    NOW()
) ON CONFLICT (version) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    changes = EXCLUDED.changes;
