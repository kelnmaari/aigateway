INSERT INTO changelogs (version, release_date, content) VALUES
('4.11.3', '2026-02-17', '## [4.11.3] - 2026-02-17

### Fixed
- **Model Registry**: Added missing `POST /registry/providers/:id/health` endpoint — individual provider health check now works from UI
- **Model Registry**: Added missing `POST /registry/providers/:id/discover` endpoint — per-provider model discovery now works from UI

### Technical
- `HealthCheckProvider` handler: checks single provider, returns status/message/response_time_ms, updates DB health status
- `DiscoverModelsFromProvider` in ProviderManager: discovers models from one provider, returns new + existing models
- `DiscoverModelsProvider` handler: returns message/models_found/models matching frontend DiscoverModelsResponse type')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
