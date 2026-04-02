INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.3', '2026-04-02', '## [5.4.3] - 2026-04-02

### Fixed
- Worker registration: fixed invalid UUID generation in CreateWorkerNode (was using UnixNano instead of gen_random_uuid())')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
