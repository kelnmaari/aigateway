
-- ========================================
-- Add Changelog v1.5.6 (Migration v11)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.6', '2025-10-10', '## [1.5.6] - 2025-10-10

### Improved
- **Modelfile Display**: LICENSE скрывается
  - Только конфигурация модели
  - 20 строк с прокруткой

### Changed
- Frontend: stripLicenseFromModelfile()
- CSS: code-block-large класс

### Technical
- Frontend-only изменение
- Regex парсинг LICENSE')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	