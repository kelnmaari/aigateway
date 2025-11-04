
-- ========================================
-- Add Changelogs for v1.5.11 - v1.5.16 (Migration v17)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.5.16', '2025-10-11', '## [1.5.16] - 2025-10-11

### Changed
- **All WebUI Pages**: Система уведомлений внедрена во все 8 страниц
  - dashboard, chat, profile, tenants, usage, api-keys, mcp, about
  - ~10 alert() → toast notifications
  - ~8 confirm() → modal confirmations
  - Единообразный UX на всех страницах

### Technical
- notifications.css и notifications.js подключены во все HTML
- Консистентный порядок загрузки скриптов
- toast.* и modal.* API используется везде'),

('1.5.15', '2025-10-11', '## [1.5.15] - 2025-10-11

### Added
- **Toast Notification System**: Профессиональная система всплывающих уведомлений
  - Уведомления справа снизу: зеленые (успех, 10с), красные (ошибка, 60с)
  - Желтые (предупреждения) и синие (информация)
  - Возможность закрыть вручную, плавные анимации
  - Поддержка темной темы и mobile

- **Modal Confirmation System**: Модальные окна для подтверждения действий
  - Заменяют стандартные confirm dialogs
  - confirm(), danger(), warning() с Promise-based API
  - Красивый дизайн с иконками

- **Global API**: window.toast и window.modal для всех страниц
  - toast.success/error/warning/info(message)
  - await modal.confirm/danger/warning(message, title)

### Changed
- **Admin Panel**: Полностью переведен на новую систему уведомлений
  - Все alert() → toast notifications
  - Все confirm() → modal confirmations
  - Улучшен UX для операций backup, users, API keys, MCP

### Technical
- Class-based architecture, HTML escape для XSS
- Auto-stacking для multiple toasts
- CSS transitions 300ms, backdrop blur
- Mobile-responsive, dark theme support'),

('1.5.14', '2025-10-11', '## [1.5.14] - 2025-10-11

### Added
- **Backup & Restore System**: Полнофункциональная система резервного копирования и восстановления БД
  - POST /api/admin/backup - создание бэкапа с автоматическим timestamp
  - GET /api/admin/backups - список доступных бэкапов
  - GET /api/admin/backup/:filename - скачивание backup файла
  - POST /api/admin/restore/:filename - восстановление из бэкапа
  - DELETE /api/admin/backup/:filename - удаление старых бэкапов
  - Автоматическая ротация: хранение последних 10 бэкапов
  - Safety backup перед restore операцией
  - Бэкапы сохраняются в ./data/backups/ директорию

### Changed
- internal/api/handlers/backup.go - новый handler для backup операций
- internal/api/router/router.go - добавлены роуты для backup/restore в admin API
- Все backup операции требуют JWT аутентификацию + admin role

### Technical
- Использование io.Copy для эффективного копирования больших файлов
- Path security checks для защиты от path traversal
- Atomic restore with rollback на случай ошибки
- File sync после записи для data integrity
- Structured logging для всех backup операций'),

('1.5.13', '2025-10-11', '## [1.5.13] - 2025-10-11

### Changed
- **Error Type Checking**: Улучшена обработка типов ошибок в admin handlers
  - Реализована функция isNotFoundError() с использованием errors.As
  - Добавлены функции isAlreadyExistsError, isInvalidDataError, isPermissionError
  - Type-safe error checking вместо string comparison

### Technical
- Использование Go 1.20+ errors.As() для type assertions
- Proper unwrapping of wrapped errors
- Удален TODO комментарий из admin.go'),

('1.5.12', '2025-10-11', '## [1.5.12] - 2025-10-11

### Fixed
- **BUG-03: WebUI Chat Usage Tracking**: FOREIGN KEY constraint failed при использовании WebUI чата
  - Схема api_usage.api_key_id теперь nullable
  - Foreign key с ON DELETE SET NULL
  - JWT authenticated requests теперь используют NULL вместо jwt_auth/unknown
  - Миграция v16: конвертация существующих данных

### Changed
- models.APIUsage.APIKeyID изменен с string на *string
- middleware.UsageTracking обновлен для NULL значений
- internal/storage/sqlite/sqlite.go: новая миграция v16

### Technical
- Правильная обработка nullable fields в Go (*string)
- Database migration с data conversion
- Foreign key constraints с ON DELETE SET NULL'),

('1.5.11', '2025-10-11', '## [1.5.11] - 2025-10-11

### Fixed
- **BUG-02: Tenant API Keys Creation**: 404 ошибка при создании API ключей для организаций
  - Реализованы недостающие endpoints для tenant API keys
  - POST /api/tenants/:id/api-keys - создание ключа организации
  - GET /api/tenants/:id/api-keys - список ключей организации
  - DELETE /api/tenants/:id/api-keys/:key_id - удаление ключа

### Added
- internal/api/handlers/tenant.go: три новых метода
  - ListTenantAPIKeys с проверкой membership
  - CreateTenantAPIKey с owner/admin role check
  - DeleteTenantAPIKey с key ownership validation

### Changed
- internal/api/router/router.go: добавлены новые роуты
- Улучшена валидация прав доступа (owner/admin)
- Маскирование sensitive данных в ListTenantAPIKeys

### Technical
- Robust access control с role-based checks
- Proper key ownership validation
- Secure key generation с GenerateAPIKeyWithID
- Default rate limits для tenant keys')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
