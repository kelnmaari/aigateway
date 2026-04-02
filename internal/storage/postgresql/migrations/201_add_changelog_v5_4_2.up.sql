INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.2', '2026-04-02', '## [5.4.2] - 2026-04-02

### Fixed
- Generate Config now auto-registers worker node in DB (status=pending)
- Added required address field to Generate Config (main server → agent URL)
- CI: fixed YAML anchor in build:agent job
- Added workers.enabled to dev.yaml

### Changed
- GenerateConfigRequest: new required field address
- GenerateConfigResponse: new field worker_id
- Workers Admin UI: address field with helper text, auto-refresh after generate')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
