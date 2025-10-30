
-- ========================================
-- Add Changelog v1.5.3 (Migration v8)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.3', '2025-10-10', '## [1.5.3] - 2025-10-10

### Improved
- **Enhanced Log Parsing**: Полноценный парсер logrus
  - Извлечение всех полей (key=value)
  - Timestamp: HH:MM:SS.mmm формат
  - Context fields в UI

### Changed
- Backend: parseLogrusFields() метод
- Frontend: syntax highlighting для key=value
- CSS: VS Code-style colors

### Technical
- Парсинг: quoted и unquoted values
- Highlighting: #569cd6 / #ce9178');
	