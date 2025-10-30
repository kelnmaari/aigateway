
-- ========================================
-- Add Changelog v1.5.8 (Migration v13)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.8', '2025-10-10', '## [1.5.8] - 2025-10-10

### Added
- **Tenant Membership Check**: Проверка прав доступа
  - Middleware для проверки членства
  - RBAC для tenant ресурсов
  - TODO удален из GetTenantUsage

- **Enhanced Member Management**: Управление участниками
  - Search endpoint для поиска пользователей
  - UI modal с live search
  - Change Role и Remove кнопки
  - Debounced search (500ms)

### Security
- Access control для tenant ресурсов
- Audit logging (403)
- Directory traversal protection

### Technical
- Middleware chain
- Context-based role storage
- Lazy loading user info');
	