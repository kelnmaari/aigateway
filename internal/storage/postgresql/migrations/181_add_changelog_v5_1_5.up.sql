INSERT INTO changelogs (version, release_date, content) VALUES
('5.1.5', '2026-03-30', '## [5.1.5] - 2026-03-30

### Fixed

- **Container Crash Not Detected During Health Check**: When a container crashed (OOM, SGLang memory balance error), the health check kept polling the dead endpoint until timeout, leaving the model stuck in "Starting" status. Added container liveness check — fails fast with clear error on container exit.
- **Model Reload Deadlock After Evict**: Loading a model after Evict caused the UI to hang indefinitely. Added per-alias cancellation context to abort in-flight health checks when Evict is called.')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
