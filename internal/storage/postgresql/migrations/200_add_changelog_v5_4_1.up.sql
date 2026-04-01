INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.1', '2026-04-02', '## [5.4.1] - 2026-04-02

### Added
- Workers Admin UI: full management page in admin panel (add/remove workers, GPU metrics, load/stop models, generate agent config)

### Fixed
- CI: build:agent YAML anchor fix (inline compute_version)
- Security: JSON injection fix in PullImageWithAuth, DockerLogin integration, input validation improvements
- WorkersConfig defaults, EnsureByAlias nodeAddr validation, GetModel remote fallback')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
