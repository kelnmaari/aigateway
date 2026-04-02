INSERT INTO changelogs (version, release_date, content) VALUES
('5.4.6', '2026-04-02', '## [5.4.6] - 2026-04-02

### Added
- Target Node selector in Admin Models Load form — choose local or remote worker
- target_node field in LoadRequest — backend routes to agent via LoadModelOnNode
- GET /api/system/inference/workers — worker list endpoint for UI dropdown
- InferenceHandler agent manager integration for remote model loading')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
