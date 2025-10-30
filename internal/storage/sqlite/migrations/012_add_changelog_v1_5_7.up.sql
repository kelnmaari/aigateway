
-- ========================================
-- Add Changelog v1.5.7 (Migration v12)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.7', '2025-10-10', '## [1.5.7] - 2025-10-10

### Added
- **Password Change**: Смена пароля
  - Endpoint /api/users/me/password
  - Валидация текущего пароля
  - Проверка силы нового пароля

### Changed
- Backend: UpdateUserPassword() метод
- SQLite/PostgreSQL реализация

### Security
- Bcrypt хеширование
- Audit logging
- Валидация силы пароля

### Technical
- Транзакционная поддержка
- Structured logging');
	