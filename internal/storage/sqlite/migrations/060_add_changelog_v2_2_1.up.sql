
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.2.1', '2025-10-28', '## [2.2.1] - 2025-10-28

### Fixed

- **RBAC Management WebUI**: Исправлена критическая ошибка загрузки пользователей во вкладке "User Roles"
  - Несоответствие ID элемента между HTML (id="userSelector") и JavaScript ($("#userSelect"))
  - Теперь корректно отображается список всех пользователей системы (24 users)
  - Корректно загружается список всех tenants (26 tenants) при назначении ролей
  
- **RBAC Permissions Reference**: Исправлена загрузка permissions при переключении вкладок
  - Добавлен механизм callbacks для табов с явным вызовом handlePermissionsTab() и handleUserRolesTab()
  - Экспортированы функции в window scope для доступа из HTML
  - Теперь 35 системных permissions корректно отображаются с группировкой по ресурсам

### Added

- **Admin API Endpoint**: Новый endpoint /api/admin/tenants для получения списка всех tenants
  - Реализация ListAllTenants() в SQLite storage
  - Stub реализация для PostgreSQL
  - Интеграция в router с admin middleware и JWT auth
  - Delegation методы в transaction wrappers

### Technical

- Обновлена структура TenantHandler с методом ListAllTenants()
- Добавлена функция loadUsersAndTenants() в admin-rbac.js с параллельной загрузкой данных
- Улучшены empty state сообщения для permissions и users с actionable инструкциями
- Добавлены callbacks механизм для Bootstrap tabs в admin-rbac.html
- Исправлен ID HTML элемента userSelector → userSelect для соответствия с JavaScript селектором');
	