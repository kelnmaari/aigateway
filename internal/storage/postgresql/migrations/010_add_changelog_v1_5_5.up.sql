
-- ========================================
-- Add Changelog v1.5.5 (Migration v10)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.5', '2025-10-10', '## [1.5.5] - 2025-10-10

### Fixed
- **Logs Stream File Selection**: Real-time логи
  - SSE stream параметр ?file=filename
  - Auto-select текущего лог-файла

### Changed
- Backend: StreamLogs query param file
- Frontend: передача файла в stream

### Technical
- Problem: hardcoded "proxy.log"
- Solution: query param + auto-select')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	