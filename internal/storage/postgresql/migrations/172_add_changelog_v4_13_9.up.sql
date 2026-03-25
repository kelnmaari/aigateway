INSERT INTO changelogs (version, release_date, content) VALUES
('4.13.9', '2026-03-25', '## [4.13.9] - 2026-03-25

### Added
- **Model Restart Button**: Added Restart button for running models in the admin UI
- **POST /api/system/inference/restart**: New API endpoint to restart a model container

### Fixed
- **Model Stuck After Evict**: Fixed bug where model could not be restarted after stop+evict — stale SpecRegistry entries are now properly cleaned up

### Technical
- Added SpecRegistry.Unregister(), Router.GetModelInstance(), Router.ForgetModel()
- Service.Forget() now cleans up both orchestrator and spec registry')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
