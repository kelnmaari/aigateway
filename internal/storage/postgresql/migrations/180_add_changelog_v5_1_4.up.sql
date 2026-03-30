INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.4', '2026-03-30', '## [5.1.4] - 2026-03-30

### Fixed

- **Model Reload Deadlock After Evict**: Loading a model after Evict caused the UI to hang indefinitely. Added per-alias cancellation context to abort in-flight health checks when Evict is called.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
