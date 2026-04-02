INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.8', '2026-04-02', '## [5.4.8] - 2026-04-02

### Added
- Edit Saved Model: Target Node dropdown for reassigning models to different workers
- UpdateSavedRequest: target_node field for persisting worker assignment')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
