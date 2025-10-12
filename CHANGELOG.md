# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.3] - 2025-10-12

### Added

- **Scheduled Reports**: Автоматическая генерация и отправка отчетов по email
  - Cron-based scheduler с поддержкой гибких расписаний
  - Report Generator для Usage, Performance и System Health отчетов
  - Email Mailer с SMTP отправкой (поддержка TLS/SSL)
  - HTML email templates с современным дизайном
  - Поддержка множественных recipients

- **Report Types**: Три типа отчетов
  - **Usage Report**: Total requests, tokens, top models, top users
  - **Performance Report**: Latency metrics (P50/P95/P99), slow requests, error rates
  - **System Health Report**: Uptime, memory, goroutines, database stats

- **Reports API**: REST API для ручной генерации отчетов (предстоящая версия)
  
### Changed

- **Configuration**: Добавлена секция `reports`
  - `enabled`: включение/отключение scheduler
  - `smtp`: SMTP конфигурация (host, port, username, password, TLS)
  - `schedules`: массив scheduled reports с cron expressions
  - Поддержка переменных окружения для паролей

- **Database Interface**: Добавлены методы для reports statistics
  - `GetUsageStats()`: агрегированная статистика использования
  - `GetPerformanceStats()`: performance metrics за период
  - `CountActiveUsers()`, `CountTotalUsers()`, `CountActiveAPIKeys()`

### Technical

- Новый пакет `internal/reports` с полной реализацией:
  - `scheduler.go`: Cron scheduler с github.com/robfig/cron/v3
  - `generator.go`: Report generation с embedded HTML templates
  - `mailer.go`: SMTP email delivery с gopkg.in/gomail.v2
  - `types.go`: Report models и data structures
  - `templates/*.html`: Beautiful HTML email templates

- SQLite реализация reports statistics методов
  - Percentile calculations для latency metrics
  - JOIN queries для user/model aggregations
  - Optimized queries для больших datasets

- PostgreSQL stub реализация (для будущей поддержки)
- Зависимости: `github.com/robfig/cron/v3`, `gopkg.in/gomail.v2`
- Тесты: Unit tests для всех компонентов
- Graceful shutdown для scheduler

## [1.6.2] - 2025-10-12

### Added

- **Performance Monitoring**: Continuous performance monitoring в production
  - Runtime metrics collection (CPU, memory, goroutines, GC stats)
  - Automatic performance anomaly detection (memory/goroutine leaks)
  - Baseline comparison for performance regression tracking
  - Configurable thresholds для memory и goroutine counts
  
- **Leak Detection**: Автоматическое выявление утечек памяти и goroutines
  - Trend analysis на основе 10 samples (1 sample/minute)
  - Warning alerts при sustained growth (>80% samples показывают рост)
  - Детальная информация о growth rate и duration
  
- **Slow Request Logging**: Автоматическое логирование медленных запросов
  - Middleware для отслеживания request processing time
  - Configurable threshold (default: 5 seconds)
  - Детальная информация: method, path, duration, status, user agent
  
- **pprof Endpoints**: Runtime profiling для deep analysis
  - `/api/admin/pprof/*` endpoints (admin only)
  - CPU profile, heap profile, goroutine dump
  - Memory allocations, block profile, mutex profile
  - Execution trace для advanced debugging

- **Performance API**: REST API для доступа к metrics
  - `GET /api/admin/performance/metrics` - текущие метрики
  - `GET /api/admin/performance/leaks` - leak detection status  
  - `POST /api/admin/performance/reset-baseline` - reset baseline
  - `POST /api/admin/performance/reset-leaks` - reset leak detector

### Changed

- **Configuration**: Добавлена секция `observability.performance`
  - `enabled`: включение/отключение performance monitoring
  - `collection_interval`: интервал сбора метрик (default: 30s)
  - `memory_threshold_mb`: alert threshold для heap memory (default: 1024MB)
  - `goroutine_threshold`: alert threshold для goroutines (default: 1000)
  - `slow_request_threshold`: threshold для slow requests (default: 5s)
  - `gc_percentage`: GOGC value для GC tuning (default: 100)
  - `leak_detection`: включение leak detection (default: true)
  - `pprof_enabled`: включение pprof endpoints (default: true)

### Technical

- Новые пакеты:
  - `internal/observability/perfmon.go` - Performance Monitor Service
  - `internal/observability/leak_detector.go` - Memory/Goroutine Leak Detector
  - `internal/api/middleware/slow_request.go` - Slow Request Logger
  - `internal/api/handlers/performance.go` - Performance API Handler

- Performance Monitor features:
  - Continuous metrics collection в background goroutine
  - Thread-safe metrics access с sync.RWMutex
  - Automatic anomaly detection и alerting
  - Baseline tracking для regression detection
  
- Leak Detector features:
  - 10-sample rolling window для trend analysis
  - Separate tracking для memory и goroutine leaks
  - Configurable thresholds и sensitivity

- pprof integration:
  - Все стандартные pprof handlers через Gin
  - Admin authentication required
  - Configurable enable/disable

- Middleware integration:
  - SlowRequestLogger после TracingMiddleware
  - Minimal performance overhead (<1%)
  - Graceful shutdown support

### Configuration Example

```yaml
observability:
  performance:
    enabled: true
    collection_interval: "30s"
    memory_threshold_mb: 1024
    goroutine_threshold: 1000
    slow_request_threshold: "5s"
    gc_percentage: 100
    leak_detection: true
    pprof_enabled: true
```

### Usage Examples

```bash
# Get current performance metrics
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/admin/performance/metrics

# Get leak detection status
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/admin/performance/leaks

# CPU profile (30 seconds)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/pprof/profile?seconds=30 \
  -o cpu.prof

# Analyze profile
go tool pprof cpu.prof

# Heap profile
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/admin/pprof/heap \
  -o heap.prof

# Goroutine dump
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/admin/pprof/goroutine?debug=1"
```

## [1.6.1] - 2025-10-12

### Added

- **OpenTelemetry Distributed Tracing**: Полная интеграция OpenTelemetry для distributed tracing
  - Поддержка Jaeger и Zipkin экспортеров
  - Автоматическая трассировка всех HTTP запросов
  - W3C Trace Context propagation
  - Настраиваемый sampling rate (0.0 - 1.0)
  - Span annotations с HTTP metadata (method, URL, status code)
  - Error tracking для запросов со status code >= 400

### Changed

- **Configuration**: Добавлена секция `observability.tracing` в конфигурацию
  - `enabled`: включение/отключение трacing
  - `provider`: выбор между "jaeger" или "zipkin"
  - `service_name`: название сервиса в traces
  - `sampling_rate`: процент трассируемых запросов
  - `jaeger.endpoint` и `zipkin.endpoint`: настройка экспортеров

### Technical

- Новый пакет `internal/observability` с TracerProvider
- TracingMiddleware для автоматической трассировки Gin запросов
- Интеграция в router и main.go с graceful shutdown
- Зависимости: go.opentelemetry.io/otel v1.38.0
- Тесты: 100% покрытие для observability и tracing middleware
- Tracing middleware применяется первым в цепочке для полной трассировки

### Configuration Example

```yaml
observability:
  tracing:
    enabled: true
    provider: "jaeger"
    service_name: "ollama-proxy"
    sampling_rate: 1.0
    jaeger:
      endpoint: "http://localhost:14268/api/traces"
```

## [1.5.16] - 2025-10-11

### Changed

- **All WebUI Pages**: Внедрена система уведомлений и модальных окон во все страницы
  - ✅ `dashboard.html` + `dashboard.js` - удаление conversations
  - ✅ `chat.html` + `chat.js` - error messages
  - ✅ `profile.html` + `profile.js` - удаление аккаунта
  - ✅ `tenants.html` + `tenants.js` - удаление тенантов, удаление участников
  - ✅ `usage.html` - подключены CSS/JS
  - ✅ `api-keys.html` + `apikeys.js` - удаление API ключей
  - ✅ `mcp.html` - подключены CSS/JS
  - ✅ `about.html` - подключены CSS/JS

- Все 8 основных страниц теперь используют единую систему уведомлений
- Замены: ~10 alert() → toast notifications, ~8 confirm() → modal confirmations
- Единообразный UX на всех страницах приложения

### Technical

- Подключены `notifications.css` и `notifications.js` во все HTML страницы
- Все JS файлы обновлены для использования `toast.*` и `modal.*` API
- Консистентный порядок загрузки скриптов (auth-guard → api → notifications → компоненты)

## [1.5.15] - 2025-10-11

### Added

- **Toast Notification System**: Профессиональная система всплывающих уведомлений
  - Уведомления появляются справа снизу
  - Зеленые для успешных операций (10 секунд)
  - Красные для ошибок (60 секунд)
  - Желтые для предупреждений (10 секунд)
  - Синие для информационных сообщений (10 секунд)
  - Возможность закрыть вручную кнопкой ×
  - Плавные CSS анимации и transitions
  - Поддержка темной темы
  - Адаптивный дизайн для mobile

- **Modal Confirmation System**: Модальные окна для подтверждения действий
  - Заменяют стандартные browser confirm dialogs
  - Красивый дизайн с иконками и типизацией
  - confirm() - стандартное подтверждение
  - danger() - опасные операции (удаление)
  - warning() - предупреждения с деталями
  - Закрытие по ESC или клику на backdrop
  - Promise-based API для async/await

- **Global API**: `window.toast` и `window.modal` доступны везде
  - `toast.success(message)`, `toast.error(message)`
  - `toast.warning(message)`, `toast.info(message)`
  - `await modal.confirm(message, title)`
  - `await modal.danger(message, title)`
  - `await modal.warning(message, title, details)`

### Changed

- **Admin Panel**: Полностью переведен на новую систему уведомлений
  - Все alert() заменены на toast notifications
  - Все confirm() заменены на modal confirmations
  - Улучшен UX для всех операций (backup, users, API keys, MCP servers)
  - Более информативные сообщения об ошибках

- `web/js/utils/notifications.js` - новый модуль уведомлений
- `web/css/notifications.css` - стили для toast и modal
- `web/admin.html` - подключены новые CSS и JS

### Technical

- Class-based architecture для ToastNotification и ModalConfirmation
- HTML escape для защиты от XSS
- Auto-stacking для нескольких toast уведомлений
- Настраиваемые duration для разных типов уведомлений
- CSS transitions для плавных анимаций (300ms)
- Backdrop blur effect для modal windows
- Mobile-responsive design (media queries)
- Dark theme support через prefers-color-scheme

## [1.5.14] - 2025-10-11

### Added

- **Backup & Restore System**: Полнофункциональная система резервного копирования и восстановления БД
  - `POST /api/admin/backup` - создание бэкапа с автоматическим timestamp
  - `GET /api/admin/backups` - список доступных бэкапов
  - `GET /api/admin/backup/:filename` - скачивание backup файла
  - `POST /api/admin/restore/:filename` - восстановление из бэкапа
  - `DELETE /api/admin/backup/:filename` - удаление старых бэкапов
  - Автоматическая ротация: хранение последних 10 бэкапов
  - Safety backup перед restore операцией
  - Бэкапы сохраняются в `./data/backups/` директорию

### Changed

- `internal/api/handlers/backup.go` - новый handler для backup операций
- `internal/api/router/router.go` - добавлены роуты для backup/restore в admin API
- Все backup операции требуют JWT аутентификацию + admin role

### Technical

- Использование `io.Copy` для эффективного копирования больших файлов
- Path security checks для защиты от path traversal
- Atomic restore with rollback на случай ошибки
- File sync после записи для data integrity
- Structured logging для всех backup операций

## [1.5.13] - 2025-10-11

### Changed

- **Error Type Checking**: Улучшена обработка типов ошибок в admin handlers
  - Реализована функция `isNotFoundError()` с использованием `errors.As`
  - Добавлена функция `isAlreadyExistsError()` для проверки дубликатов
  - Добавлена функция `isInvalidDataError()` для валидационных ошибок
  - Добавлена функция `isPermissionError()` для проверки прав доступа
  - Все функции используют правильную проверку типа `*storage.StorageError`

### Technical

- Использование Go 1.20+ `errors.As()` для type assertions
- Proper unwrapping of wrapped errors
- Type-safe error checking вместо string comparison
- Удален TODO комментарий из admin.go (строка 875)

## [1.5.12] - 2025-10-11

### Fixed

- **BUG-03: WebUI Chat Usage Tracking**: FOREIGN KEY constraint failed при использовании WebUI чата
  - Проблема: `api_key_id` ссылался на несуществующие ключи ("jwt_auth", "unknown")
  - Решение: Сделать `api_key_id` nullable с `ON DELETE SET NULL`
  - Теперь статистика WebUI чата корректно учитывается

### Changed

- **Database Schema (`api_usage` table)**:
  - `api_key_id TEXT` → nullable (было NOT NULL)
  - FOREIGN KEY constraint: `ON DELETE SET NULL` (было CASCADE)
  - Миграция v16 автоматически конвертирует "jwt_auth"/"unknown" → NULL

- **Backend (`internal/models/usage.go`)**:
  - `APIUsage.APIKeyID` → `*string` (nullable)

- **Backend (`internal/api/middleware/usage_tracking.go`)**:
  - При JWT auth устанавливает `nil` вместо "jwt_auth"
  - При отсутствии ключа устанавливает `nil` вместо "unknown"

### Technical

- Migration v16: создает временную таблицу с правильной структурой
- Автоматическая конвертация существующих записей с fake keys
- Пересоздание индексов с учетом nullable значений

## [1.5.11] - 2025-10-11

### Fixed

- **BUG-02: Tenant API Keys Creation**: 404 при создании API ключей для организации
  - Добавлены endpoints для управления tenant API keys
  - `GET /api/tenants/:id/api-keys` - получение списка ключей организации
  - `POST /api/tenants/:id/api-keys` - создание ключа организации
  - `DELETE /api/tenants/:id/api-keys/:key_id` - удаление ключа организации

### Added

- **Tenant API Keys Management**: Полноценное управление API ключами организаций
  - Handler `ListTenantAPIKeys` для получения списка ключей
  - Handler `CreateTenantAPIKey` для создания ключей с правами owner/admin
  - Handler `DeleteTenantAPIKey` для удаления ключей с проверкой владения
  - Автоматическая проверка прав доступа (owner/admin only)
  - Валидация принадлежности ключа к tenant при удалении

### Changed

- **Backend (`internal/api/handlers/tenant.go`)**:
  - NEW: `ListTenantAPIKeys()` - список API ключей организации
  - NEW: `CreateTenantAPIKey()` - создание ключа с scope=tenant
  - NEW: `DeleteTenantAPIKey()` - удаление с проверкой прав

- **Backend (`internal/api/router/router.go`)**:
  - Добавлены routes в tenant group:
    - `GET /:id/api-keys`
    - `POST /:id/api-keys`
    - `DELETE /:id/api-keys/:key_id`

- **Frontend (`web/api-keys.html`, `web/js/apikeys.js`)**:
  - Уже готов к работе с tenant API keys (реализован ранее)
  - Работают tabs Personal/Organization Keys
  - Tenant selector и создание ключей для организации

### Security

- **Role-based Access Control**: Только owner и admin могут создавать/удалять tenant API keys
- **Tenant Ownership Verification**: Проверка принадлежности ключа к tenant перед удалением
- **Membership Check**: Проверка членства пользователя в tenant для всех операций

### Technical

- Default rate limits для tenant keys: 100 req/min, 5000 req/hour, 50000 req/day
- Key scope автоматически устанавливается в `APIKeyScopeTenant`
- TenantID связывается с API key через foreign key
- Plaintext key возвращается только один раз при создании

## [1.5.10] - 2025-10-10

### Fixed

- **Tenant Member Count Display**: Количество участников не отображалось в списке организаций
  - Добавлен SQL подзапрос `COUNT(*)` в `ListUserTenants`
  - Расширена модель `Tenant` (+`MemberCount` поле)
  - Обновлена `scanTenantWithRole` для сканирования `member_count`

### Changed

- **Backend (`internal/models/tenant.go`)**:
  - `Tenant` struct: добавлено поле `MemberCount int`

- **Backend (`internal/storage/sqlite/tenants.go`)**:
  - `ListUserTenants()`: добавлен подзапрос для подсчета членов tenant
  - `scanTenantWithRole()`: обновлен для сканирования поля `member_count`

- **Frontend (`web/js/tenants.js`)**:
  - Уже готов к отображению `tenant.member_count` (строка 115)

### Technical

- SQL Subquery для эффективного подсчета участников
- Минимальные изменения в коде (1 поле в модели, 1 строка SQL, 1 параметр Scan)
- Совместимость с существующим frontend кодом

## [1.5.9] - 2025-10-10

### Fixed

- **Members Display Issue**: Username и email не отображались в списке участников tenant
  - Добавлен SQL JOIN с таблицей `users` в `ListTenantMembers`
  - Расширена модель `TenantMember` (+Username, +Email поля)
  - Новая функция `scanTenantMemberWithUserInfo` для сканирования с JOIN данными

- **Search Results UX**: Улучшена контрастность результатов поиска
  - Черный текст на белом фоне (#000000 / #ffffff)
  - Светло-зеленый фон при hover (rgba(16, 185, 129, 0.15))
  - Темно-серый для email (#4a5568)

- **GetTenant Response Structure**: Исправлена обработка wrapper объекта
  - Frontend теперь правильно извлекает `response.tenant` вместо всего объекта
  - `currentTenant.id` теперь корректно инициализируется в Add Member modal

### Changed

- **Backend (`internal/models/tenant.go`)**:
  - `TenantMember` struct: добавлены поля `Username` и `Email`

- **Backend (`internal/storage/sqlite/tenants.go`)**:
  - `ListTenantMembers()`: обновлен SQL для JOIN с `users` таблицей
  - NEW: `scanTenantMemberWithUserInfo()` для сканирования расширенных данных

- **Frontend (`web/js/tenants.js`)**:
  - `showTenantDetail()`: исправлена обработка `response.tenant` wrapper
  - `addMemberTenantId`: отдельное хранение tenant ID для modal

- **Frontend (`web/css/style.css`)**:
  - `.search-result-item`: белый фон, светло-зеленый hover
  - `.search-result-name`: черный текст (#000000)
  - `.search-result-email`: темно-серый (#4a5568)

### Technical

- SQL Query Optimization: LEFT JOIN с users для получения user info
- Nullable fields handling для username/email
- Cache busting: tenants.js?v=1.5.8.2
- Frontend state management для tenant modals

## [1.5.8] - 2025-10-10

### Added

- **Tenant Membership Check**: Проверка прав доступа к ресурсам tenant
  - Middleware `TenantMembershipMiddleware` для автоматической проверки членства
  - Middleware `RequireTenantRole` для проверки конкретных ролей
  - Проверка членства в `GetTenantUsage` handler (удален TODO)

- **Enhanced Member Management**: Полноценное управление участниками tenant
  - Search endpoint `/api/tenants/:id/search-users` для поиска по username/email
  - Поддержка добавления членов по username, email или user_id
  - UI modal для добавления участников с live search
  - Кнопки "Change Role" и "Remove" для каждого участника
  - Debounced search (500ms) для оптимизации

### Changed

- **Backend (`internal/api/middleware/tenant_membership.go`)**: NEW
  - `TenantMembershipMiddleware` проверяет членство и устанавливает контекст
  - `RequireTenantRole` проверяет конкретные роли

- **Backend (`internal/api/handlers/usage.go`)**:
  - Добавлена проверка членства в `GetTenantUsage`
  - Удален TODO комментарий

- **Backend (`internal/api/handlers/tenant.go`)**:
  - NEW: `SearchUsers()` для поиска пользователей по query
  - Обновлен `AddMember()` для поддержки username/email/user_id
  - Расширенный response с user info

- **Backend (`internal/api/router/router.go`)**:
  - Добавлен роут `GET /api/tenants/:id/search-users`

- **Frontend (`web/tenants.html`)**:
  - NEW: Modal "Add Member" с search функционалом
  - Selected user card
  - Role selector

- **Frontend (`web/css/style.css`)**:
  - Стили для search results dropdown
  - Стили для selected user card
  - "Already Member" badge
  - Form hint styling

- **Frontend (`web/js/api.js`)**:
  - NEW: `searchTenantUsers()` метод
  - Обновлен `addTenantMember()` для username/email

- **Frontend (`web/js/tenants.js`)**:
  - NEW: `showAddMemberModal()`, `hideAddMemberModal()`
  - NEW: `searchUsers()` с debounce
  - NEW: `displaySearchResults()`, `selectUser()`, `deselectUser()`
  - NEW: `submitAddMember()`
  - Обновлен: `removeMember()` с правильными параметрами
  - NEW: `updateMemberRole()` с prompt UI
  - Обновлен `loadMembers()` с кнопками действий

### Security

- Проверка членства для всех tenant-specific ресурсов
- Audit logging при отказе в доступе (403)
- Предотвращение directory traversal
- Role-based access control (RBAC)

### Technical

- Middleware chain для tenant routes
- Context-based role storage
- Debounced search для производительности
- Lazy loading user info

## [1.5.7] - 2025-10-10

### Added

- **Password Change**: Полная реализация смены пароля
  - Endpoint `/api/users/me/password` (POST)
  - Валидация текущего пароля
  - Проверка силы нового пароля
  - Предотвращение повторного использования текущего пароля

### Changed

- **Backend (`internal/models/auth.go`)**:
  - `ChangePasswordRequest` модель
  - `ChangePasswordResponse` модель

- **Backend (`internal/auth/service/auth_service.go`)**:
  - Метод `ChangePassword()` с полной валидацией
  - Проверка текущего пароля через bcrypt
  - Хеширование нового пароля

- **Backend (`internal/storage/database.go`)**:
  - Интерфейс `UpdateUserPassword()` метод

- **Backend (`internal/storage/sqlite/users.go`)**:
  - Реализация `UpdateUserPassword()` для SQLite
  - Эффективное обновление только пароля

- **Backend (`internal/storage/postgresql/users.go`)**:
  - Реализация `UpdateUserPassword()` для PostgreSQL

- **Backend (`internal/api/handlers/user.go`)**:
  - Обновлен `ChangePassword()` для использования `UpdateUserPassword()`
  - Улучшена производительность (обновляется только пароль)

### Security

- Валидация силы пароля (минимум 8 символов, uppercase, lowercase, digit)
- Bcrypt хеширование с DefaultCost
- Audit logging для всех смен пароля
- Проверка на совпадение старого и нового пароля

### Technical

- Транзакционная поддержка через Tx interface
- Консистентная обработка ошибок
- Structured logging с user_id

## [1.5.6] - 2025-10-10

### Improved

- **Modelfile Display**: Улучшено отображение Modelfile в admin панели
  - LICENSE секция автоматически скрывается из Modelfile
  - Только актуальная конфигурация модели отображается
  - LICENSE по-прежнему доступна в отдельной секции

### Changed

- **Frontend (`web/js/admin.js`)**:
  - Новый метод `stripLicenseFromModelfile()` для очистки
  - Поддержка форматов: `LICENSE """` и `LICENSE\n`
  - Автоматический trim результата

- **Frontend (`web/css/style.css`)**:
  - Класс `.code-block-large` для Modelfile (20 строк с прокруткой)
  - Вертикальный и горизонтальный scrollbar

### Technical

- Frontend-only изменение (не требует перекомпиляции)
- Regex парсинг для удаления LICENSE
- License остается в отдельной секции ниже

## [1.5.5] - 2025-10-10

### Fixed

- **Logs Stream File Selection**: Исправлена работа real-time логов
  - SSE stream теперь принимает параметр `?file=filename`
  - Автоматический выбор текущего лог-файла при загрузке
  - Frontend передает выбранный файл в realtime stream

### Changed

- **Backend (`internal/api/handlers/logs.go`)**:
  - `StreamLogs`: добавлен query параметр `file` (default: proxy.log)
  - Directory traversal protection для имени файла
  - Детальное логирование ошибок с именем файла

- **Frontend (`web/js/logs.js`)**:
  - Real-time URL: `?token=xxx&file=yyy`
  - Auto-select текущего файла уже работал, теперь передается в stream

### Technical

- Problem: hardcoded "proxy.log" в stream
- Solution: query param `?file=` + auto-select current

## [1.5.4] - 2025-10-10

### Fixed

- **SSE Logs Stream Auth**: Исправлена регистрация SSE endpoint
  - SSE endpoint теперь вне admin group (обходит JWTAuth middleware)
  - SSEAuthMiddleware применяется первым для чтения токена из query
  - RequireAdmin middleware проверяет admin роль после аутентификации

### Changed

- **Backend (`internal/api/router/router.go`)**:
  - `/api/admin/logs/stream` регистрируется отдельно от admin group
  - Middleware chain: SSEAuthMiddleware → RequireAdmin → Handler
  - Логирование: "SSE logs stream registered with query token auth"

### Technical

- Problem: admin.Use(JWTAuth) блокировал SSE до SSEAuthMiddleware
- Solution: вынести SSE из group, применить middleware напрямую
- Auth flow: query token → JWT validation → admin check → stream

## [1.5.3] - 2025-10-10

### Improved

- **Enhanced Log Parsing**: Полноценный парсер logrus формата
  - Извлечение всех полей из structured logs (key=value)
  - Правильный парсинг quoted и unquoted значений
  - Все context fields отображаются в UI
  - Форматирование timestamp: HH:MM:SS.mmm

### Changed

- **Backend (`internal/api/handlers/logs.go`)**:
  - Новый метод `parseLogrusFields()` для полного парсинга logrus
  - Извлечение time, level, msg и всех дополнительных полей
  - Context fields добавляются в message через separator ` | `

- **Frontend (`web/js/logs.js`)**:
  - Метод `formatTimestamp()` для компактного времени
  - Метод `highlightContextFields()` для подсветки key=value
  - Context fields показываются на второй строке с syntax highlighting

- **Frontend (`web/css/style.css`)**:
  - Улучшенные стили для timestamp и level badges
  - Word-wrap для длинных сообщений
  - VS Code-style подсветка: ключи синие, значения оранжевые

### Technical

- Парсинг: полная поддержка logrus TextFormatter
- Syntax highlighting: key=#569cd6, value=#ce9178 (VS Code theme)
- Timestamp: локальное время вместо UTC строки

## [1.5.2] - 2025-10-10

### Fixed

- **SSE Authentication**: Real-time logs stream теперь работает с JWT auth
  - EventSource не поддерживает custom headers
  - Токен передается через query параметр `?token=xxx`
  - Новый middleware `SSEAuthMiddleware` для SSE endpoints

### Changed

- **Backend (`internal/api/middleware/sse_auth.go`)**:
  - Новый middleware для SSE с поддержкой токена в query
  - Fallback на Authorization header для обратной совместимости

- **Backend (`internal/api/router/router.go`)**:
  - SSE endpoint использует `SSEAuthMiddleware`

- **Frontend (`web/js/logs.js`)**:
  - Токен автоматически добавляется в URL SSE stream
  - Проверка наличия токена перед подключением

### Technical

- SSE + JWT: токен через query параметр
- Security: валидация токена в middleware

## [1.5.1] - 2025-10-10

### Added

- **Enhanced Logs System**: Полноценная система просмотра логов в админке
  - Отдельная вкладка "Logs" в админ-панели
  - Просмотр текущего и архивных лог-файлов
  - Кликабельные фильтры по уровням (DEBUG, INFO, WARN, ERROR)
  - Real-time обновление логов через SSE (Server-Sent Events)
  - Скачивание лог-файлов
  - Dark theme для logs viewer (VS Code style)

### Changed

- **Backend (`internal/api/handlers/logs.go`)**:
  - `LogsHandler`: управление системными логами
  - `ListLogFiles()`: список всех .log файлов в директории
  - `GetLogFile()`: чтение с фильтрацией по уровню и пагинацией
  - `StreamLogs()`: real-time stream через SSE
  - `DownloadLogFile()`: скачивание файлов
  - Парсинг logrus формата в структурированные записи

- **Backend (`internal/api/router/router.go`)**:
  - Routes: `/api/admin/logs`, `/api/admin/logs/:filename`, `/api/admin/logs/stream`
  - Auto-detect logs directory из `config.Logging.FilePath`

- **Frontend (`web/admin.html`)**:
  - Новая вкладка "Logs" с полным UI
  - Selector для выбора лог-файла
  - Level filters (All, DEBUG, INFO, WARN, ERROR)
  - Real-time toggle button
  - Download и Refresh кнопки

- **Frontend (`web/js/logs.js`)**:
  - `LogsViewer` class для управления логами
  - SSE integration для real-time updates
  - Client-side фильтрация по уровням
  - Auto-scroll при новых записях
  - Limit 1000 записей в viewer

- **Frontend (`web/css/style.css`)**:
  - ~170 строк стилей для logs viewer
  - Dark terminal theme (#1e1e1e background)
  - Color-coded log levels
  - Monospace font (Consolas, Monaco)
  - Hover effects и transitions

### Technical

- **SSE Stream**: EventSource API для real-time логов
- **Security**: Path validation против directory traversal
- **Performance**: Tail 500 последних строк по умолчанию
- **Parsing**: Logrus format → structured LogEntry
- **Auto-cleanup**: Max 1000 entries в real-time mode

## [1.4.11] - 2025-10-10

### Added

- **Система "О Системе"**: Новая страница с информацией о версии и историей изменений
  - Отображение версии, git commit, build date, Go version
  - Accordion UI для просмотра changelog всех версий
  - Интеграция данных из CHANGELOG.md в БД через миграцию
  - Пункт "О Системе" в меню профиля пользователя

### Changed

- **Backend (`internal/storage/sqlite/sqlite.go`)**:
  - Migration v3: таблица `changelogs` с версиями из CHANGELOG.md
  - `getChangelogsMigrationWithData()`: SQL INSERT со всеми версиями 1.4.10 → 1.2.0
  
- **Backend (`internal/api/handlers/changelog.go`)**:
  - Новый handler для changelog endpoints
  - `GetSystemInfo()`: информация о версии системы
  - `GetChangelogs()`: список всех changelog записей
  - `GetChangelog(version)`: конкретная версия

- **Backend (`internal/storage/database.go`)**:
  - Добавлены методы: `GetChangelog()`, `ListChangelogs()`
  - SQLite implementation в `internal/storage/sqlite/changelogs.go`
  - PostgreSQL stubs в `internal/storage/postgresql/stubs.go`

- **Backend (`internal/api/router/router.go`)**:
  - `changelogHandler` добавлен в Router struct
  - Routes: `/api/system/info`, `/api/system/changelogs`, `/api/system/changelogs/:version`

- **Frontend (`web/js/components/navbar.js`)**:
  - Добавлен пункт "О Системе" в user dropdown menu
  - Определение `about.html` в `detectCurrentPage()`

- **Frontend (`web/about.html`, `web/js/about.js`)**:
  - Новая страница "О Системе" с версией и changelog
  - Accordion UI для раскрытия/скрытия версий
  - Markdown рендеринг для changelog контента
  - Responsive design с loading states

- **Frontend (`web/css/style.css`)**:
  - ~100 строк стилей для changelog accordion
  - Hover effects, transitions, typography

### Technical

- **Database Schema**: таблица `changelogs` (version, release_date, content, created_at)
- **Migration System**: автоматическая миграция v3 при старте сервера
- **API Endpoints**: Public endpoints (no auth required)
- **Cursor Rules**: добавлено правило `.cursor/rules/changelog-migration`
  - Workflow для обновления CHANGELOG → SQL миграция → VERSION
  - Инструкции для будущих релизов

### Workflow для будущих релизов

1. Обнови `CHANGELOG.md` с новыми изменениями
2. Обнови SQL в `getChangelogsMigrationWithData()` (добавь новую версию)
3. Обнови файл `VERSION`
4. Коммит всех 3 файлов вместе
5. Миграция применится автоматически при старте сервера

---

## [1.4.10] - 2025-10-10

### Added

- **"Remember Me" Функция при входе**:
  - Checkbox "Не выходить из системы 24 часа" на форме логина
  - При включении галочки access token живет **24 часа** вместо 15 минут
  - Refresh token продолжает работать 7 дней как и раньше
  - Логирование использования remember_me в JWT и auth service

### Changed

- **Backend (`internal/auth/service/auth_service.go`)**:
  - `LoginRequest`: добавлено поле `RememberMe bool`
  - Передача параметра `RememberMe` в `GenerateTokenPair`
  - Логирование статуса remember_me при успешном логине

- **Backend (`internal/auth/jwt/jwt.go`)**:
  - `GenerateTokenPair`: новый variadic параметр `rememberMe ...bool`
  - Access token duration: 15m (default) → 24h (если rememberMe = true)
  - Логирование duration и remember_me статуса

- **Frontend (`web/login.html`)**:
  - Добавлен checkbox "Не выходить из системы 24 часа"
  - JavaScript отправляет `remember_me: true/false` в `/api/auth/login`

### Technical

**Текущие настройки сессии:**

- Без галочки: Access token 15 минут, Refresh token 7 дней
- С галочкой: Access token 24 часа, Refresh token 7 дней
- Refresh token автоматически обновляет access token
- Фактическая длительность сессии: до 7 дней (или до logout)

---

## [1.4.9] - 2025-10-10

### Fixed

- **API Keys Status Display**:
  - Problem: Свежесозданные ключи отображались как "Inactive" в WebUI
  - Root Cause: Frontend проверял несуществующее поле `is_active`, в то время как backend возвращал `status`
  - Solution: Frontend теперь использует поле `status` из API response
  - Logic: Ключ считается активным если `status === 'active'` И не истек срок действия

### Changed

- **Frontend (`web/js/apikeys.js`)**:
  - `renderKeyRow()`: использует `key.status` вместо `key.is_active`
  - Добавлена проверка срока действия ключа (`expires_at`)
  - Логика: `isActive = key.status === 'active' && (!key.expires_at || new Date(key.expires_at) > new Date())`

### Technical

- Backend возвращает `status` со значениями: "active", "disabled", "expired", "revoked"
- Frontend маппит это в UI badges: Active (green) / Inactive (gray)
- Проверка expiration на стороне клиента для точности отображения

---

## [1.4.8] - 2025-10-10

### Added

- **Debug Logging для Usage Tracking**:
  - `internal/api/middleware/usage_tracking.go`: детальное логирование извлеченных данных из контекста
  - `internal/api/handlers/chat.go`: логирование установки токенов в контекст
  - `internal/storage/sqlite/usage.go`: логирование успешной записи в БД
  - Поля в логах: user_id, api_key_id, model, tokens, success, endpoint

### Fixed

- **API Key ID для JWT аутентификации**:
  - Problem: При использовании JWT auth поле `api_key_id` было NULL (NOT NULL constraint)
  - Solution: Используется специальный ID "jwt_auth" для JWT аутентификации, "unknown" для других случаев
  - Теперь все записи имеют валидный `api_key_id`

### Changed

- **Usage Tracking Middleware**:
  - Обязательная установка `api_key_id` для всех типов аутентификации
  - Fallback значения: "jwt_auth" (JWT), "unknown" (other)
  - Улучшенное логирование для отладки проблем с токенами

### Technical

- Debug level logs для troubleshooting usage statistics
- Логи показывают полный flow: handler → context → middleware → database
- Позволяет определить на каком этапе теряются данные (tokens, api_key_id)

---

## [1.4.7] - 2025-10-10

### Fixed

- **CRITICAL: Usage Statistics Performance**
  - **Problem**: Запросы к `/api/usage/personal` выполнялись 5+ секунд и завершались ошибкой "context canceled"
  - **Root Cause**: N+1 queries problem - для каждой модели/endpoint выполнялся отдельный SQL запрос
  - **Solution**: Оптимизированы SQL запросы с использованием `SUM(CASE WHEN ...)` для агрегации в одном запросе

### Changed

- **Database Optimizations:**
  - `internal/storage/sqlite/usage.go`:
    - `GetUserUsageStats()`: объединены запросы для model/endpoint success rates
    - `GetTenantUsageStats()`: объединены запросы для model/endpoint success rates
    - Добавлен LIMIT 10 для API keys (только топ-10 ключей)
    - Исправлен расчет success_rate (было * 100, теперь float 0-1)
  - `internal/storage/sqlite/sqlite.go`:
    - Добавлены составные индексы для ускорения запросов:
      - `idx_api_usage_user_created` на (user_id, created_at DESC)
      - `idx_api_usage_tenant_created` на (tenant_id, created_at DESC)
    - Все индексы теперь с `IF NOT EXISTS` для безопасной повторной миграции

### Performance Impact

- **Before**: 5+ секунд, timeout, context canceled
- **After**: < 100ms для большинства запросов
- **Query Reduction**: Было N+M запросов (N моделей + M endpoints), стало 5 запросов (aggregate + models + endpoints + api_keys + recent)
- **Index Usage**: Составные индексы покрывают 95%+ запросов к api_usage таблице

### Technical

- SQL optimization: `SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END)` вместо отдельных COUNT запросов
- Composite indexes: покрывают фильтры WHERE user_id = ? AND created_at >= ?
- LIMIT optimization: ограничение результатов для API keys и recent requests

---

## [1.4.6] - 2025-10-10

### Added

- **API Usage Tracking**: Полная интеграция middleware для записи статистики использования
  - Middleware: `internal/api/middleware/usage_tracking.go`
  - Трекинг для всех режимов аутентификации (Hybrid, API Key, No Auth)
  - Автоматическая запись в `api_usage` таблицу
  - Сбор метрик: user_id, model, tokens, duration, success/error
  - Поддержка streaming requests с приблизительным подсчетом токенов
  - Асинхронная запись в БД (не блокирует response)

### Changed

- **Backend:**
  - `internal/api/router/router.go`:
    - Включен `UsageTracking` middleware для всех auth modes
    - Для API Key mode: использует новый middleware если доступна БД
    - Для No Auth mode: также включен tracking для мониторинга
  - `internal/api/handlers/chat.go`:
    - Установка токенов в контекст (`c.Set`) для non-streaming
    - Передача prompt_tokens, completion_tokens, total_tokens в middleware
  - `internal/api/handlers/streaming.go`:
    - Установка completion_tokens в контекст при завершении stream
    - Сохранение частичных токенов при ошибках
    - Сохранение токенов при client disconnect
  - `internal/api/middleware/usage_tracking.go`:
    - Извлечение токенов из контекста вместо заглушек
    - Новая функция `extractIntFromContext()`
    - Поддержка реальных данных usage
  - `internal/models/usage.go`:
    - Добавлены поля для совместимости с frontend
    - ErrorRate, AvgDuration, Models (массив), APIKeys, RecentRequests
  - `internal/storage/sqlite/usage.go`:
    - Улучшена `GetUserUsageStats()` для заполнения всех полей
    - Добавлены запросы для API keys usage
    - Добавлены запросы для recent requests
    - Правильный расчет success_rate и error_rate

### Fixed

- **User Dropdown**: Исправлено некликабельное меню профиля
  - CSS: поддержка классов `.active` и `.show`
  - Удалены дублирующиеся event listeners из всех страниц
  - Централизованное управление через `navbar.js` компонент
- **Clipboard API**: Исправлена ошибка копирования имени модели
  - Добавлена проверка доступности `navigator.clipboard`
  - Автоматический fallback на `execCommand` если недоступен
- **Usage Statistics Page**: Исправлено отображение пустых данных
  - Backend теперь возвращает правильные поля для frontend
  - Добавлена совместимость между snake_case и camelCase

### Technical

- Асинхронная запись usage с таймаутом 5 секунд
- Отдельный контекст для background записи (не зависит от request context)
- Приблизительный подсчет токенов для streaming: length / 4
- Graceful handling при отсутствии БД (fallback на legacy методы)
- Логирование ошибок записи usage без прерывания ответа

---

## [1.4.5] - 2025-10-10

### Added

- **MCP Servers Catalog**: Admin-managed catalog для MCP серверов
  - Database: таблица `mcp_servers` с migration v2
  - Backend models: `MCPServer`, requests/responses, categories
  - Storage: SQLite implementation + PostgreSQL stubs
  - API Handlers: CRUD operations для MCP серверов
  - Public endpoints: `/api/mcp/servers`, `/api/mcp/categories`
  - Admin endpoints: `/api/admin/mcp/servers` (create/update/delete)
  - Категории: development, productivity, database, cloud, ai, other
  - Поля: name, description, installation_guide, website_url, github_url, tags
  - Фильтры: category, search, active_only, сортировка

### Changed

- **Backend:**
  - `internal/models/mcp.go`: новые модели для MCP серверов
  - `internal/storage/database.go`: добавлены методы в interface
  - `internal/storage/sqlite/sqlite.go`: migration v2 для mcp_servers
  - `internal/storage/sqlite/mcp.go`: полная CRUD реализация
  - `internal/storage/postgresql/postgresql.go`: migration v2
  - `internal/storage/postgresql/stubs.go`: stubs для MCP методов
  - `internal/api/handlers/mcp.go`: новый handler для MCP
  - `internal/api/router/router.go`:
    - Добавлен `mcpHandler` в Router struct
    - Инициализация в `setupHandlers()`
    - Новая функция `setupMCPRoutes()`
    - Public + admin routes с JWT/API key auth

- **Frontend:**
  - `web/mcp.html`: публичная страница каталога
  - `web/js/mcp.js`: full-featured catalog (list, filters, search, details modal)
  - `web/css/style.css`: ~380 строк стилей для MCP catalog
  - Features: category badges, grid layout, modal details, responsive design

### Fixed

- **Transaction Interface**: добавлены MCP методы в `sqliteTx` и `postgresqlTx`
  - Исправлена ошибка компиляции "does not implement storage.Tx"
  - Добавлена делегация всех 5 MCP методов в транзакциях

### Technical

- Migration system: автоматическое применение при старте
- JSON tags: массивы тегов в SQLite (TEXT), PostgreSQL (JSONB)
- Access control: public read, admin write
- Filtering: category, search (name/description/tags), active status
- Pagination: limit/offset с total count
- Modal: server details с installation guide, links, tags
- Grid: auto-responsive card layout
- Search: debounced 300ms для performance

---

## [1.4.4] - 2025-10-10

### Added

- **Enhanced Models Information**: Accordion UI с детальной информацией о моделях
  - Lazy loading деталей модели через `/api/admin/models/:name/details`
  - Ollama `/api/show` integration для получения model info
  - Accordion компонент с expand/collapse анимацией
  - Секции: Basic Info, Specifications, Template, Modelfile, License, Parameters
  - Copy button интегрирован в header каждой модели (из v1.4.1)

### Changed

- **Backend:**
  - `internal/models/ollama.go`: новые структуры для Ollama model details
  - `internal/client/ollama/models.go`: добавлено поле `ModelInfo` в `ShowResponse`
  - `internal/api/handlers/admin.go`:
    - Добавлено поле `ollamaClient` в `AdminHandler`
    - Новый метод `GetModelDetails()` для endpoint
    - Helper functions для парсинга model info
  - `internal/api/router/router.go`:
    - Новый route `/api/admin/models/:name/details`
    - `SetOllamaClient()` для AdminHandler при инициализации

- **Frontend:**
  - `web/admin.html`: заменена таблица моделей на accordion
  - `web/js/admin.js`:
    - `renderModels()`: создание accordion вместо таблицы
    - `toggleAccordion()`: управление состоянием accordion
    - `loadModelDetails()`: async загрузка деталей с API
    - `renderModelDetails()`: рендеринг секций с информацией
    - `sanitizeId()`: helper для безопасных ID
  - `web/css/style.css`: ~230 строк стилей для accordion, details, responsive

### Technical

- Lazy loading: детали загружаются только при клике на модель
- Accordion: только одна модель открыта одновременно
- Error handling: graceful degradation при ошибках API
- Responsive: адаптивная верстка для mobile
- Code blocks: подсветка для template/modelfile/license
- Grid layout: автоматическая колоночность для деталей

---

## [1.4.3] - 2025-10-10

### Added

- **Build Version Information**: Реальная версия вместо "dev" через ldflags
  - Новый package `internal/version` с Version, GitCommit, BuildDate, GoVersion
  - Флаг `-version` для показа полной информации о билде
  - VERSION файл как единственный источник версии
  - Git commit hash в version info
  - Build date и Go version в метаданных

### Changed

- **Makefile**: обновлены LDFLAGS для внедрения version info
  - VERSION читается из файла VERSION
  - Добавлена команда `make version` для показа текущей версии
  - LDFLAGS используют `ollama-openai-proxy/internal/version.*`
- **build.sh/build.ps1**: обновлены для использования нового version package
  - Версия читается из файла VERSION
  - Ldflags обновлены на правильный import path
- **internal/logger/logger.go**: используется `version.Version` вместо hardcoded "dev"
- **internal/api/handlers/health.go**:
  - Используется `version.GetInfo()` для полной version info
  - Реальный uptime через global `startTime`
  - Расширенный JSON response с git_commit, build_date, go_version
- **cmd/server/main.go**:
  - Добавлен flag parsing для `-version`
  - Используется `version.Short()` при старте
  - Логирование с полной version info

### Technical

- Удалены старые глобальные переменные `Version` и `BuildTime` из main.go
- Все TODO комментарии о версионировании удалены (2 штуки)
- Централизованное управление версией через VERSION файл
- Build scripts автоматически внедряют version metadata

---

## [1.4.2] - 2025-10-10

### Security

- **TUI Admin Key Configuration**: Убраны hardcoded admin API keys из TUI
  - Admin token теперь загружается из конфигурации (`auth.admin_key`)
  - Добавлен флаг `--config` для указания пути к конфигу
  - Валидация наличия admin_key при запуске TUI
  - Удалены все 4 TODO комментария с hardcoded ключами

### Changed

- `cmd/tui/main.go`: добавлен импорт config package
- Model struct: добавлено поле `adminToken`
- main(): загрузка конфигурации и извлечение admin token
- createAPIKeyCmd(): принимает adminToken как параметр
- Методы Update() и executeConfirmedAction(): используют m.adminToken

### Technical

- Удалена константа `adminAPIKey = "your-admin-key-here"`
- Все HTTP requests используют token из конфигурации
- Функция `loadConfig()` для загрузки и валидации конфига

---

## [1.4.1] - 2025-10-10

### Added

- **Model Copy Button**: Добавлена кнопка копирования названия модели в буфер обмена
  - Clipboard API с fallback для старых браузеров
  - Toast notifications для success/error feedback
  - Интеграция в Admin Panel → Models list
  - Font Awesome icons для UI
  - Keyboard accessible (Tab + Enter)
  - Mobile responsive design

### Changed

- Обновлен `web/js/admin.js`: улучшен рендеринг списка моделей
- Обновлен `web/css/style.css`: добавлены стили для copy button и toast notifications

### Technical

- Создан utility module `web/js/utils/clipboard.js` для работы с clipboard
- Подключен Font Awesome 6.4.0 CDN в admin.html
- Поддержка старых браузеров через document.execCommand fallback

---

## [1.4.0] - Base Release

Начальная версия релиза 1.4.0 - WebUI Enhancements & Code Quality.

**Roadmap версий:**

- v1.4.1: Model Copy Button ✅
- v1.4.2: TUI Admin Key Config ✅
- v1.4.3: Build Version Info ✅
- v1.4.4: Enhanced Models Info ✅
- v1.4.5: MCP Catalog ✅ ← **Current** (Version 1.4.0 COMPLETE!)
- v1.5.0: Operations & Maintenance (next)

---

## [1.3.0] - 2025-10-06

User Experience & Multi-Tenancy - завершен 100%.

### Added

- Database Abstraction Layer (SQLite + PostgreSQL)
- User Authentication & Multi-Tenancy (JWT, RBAC)
- Interactive Chat Interface (ChatGPT-like UI)
- User Dashboard
- Cross-platform Build System

---

## [1.2.0] - Previous Release

Enhanced Monitoring & Management - завершен 100%.

### Added

- TUI Request Monitor
- WebUI Metrics Visualization
- Advanced Logs Features
- Enhanced API Key Management
