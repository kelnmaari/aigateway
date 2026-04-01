INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.0', '2026-04-02', '## [5.4.0] - 2026-04-02

### Added
- **Agent Mode**: Remote inference workers for distributed GPU/CPU model serving
  - New binary: aigateway-agent (minimal HTTP server + inference subsystem)
  - Agent API: model lifecycle, health checks, GPU/CPU metrics, inference proxy with streaming
  - Worker node management with health check loop, per-node locks, model sync
  - Inference Router integration: RemoteModelResolver, transparent proxying
  - /v1/models merge: agent models visible in Chat UI (owned_by: provider@worker)
  - Admin API: 13 endpoints for worker CRUD, model management, config generation
  - Database: worker_nodes table (migration 198)
- **Docker image transfer**: push local custom images to agents (docker save/load)
- **Private Docker registry auth**: docker login with credentials on agent startup
- **RPM packaging for agent**: spec + systemd service + install script + CI pipeline
- **Config generator**: generate agent.yaml + API key from Admin UI')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
