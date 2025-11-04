
-- ========================================
-- Add Changelog v1.5.2 (Migration v7)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.2', '2025-10-10', '## [1.5.2] - 2025-10-10

### Fixed
- **SSE Authentication**: Real-time logs работает с JWT
  - Токен через query параметр (EventSource не поддерживает headers)
  - SSEAuthMiddleware для SSE endpoints

### Changed
- Backend: sse_auth.go middleware
- Frontend: токен в URL stream

### Technical
- SSE + JWT через query параметр')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	