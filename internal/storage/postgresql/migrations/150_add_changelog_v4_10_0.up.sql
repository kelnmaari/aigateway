INSERT INTO changelogs (version, release_date, content) VALUES
('4.10.0', '2026-02-10', '## [4.10.0] - 2026-02-10

### Added
- **GitLab Webhooks Pause Toggle**: Lightweight switch to pause incoming webhook processing (MR reviews) without disabling the entire integration
- New `webhooks_paused` field in integration settings (JSONB, no schema migration needed)
- Webhook handler returns HTTP 200 when paused to prevent GitLab retries
- Admin UI toggle with yellow visual indicator when paused

### Technical
- Added `WebhooksPaused` bool to `GitLabIntegrationSettings` Go struct
- Added `webhooks_paused` to frontend `GitLabIntegrationSettings` TypeScript interface')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
