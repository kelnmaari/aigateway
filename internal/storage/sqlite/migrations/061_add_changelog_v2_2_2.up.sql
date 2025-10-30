
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.2.2', '2025-10-28', '## [2.2.2] - 2025-10-28

### Added

- **Enhanced User Management**: Админ-панель теперь отображает расширенную информацию о пользователях
  - **Auth Provider Badge**: Отображение способа регистрации пользователя (Local, OIDC, LDAP)
    - Local: фиолетовый градиент
    - OIDC: розовый градиент
    - LDAP: синий градиент
  - **RBAC Roles Display**: Показывает роли пользователя из системы RBAC
    - Отображение до 2 ролей с badge''ами
    - Счетчик "+N" для остальных ролей
    - Иконка 🏢 для tenant-specific ролей
    - "No roles" для пользователей без ролей
  - **Tenants Display**: Отображение всех тенантов в которых участвует пользователь
    - Отображение до 2 тенантов с badge''ами
    - Иконка 👑 для Owner, 👤 для Member
    - Золотой градиент для Owner, оранжевый для Member
    - Счетчик "+N" для остальных тенантов
  - **XSS Protection**: Все пользовательские данные экранируются через escapeHtml() для безопасности

### Technical

- **Backend (Go)**:
  - Новые модели: UserWithDetails, RoleInfo, TenantInfo в internal/models/user.go
  - Реализован метод GetUsersWithDetails(ctx, filters) для SQLite с JOIN''ами к RBAC и tenants таблицам
  - Добавлен метод ListAllTenants(ctx) для получения списка всех tenants
  - Stub реализации для PostgreSQL в internal/storage/postgresql/stubs.go
  - Delegation методы в sqliteTx и postgresqlTx для поддержки транзакций
  - Обновлен AdminUserHandler.ListUsers() для возврата enriched данных

- **Frontend (HTML/JS/CSS)**:
  - Обновлена таблица Users в web/admin.html с 3 новыми колонками: AUTH PROVIDER, RBAC ROLES, TENANTS
  - Новые helper функции в web/js/admin.js:
    - renderAuthProviderBadge(authProvider) - рендеринг badge для способа аутентификации
    - renderRolesBadges(roles) - рендеринг RBAC ролей с ограничением отображения
    - renderTenantsBadges(tenants) - рендеринг тенантов с owner/member индикацией
    - escapeHtml(text) - защита от XSS атак
  - Добавлены CSS стили для новых badges в web/css/style.css:
    - .badge-auth-local, .badge-auth-oidc, .badge-auth-ldap (градиентные фоны)
    - .badge-role, .badge-tenant, .badge-tenant-owner (роли и тенанты)
    - .badge-count (счетчик для скрытых элементов)
  - Обновлен метод renderUsers() для использования новых helper функций

### Fixed

- **JavaScript Syntax Error**: Исправлена критическая ошибка в web/js/admin.js на строке 342
  - Незакрытая arrow function в методе renderUsers()
  - Добавлено корректное закрытие: }).join(''''); вместо ).join('''');

### Changed

- API endpoint /api/admin/users теперь возвращает UserWithDetails вместо базовой модели User
- Таблица Users расширена с 6 до 9 колонок для отображения новой информации');
	