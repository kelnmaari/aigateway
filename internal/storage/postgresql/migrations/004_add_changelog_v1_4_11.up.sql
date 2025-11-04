
-- ========================================
-- Fix Changelog Table Structure (Migration v4)
-- ========================================

-- Удаляем старую таблицу если она была с неправильной структурой
DROP TABLE IF EXISTS changelogs;

-- Создаем таблицу с правильной структурой
CREATE TABLE changelogs (
	version TEXT PRIMARY KEY,
	release_date DATE NOT NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_changelogs_release_date ON changelogs(release_date DESC);

-- Добавляем только версию 1.4.11
INSERT INTO changelogs (version, release_date, content) VALUES
('1.4.11', '2025-10-10', '## [1.4.11] - 2025-10-10

### Added
- **Система "О Системе"**: Новая страница
  - Отображение версии, git commit, build date
  - Accordion UI для changelog всех версий
  - Пункт "О Системе" в меню профиля

### Changed
- **Backend**: Migration, changelog handler, API endpoints
- **Frontend**: about.html, navbar, accordion UI

### Technical
- Database: таблица changelogs
- API: /api/system/info, /api/system/changelogs')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	