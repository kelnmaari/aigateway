
-- ========================================
-- Add Changelog v1.5.1 (Migration v6)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.1', '2025-10-10', '## [1.5.1] - 2025-10-10

### Added
- **Enhanced Logs System**: Полноценная система просмотра логов в админке
  - Отдельная вкладка "Logs" в админ-панели
  - Просмотр текущего и архивных лог-файлов
  - Кликабельные фильтры по уровням (DEBUG, INFO, WARN, ERROR)
  - Real-time обновление логов через SSE (Server-Sent Events)
  - Скачивание лог-файлов
  - Dark theme для logs viewer (VS Code style)

### Changed
- **Backend**: LogsHandler с SSE stream, фильтрацией, пагинацией
- **Frontend**: Logs tab, real-time updates, level filters
- **UI**: Dark terminal theme, monospace font

### Technical
- SSE Stream через EventSource API
- Security: path validation
- Performance: tail 500 строк')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	