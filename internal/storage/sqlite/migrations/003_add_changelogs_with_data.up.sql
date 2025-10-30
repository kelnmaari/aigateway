
-- ========================================
-- Changelogs Table (System Info: v1.4.11+)
-- ========================================
CREATE TABLE IF NOT EXISTS changelogs (
	version TEXT PRIMARY KEY,
	release_date DATE NOT NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_changelogs_release_date ON changelogs(release_date DESC);

-- ========================================
-- Initial Changelog Data from CHANGELOG.md
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
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
  - Проблема: api_key_id ссылался на несуществующие ключи (jwt_auth, unknown)
  - Решение: Сделать api_key_id nullable с ON DELETE SET NULL
  - Теперь статистика WebUI чата корректно учитывается

### Changed
- Database Schema: api_key_id TEXT nullable (было NOT NULL)
- FOREIGN KEY constraint: ON DELETE SET NULL (было CASCADE)
- Миграция v16 автоматически конвертирует jwt_auth/unknown в NULL

### Technical
- Migration v16: создает временную таблицу с правильной структурой
- Автоматическая конвертация существующих записей с fake keys
- Пересоздание индексов с учетом nullable значений'),

('1.5.11', '2025-10-11', '## [1.5.11] - 2025-10-11

### Fixed
- **BUG-02: Tenant API Keys Creation**: 404 при создании API ключей для организации
  - Добавлены endpoints для управления tenant API keys
  - GET /api/tenants/:id/api-keys - получение списка ключей организации
  - POST /api/tenants/:id/api-keys - создание ключа организации
  - DELETE /api/tenants/:id/api-keys/:key_id - удаление ключа организации

### Added
- **Tenant API Keys Management**: Полноценное управление API ключами организаций
  - Handler ListTenantAPIKeys для получения списка ключей
  - Handler CreateTenantAPIKey для создания ключей с правами owner/admin
  - Handler DeleteTenantAPIKey для удаления ключей с проверкой владения
  - Автоматическая проверка прав доступа (owner/admin only)
  - Валидация принадлежности ключа к tenant при удалении

### Security
- **Role-based Access Control**: Только owner и admin могут создавать/удалять tenant API keys
- **Tenant Ownership Verification**: Проверка принадлежности ключа к tenant перед удалением
- **Membership Check**: Проверка членства пользователя в tenant для всех операций

### Technical
- Default rate limits для tenant keys: 100 req/min, 5000 req/hour, 50000 req/day
- Key scope автоматически устанавливается в APIKeyScopeTenant
- TenantID связывается с API key через foreign key
- Plaintext key возвращается только один раз при создании'),

('1.4.10', '2025-10-10', '## [1.4.10] - 2025-10-10

### Added
- **"Remember Me" Функция при входе**:
  - Checkbox "Не выходить из системы 24 часа" на форме логина
  - При включении галочки access token живет **24 часа** вместо 15 минут
  - Refresh token продолжает работать 7 дней как и раньше
  - Логирование использования remember_me в JWT и auth service

### Changed
- **Backend**: LoginRequest, GenerateTokenPair, JWT логирование
- **Frontend**: Добавлен checkbox на форму входа

### Technical
- Без галочки: Access token 15 минут, Refresh token 7 дней
- С галочкой: Access token 24 часа, Refresh token 7 дней
- Фактическая длительность сессии: до 7 дней'),

('1.4.9', '2025-10-10', '## [1.4.9] - 2025-10-10

### Fixed
- **API Keys Status Display**:
  - Problem: Свежесозданные ключи отображались как "Inactive"
  - Solution: Frontend использует поле status из API response
  - Ключ активен если status === ''active'' И не истек срок'),

('1.4.8', '2025-10-10', '## [1.4.8] - 2025-10-10

### Added
- **Debug Logging для Usage Tracking**
- Логирование user_id, api_key_id, model, tokens, success

### Fixed
- **API Key ID для JWT аутентификации**: используется "jwt_auth" ID'),

('1.4.7', '2025-10-10', '## [1.4.7] - 2025-10-10

### Fixed
- **CRITICAL: Usage Statistics Performance**
  - Problem: Запросы 5+ секунд, context canceled
  - Solution: Оптимизированы SQL с SUM(CASE WHEN ...)
  - Performance: < 100ms для большинства запросов'),

('1.4.6', '2025-10-10', '## [1.4.6] - 2025-10-10

### Added
- **API Usage Tracking**: Полная интеграция middleware
- Автоматическая запись в api_usage таблицу
- Сбор метрик: user_id, model, tokens, duration'),

('1.4.5', '2025-10-10', '## [1.4.5] - 2025-10-10

### Added
- **MCP Servers Catalog**: Admin-managed catalog
- Database: таблица mcp_servers
- Frontend: web/mcp.html с публичной страницей'),

('1.4.4', '2025-10-10', '## [1.4.4] - 2025-10-10

### Added
- **Enhanced Models Information**: Accordion UI
- Lazy loading через /api/admin/models/:name/details
- Секции: Basic Info, Specifications, Template, Modelfile'),

('1.4.3', '2025-10-10', '## [1.4.3] - 2025-10-10

### Added
- **Build Version Information**: Реальная версия через ldflags
- Флаг -version для показа информации о билде
- VERSION файл как единственный источник версии'),

('1.4.2', '2025-10-10', '## [1.4.2] - 2025-10-10

### Security
- **TUI Admin Key Configuration**: Убраны hardcoded keys
- Admin token загружается из конфигурации
- Добавлен флаг --config для указания пути'),

('1.4.1', '2025-10-10', '## [1.4.1] - 2025-10-10

### Added
- **Model Copy Button**: Кнопка копирования модели
- Clipboard API с fallback
- Toast notifications для feedback'),

('1.4.0', '2025-10-08', '## [1.4.0] - 2025-10-08

WebUI Enhancements & Code Quality - Base Release

Включает: Model Copy, TUI Admin Config, Build Version, Enhanced Models, MCP Catalog'),

('1.3.0', '2025-10-06', '## [1.3.0] - 2025-10-06

User Experience & Multi-Tenancy

### Added
- Database Abstraction Layer (SQLite + PostgreSQL)
- User Authentication & Multi-Tenancy (JWT, RBAC)
- Interactive Chat Interface
- User Dashboard
- Cross-platform Build System'),

('1.2.0', '2025-10-01', '## [1.2.0] - 2025-10-01

Enhanced Monitoring & Management

### Added
- TUI Request Monitor
- WebUI Metrics Visualization
- Advanced Logs Features
- Enhanced API Key Management');
	