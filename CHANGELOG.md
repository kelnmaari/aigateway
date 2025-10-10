# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
