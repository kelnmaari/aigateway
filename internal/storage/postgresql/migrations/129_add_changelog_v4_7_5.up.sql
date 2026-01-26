-- Migration 129: Add changelog for v4.7.5
INSERT INTO changelogs (version, release_date, change_type, component, description)
VALUES 
('4.7.5', '2026-01-26', 'added', 'GitLab Integration', 'Added ability to select specific branches for repository indexing in the project settings.'),
('4.7.5', '2026-01-26', 'added', 'GitLab API', 'New endpoint to fetch project branches via integrated GitLab credentials.');
