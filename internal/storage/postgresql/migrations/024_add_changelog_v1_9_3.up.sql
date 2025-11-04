
INSERT INTO changelogs (version, release_date, content) VALUES
('1.9.3', '2025-10-14', '## [1.9.3] - 2025-10-14

### Added
- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
- **Admin Panel Reorganization**: Улучшенная структура навигации

### Technical
- MoniGo integration
- Admin panel restructuring')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	