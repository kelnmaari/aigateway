
-- ========================================
-- Add Changelog v1.5.4 (Migration v9)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.4', '2025-10-10', '## [1.5.4] - 2025-10-10

### Fixed
- **SSE Logs Stream Auth**: Endpoint вне admin group
  - SSEAuthMiddleware читает token из query
  - RequireAdmin после аутентификации

### Changed
- Router: SSE endpoint отдельная регистрация
- Middleware chain: SSEAuth → RequireAdmin

### Technical
- Problem: JWTAuth блокировал SSE
- Solution: endpoint вне group')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	