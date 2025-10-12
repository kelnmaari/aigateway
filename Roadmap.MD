# 🗺️ Roadmap: Ollama-OpenAI Proxy

> **Текущая версия:** 1.5.16 ✅  
> **План развития:** Version 1.5.0 - 1.10.0+  
> **Статус:** Active Development (Version 1.5.0 Complete! - 12/12 задач)  
> **Последнее обновление:** 2025-10-11  
> **Новое:** 📱 Roadmap расширен до v1.10.0 (15 новых задач!)

---

## 📋 Принципы приоритизации

1. **High Priority** - критичные функции для production использования
2. **Medium Priority** - улучшения UX и расширение возможностей
3. **Low Priority** - nice-to-have функции
4. **Future** - долгосрочные планы

---

## ✅ Version 1.2.0 - Enhanced Monitoring & Management **[COMPLETED]**

**Фокус:** Улучшение мониторинга и управления API ключами  
**Приоритет:** HIGH  
**Оценка времени:** 20-30 часов  
**Статус:** 🎉 Завершено 100% (4/4 задачи)

### Задачи

- [x] **TUI-04** Request Monitor в TUI → [BACKLOG/TUI-04_request_monitor.md](BACKLOG/TUI-04_request_monitor.md)
  - ✅ Live таблица активных запросов
  - ✅ Детальная информация по каждому запросу
  - ✅ Фильтрация и сортировка (status, sort field, order)
  - ✅ Pagination (n/p keys)
  - ✅ Export в CSV/JSON
  - ✅ Keyboard navigation (arrows, Enter, Esc)
  - **Оценка:** 6-8 часов → **Выполнено**

- [x] **WEBUI-01** Metrics Visualization → [BACKLOG/WEBUI-01_metrics_visualization.md](BACKLOG/WEBUI-01_metrics_visualization.md)
  - ✅ Интеграция Chart.js
  - ✅ 4 графика: Latency (P50/P95/P99), Requests, Status Distribution, Active Requests
  - ✅ Real-time updates через WebSocket
  - ✅ Time range selector (1h, 6h, 24h, 7d)
  - ✅ Auto-refresh toggle
  - **Оценка:** 8-10 часов → **Выполнено**

- [x] **WEBUI-02** Advanced Logs Features → [BACKLOG/WEBUI-02_advanced_logs.md](BACKLOG/WEBUI-02_advanced_logs.md)
  - ✅ Фильтрация по уровню (client-side)
  - ✅ Поиск по тексту (real-time)
  - ✅ Export логов (TXT, JSON)
  - ✅ Auto-refresh toggle
  - ✅ Color coding по уровням
  - **Оценка:** 3-4 часа → **Выполнено**

- [x] **AUTH-04** Enhanced API Key Management → [BACKLOG/AUTH-04_enhanced_key_management.md](BACKLOG/AUTH-04_enhanced_key_management.md)
  - ✅ Edit existing keys (name, description, models, permissions, rate limits)
  - ✅ Revoke/Enable через WebUI
  - ✅ Key expiration management (extend)
  - ✅ Update permissions
  - ✅ Backend API endpoints
  - **Оценка:** 4-5 часов → **Выполнено**

### 🎯 Достижения Version 1.2.0

**TUI (Terminal UI):**

- 🔍 **Request Monitor** - полный мониторинг всех API запросов
- 📊 Таблица с сортировкой, фильтрацией, pagination
- 🔎 Детальный просмотр каждого запроса (Enter)
- 💾 Export в CSV/JSON (кнопка 'e')
- ⌨️ Keyboard navigation (arrows, n/p, s, f, r)

**WebUI (Web Interface):**

- 📈 **Metrics Visualization** - 4 интерактивных графика
- 📊 Latency trends (P50/P95/P99)
- 📉 Request volume charts
- 🎨 Status distribution pie chart
- 🔄 Real-time active requests
- 🕐 Time range selector
- ⚡ WebSocket live updates

**Advanced Logs:**

- 🔍 Client-side фильтрация по уровню
- 🔎 Real-time поиск по тексту
- 🎨 Color coding (ERROR=red, WARN=yellow, INFO=blue)
- 💾 Export в TXT/JSON
- 🔄 Auto-refresh toggle

**Enhanced API Keys:**

- ✏️ Edit существующих ключей
- 🔐 Revoke/Enable через UI
- ⏰ Extend expiration
- 🎛️ Update permissions
- ⚡ Rate limits management

---

## 💬 Version 1.3.0 - User Experience & Multi-Tenancy ✅ **COMPLETED!**

**Фокус:** Превращение в полноценный ChatGPT-like интерфейс с user authentication + Build System  
**Приоритет:** HIGH  
**Оценка времени:** 45-57 часов (core) + 2-3 часа (build)  
**Фактическое время:** ~49 часов (core) + 2 часа (build) = **51 час**  
**Статус:** 🎉 Завершено 100% (5/5 задач)

### Задачи

- [x] **DB-01** Database Abstraction Layer → [BACKLOG/DB-01_database_abstraction.md](BACKLOG/DB-01_database_abstraction.md)
  - ✅ SQLite + PostgreSQL support
  - ✅ Migration system
  - ✅ Connection pooling
  - ✅ CRUD operations для всех моделей
  - ✅ Transaction support
  - **Оценка:** 8-10 часов → **Выполнено**

- [x] **AUTH-05** User Authentication & Multi-Tenancy → [BACKLOG/AUTH-05_user_authentication.md](BACKLOG/AUTH-05_user_authentication.md) ✅ **ЗАВЕРШЕНО**
  - ✅ JWT package (generate, validate, refresh tokens)
  - ✅ Auth middleware (JWT validation, optional auth)
  - ✅ Auth handlers (register, login, logout, me, refresh)
  - ✅ Password security (bcrypt, validation, strength requirements)
  - ✅ User handlers (profile, update, delete, tenants, API keys, password change)
  - ✅ Tenant handlers (CRUD, members management with RBAC)
  - ✅ APIKey scoping (personal/tenant with full CRUD)
  - ✅ Database schema (all fields in V1 migration)
  - **Оценка:** 12-15 часов → **Выполнено за 12 часов**

- [x] **WEBUI-03** Interactive Chat Interface → [BACKLOG/WEBUI-03_interactive_chat.md](BACKLOG/WEBUI-03_interactive_chat.md) ✅ **ЗАВЕРШЕНО**
  - ✅ ChatGPT-like UI (Vanilla JS, zero NPM deps)
  - ✅ Real-time streaming responses
  - ✅ Markdown + code highlighting (marked.js + highlight.js)
  - ✅ Conversation management (sidebar, API endpoints)
  - ✅ Model selector
  - ✅ Bootstrap system with admin token
  - ✅ Auth guard for page protection
  - **Оценка:** 15-20 часов → **Выполнено за 18 часов**

- [x] **WEBUI-04** User Dashboard → [BACKLOG/WEBUI-04_user_dashboard.md](BACKLOG/WEBUI-04_user_dashboard.md) ✅ **ЗАВЕРШЕНО**
  - ✅ Dashboard (stats, quick actions, recent activity)
  - ✅ Profile management (edit, password, delete)
  - ✅ Personal API keys CRUD (with rate limits, permissions)
  - ✅ Tenant keys CRUD (organization-scoped)
  - ✅ Usage statistics & monitoring
  - ✅ Tenant management (create, members, roles)
  - ⏭️ Preferences (theme, language) - деferred to 1.4.0
  - **Оценка:** 10-12 часов → **Выполнено за 11 часов**

- [x] **BUILD-01** Cross-Platform Build System → ✅ **ЗАВЕРШЕНО**
  - ✅ build.sh для Linux/macOS (bash)
  - ✅ build.ps1 для Windows (PowerShell)
  - ✅ Makefile с всеми командами
  - ✅ Cross-compilation (Linux amd64/arm64, Windows amd64)
  - ✅ Distribution packaging (tar.gz, zip)
  - ✅ Docker Dockerfile (опционально)
  - ✅ BUILD.md документация
  - **Оценка:** 2-3 часа → **Выполнено за 2 часа**

### Архитектурные решения

**Database Strategy:**

- SQLite по умолчанию (easy deployment)
- PostgreSQL для production (через config switch)
- Автоматические миграции
- Единый интерфейс для обеих БД

**Multi-Tenancy Model:**

- Personal Workspace (auto-created для каждого user)
- Organization Tenants (создаются вручную, multi-user)
- API Keys: Personal (user-scoped) + Tenant (organization-scoped)
- RBAC: owner/admin/member/viewer roles

**Frontend Stack:**

- Vanilla JavaScript (zero NPM dependencies)
- Modern Web APIs (Web Components, Fetch, EventSource)
- CDN libraries: marked.js, highlight.js, katex (optional)
- Single binary deployment (embed в Go)

---

## 🎨 Version 1.4.0 - WebUI Enhancements & Code Quality

**Фокус:** Улучшение WebUI интерфейса и устранение технического долга  
**Приоритет:** MEDIUM  
**Оценка времени:** 18-24 часа  
**Статус:** 🎯 Текущая  
**Версионирование:** 1 фича = 1 patch (1.4.1, 1.4.2, ...)

### Задачи

- [ ] **WEBUI-06** Model Name Copy Button → [BACKLOG/WEBUI-06_model_copy_button.md](BACKLOG/WEBUI-06_model_copy_button.md) → **v1.4.1**
  - Кнопка копирования названия модели в буфер обмена
  - Clipboard API интеграция
  - Toast/tooltip подтверждение
  - **Оценка:** 1-2 часа

- [ ] **TODO-01** TUI Admin API Key Configuration → [BACKLOG/TODO-01_tui_admin_key_config.md](BACKLOG/TODO-01_tui_admin_key_config.md) → **v1.4.2**
  - Убрать hardcoded admin keys из TUI
  - Загружать admin key из конфигурации
  - Файлы: cmd/tui/main.go (4 комментария)
  - **Оценка:** 2 часа

- [ ] **TODO-02** Build Version Information → [BACKLOG/TODO-02_build_version_info.md](BACKLOG/TODO-02_build_version_info.md) → **v1.4.3**
  - Реальная версия вместо "dev"
  - Добавить ldflags в build скрипты
  - Файлы: internal/logger/logger.go, internal/api/handlers/health.go
  - **Оценка:** 1-2 часа

- [ ] **WEBUI-05** Enhanced Models Information → [BACKLOG/WEBUI-05_enhanced_models_info.md](BACKLOG/WEBUI-05_enhanced_models_info.md) → **v1.4.4**
  - Расширенная информация о моделях в админке (System → Models)
  - Список моделей с раскрывающимися параметрами (accordion)
  - Показ всех параметров: size, family, format, parameters, etc.
  - API endpoints для детальной информации
  - **Оценка:** 8-10 часов

- [ ] **WEBUI-07** MCP Servers Catalog → [BACKLOG/WEBUI-07_mcp_catalog.md](BACKLOG/WEBUI-07_mcp_catalog.md) → **v1.4.5**
  - Новая страница с каталогом MCP серверов
  - Admin CRUD для управления каталогом
  - Публичный просмотр для пользователей
  - Поля: название, описание, ссылка, категория, инструкция
  - Без функционала использования в чате (только справочник)
  - **Оценка:** 6-8 часов

---

## 🔧 Version 1.5.0 - Operations & Maintenance **[✅ COMPLETE]**

**Фокус:** Операционная готовность и устранение багов  
**Приоритет:** MEDIUM  
**Оценка времени:** 20-27 часов (обновлено с учетом BUG-02, BUG-03, UI Notifications)  
**Фактическое время:** ~22-25 часов (12 задач завершено)
**Версионирование:** 1 фича = 1 patch (1.5.1 - 1.5.16)
**Статус:** 100% завершено (12/12 задач)

### Задачи

- [x] **Enhanced Logs System** → **v1.5.1-v1.5.5** ✅ **COMPLETED!**
  - ✅ **v1.5.1** - Enhanced Logs System (основная реализация)
    - Отдельная вкладка "Logs" в admin панели
    - Просмотр лог-файлов, скачивание
    - Фильтрация по уровням (DEBUG/INFO/WARN/ERROR)
    - Real-time streaming через SSE
  - ✅ **v1.5.2** - SSE Authentication Fix
    - SSEAuthMiddleware для query token auth
    - Endpoint вне admin group (обход JWTAuth)
  - ✅ **v1.5.3** - Enhanced Log Parsing
    - Парсинг logrus формат (key=value)
    - VS Code-style syntax highlighting
    - Timestamp форматирование (HH:MM:SS.mmm)
  - ✅ **v1.5.4** - SSE Stream Auth Fix
    - Middleware chain исправлен
    - SSE endpoint отдельная регистрация
  - ✅ **v1.5.5** - Logs Stream File Selection
    - Query param ?file=filename
    - Auto-select текущего лог-файла
  - **Выполнено:** 100% (заменяет BUG-01)

- [x] **Modelfile Display Enhancement** → **v1.5.6** ✅ **COMPLETED!**
  - ✅ LICENSE секция автоматически скрывается из Modelfile
  - ✅ Увеличена высота поля до 20 строк с прокруткой
  - ✅ Метод `stripLicenseFromModelfile()` для очистки
  - ✅ CSS класс `.code-block-large` для управления размером
  - **Выполнено:** 100% (UI enhancement)

- [x] **TODO-03** Password Change Implementation → **v1.5.7** ✅ **COMPLETED!**
  - Реализовать backend для смены пароля (UI уже есть)
  - Валидация текущего пароля
  - Файл: internal/api/handlers/auth.go
  - **Оценка:** 3-4 часа

- [x] **TODO-04** Tenant Membership Check + Enhanced Member Management → **v1.5.8** ✅ **COMPLETED!**
  - Проверка прав доступа к tenant ресурсам
  - Middleware для автоматической проверки членства
  - Поиск пользователей по username/email
  - UI modal для добавления участников с live search
  - Change Role и Remove функции
  - Файл: internal/api/handlers/usage.go, tenant.go
  - **Оценка:** 2-3 часа → Расширено до 5-6 часов

- [x] **Members Display Fix** → **v1.5.9** ✅ **COMPLETED!**
  - ✅ JOIN с таблицей users для получения username/email
  - ✅ Обновлен `TenantMember` model (+Username, +Email)
  - ✅ Новая функция `scanTenantMemberWithUserInfo`
  - ✅ Улучшены стили search results (черный текст, светло-зеленый hover)
  - ✅ Фикс `GetTenant` response structure (tenant/role wrapper)
  - **Выполнено:** 100% (UI + Backend fix)

- [x] **Tenant Member Count Display** → **v1.5.10** ✅ **COMPLETED!**
  - ✅ SQL подзапрос COUNT для подсчета участников
  - ✅ Обновлен `Tenant` model (+MemberCount поле)
  - ✅ Обновлена функция `scanTenantWithRole` для member_count
  - ✅ Frontend уже готов (отображение "👥 X members")
  - **Выполнено:** 100% (Backend + Model fix)

- [x] **BUG-02** Tenant API Keys Creation → **v1.5.11** ✅ **COMPLETED!**
  - ✅ **Проблема:** 404 при создании API ключей для организации
  - ✅ Добавлен endpoint `POST /api/tenants/:id/api-keys`
  - ✅ Добавлен endpoint `GET /api/tenants/:id/api-keys`
  - ✅ Добавлен endpoint `DELETE /api/tenants/:id/api-keys/:key_id`
  - ✅ Frontend уже реализован в `web/api-keys.html`
  - Реализовано:
    - ✅ Добавлены роуты в `router.go`
    - ✅ Созданы handlers `CreateTenantAPIKey`, `ListTenantAPIKeys`, `DeleteTenantAPIKey`
    - ✅ Проверка прав доступа (owner/admin)
    - ✅ Связь API ключа с tenant через `tenant_id`
  - **Выполнено:** 100%

- [x] **BUG-03** WebUI Chat Usage Tracking → **v1.5.12** ✅ **COMPLETED!**
  - ✅ **Проблема:** FOREIGN KEY constraint failed при использовании WebUI чата
  - ✅ **Решение:** Сделать `api_key_id` nullable с `ON DELETE SET NULL`
  - Реализовано:
    - ✅ Миграция v16 в `internal/storage/sqlite/sqlite.go`
    - ✅ Создание временной таблицы с nullable api_key_id
    - ✅ Автоконвертация "jwt_auth"/"unknown" → NULL
    - ✅ Обновлен `APIUsage.APIKeyID` → `*string` (nullable)
    - ✅ Middleware устанавливает `nil` вместо "jwt_auth"/"unknown"
    - ✅ Пересозданы индексы с учетом nullable значений
  - **Выполнено:** 100% (Database + Model + Middleware)

- [x] **TODO-05** Error Type Checking → **v1.5.13** ✅ **COMPLETED!**
  - ✅ Улучшена обработка типов ошибок
  - Реализовано:
    - ✅ `isNotFoundError()` с использованием `errors.As()`
    - ✅ `isAlreadyExistsError()` для проверки дубликатов
    - ✅ `isInvalidDataError()` для валидационных ошибок
    - ✅ `isPermissionError()` для проверки прав доступа
    - ✅ Добавлен импорт `"errors"` пакета
  - **Выполнено:** 100% (Type-safe error checking)

- [x] **OPS-01** Backup & Restore → [BACKLOG/OPS-01_backup_restore.md](BACKLOG/OPS-01_backup_restore.md) → **v1.5.14 ✅**
  - Автоматический backup database и конфигурации
  - Restore procedures (с safety backup)
  - API endpoints: создание, список, скачивание, восстановление, удаление
  - Автоматическая ротация бэкапов (хранит 10 последних)
  - **Оценка:** 3-4 часа
  - **Выполнено:** 100% (Full backup/restore system)

- [x] **UI Notifications & Modals** → **v1.5.15-v1.5.16 ✅**
  - **v1.5.15**: Toast notification system + Modal confirmations
    - Зеленые/красные уведомления снизу справа
    - Автоскрытие: 10s (успех), 60s (ошибка)
    - Global API: `window.toast`, `window.modal`
    - Полный рефакторинг Admin Panel
  - **v1.5.16**: Внедрение во все остальные страницы
    - Dashboard, Chat, Profile, Tenants, Usage, API Keys, MCP, About
    - ~10 alert() → toast, ~8 confirm() → modal
    - Единообразный UX на всех 8 страницах
  - **Оценка:** 2-3 часа (v1.5.15) + 1-2 часа (v1.5.16)
  - **Выполнено:** 100% (Complete notification system across all pages)

---

## 📊 Version 1.6.0 - Observability

**Фокус:** Observability и мониторинг production  
**Приоритет:** LOW  
**Оценка времени:** 22-28 часов  
**Версионирование:** 1 фича = 1 patch (1.6.1, 1.6.2, ...)

### Задачи

- [x] **OBSERV-01** OpenTelemetry Integration → [BACKLOG/OBSERV-01_opentelemetry.md](BACKLOG/OBSERV-01_opentelemetry.md) → **v1.6.1** ✅
  - Distributed tracing
  - Trace context propagation
  - Span annotations
  - Integration с Jaeger/Zipkin
  - **Оценка:** 10-12 часов

- [x] **OBSERV-02** Performance Monitoring → [BACKLOG/OBSERV-02_perf_monitoring.md](BACKLOG/OBSERV-02_perf_monitoring.md) → **v1.6.2** ✅
  - Continuous profiling в production
  - Memory leak detection
  - Slow request analysis
  - Performance regression detection
  - **Оценка:** 6-8 часов

- [x] **REPORT-01** Scheduled Reports → [BACKLOG/REPORT-01_scheduled_reports.md](BACKLOG/REPORT-01_scheduled_reports.md) → **v1.6.3** ✅
  - Автоматические отчеты по email
  - Usage statistics reports
  - Performance summaries
  - Configurable schedules
  - **Оценка:** 6-8 часов

---

## 📱 Version 1.7.0 - Smart Chat & Content (переименована из 1.8.0)

**Фокус:** Улучшенный чат и работа с контентом  
**Приоритет:** HIGH  
**Оценка времени:** 38-52 часа  
**Статус:** 📋 Запланировано (0/5 задач)

### Задачи

- [ ] **VISION-01** Vision OCR Service (Multimodal Chat) → [BACKLOG/VISION-01_ocr_multimodal_chat.md](BACKLOG/VISION-01_ocr_multimodal_chat.md) → **v1.7.1**
  - Multimodal models через Ollama (LLaVA, BakLLaVA)
  - Upload изображений в чат + drag&drop
  - Распознавание текста → контекст для LLM
  - Image storage & management
  - Thumbnail generation
  - **Оценка:** 8-12 часов

- [ ] **WEB-01** Web Content Fetcher & Summarization → [BACKLOG/WEB-01_content_fetcher.md](BACKLOG/WEB-01_content_fetcher.md) → **v1.7.2**
  - HTTP client + goquery для парсинга HTML
  - Извлечение текста, metadata
  - Summarization через LLM
  - URL validation и security
  - Rate limiting для web fetches
  - **Оценка:** 6-10 часов

- [ ] **FILE-01** File Upload Support (PDF, DOCX, TXT) → [BACKLOG/FILE-01_file_upload_support.md](BACKLOG/FILE-01_file_upload_support.md) → **v1.7.3**
  - PDF, DOCX, TXT text extraction
  - Интеграция с чатом (file → context)
  - File storage & management
  - Validation и size limits
  - **Оценка:** 8-10 часов

- [ ] **EXPORT-01** Conversation Export/Import → [BACKLOG/EXPORT-01_conversation_export_import.md](BACKLOG/EXPORT-01_conversation_export_import.md) → **v1.7.4**
  - Export to JSON, Markdown
  - Bulk export для всех conversations
  - Import для восстановления истории
  - Date range filtering
  - **Оценка:** 4-6 часов

- [ ] **WS-01** WebSocket Real-time Updates (WebUI only) → [BACKLOG/WS-01_websocket_realtime.md](BACKLOG/WS-01_websocket_realtime.md) → **v1.7.5**
  - Real-time updates вместо polling
  - Live chat streaming
  - System notifications push
  - Auto-reconnect logic
  - **Примечание:** TUI development остановлена, фокус на WebUI
  - **Оценка:** 8-12 часов

---

## 🔐 Version 1.8.0 - Enterprise Auth

**Фокус:** Корпоративная аутентификация  
**Приоритет:** HIGH  
**Оценка времени:** 58-78 часов  
**Статус:** 📋 Запланировано (0/5 задач)

### Задачи

- [ ] **OIDC-01** Keycloak SSO Integration → [BACKLOG/OIDC-01_keycloak_sso.md](BACKLOG/OIDC-01_keycloak_sso.md) → **v1.8.1**
  - OAuth2/OIDC flow
  - Token validation & refresh
  - UI для настройки через админку
  - User auto-provisioning
  - Fallback to local auth
  - **Оценка:** 16-24 часа

- [ ] **OIDC-02** Auto-tenant Provisioning from OIDC Groups → [BACKLOG/OIDC-02_auto_tenant_groups.md](BACKLOG/OIDC-02_auto_tenant_groups.md) → **v1.8.2**
  - Claims parsing (groups, roles)
  - Automatic tenant creation/mapping
  - Group sync (manual/scheduled)
  - Admin groups role assignment
  - **Оценка:** 8-12 часов

- [ ] **LDAP-01** LDAP/Active Directory Integration → [BACKLOG/LDAP-01_ldap_integration.md](BACKLOG/LDAP-01_ldap_integration.md) → **v1.8.3**
  - LDAP bind + search
  - User/group sync
  - UI для настройки через админку
  - TLS/LDAPS support
  - **Оценка:** 12-16 часов

- [ ] **AUDIT-01** Enhanced Audit Logging → [BACKLOG/AUDIT-01_enhanced_audit_logging.md](BACKLOG/AUDIT-01_enhanced_audit_logging.md) → **v1.8.4**
  - Security events (login, permissions)
  - Compliance logs
  - Query API с filters
  - Export to CSV
  - Retention policy
  - **Оценка:** 8-12 часов

- [ ] **RBAC-01** Custom Roles & Permissions → [BACKLOG/RBAC-01_custom_roles_permissions.md](BACKLOG/RBAC-01_custom_roles_permissions.md) → **v1.8.5**
  - Fine-grained permissions
  - Per-tenant role templates
  - Permission inheritance
  - RBAC middleware
  - Admin UI для role management
  - **Оценка:** 10-14 часов

---

## ⚙️ Version 1.9.0 - Model Management Pro

**Фокус:** Продвинутое управление моделями  
**Приоритет:** MEDIUM  
**Оценка времени:** 44-60 часов  
**Статус:** 📋 Запланировано (0/5 задач)

### Задачи

- [ ] **MODEL-01** Dynamic Model Parameters Configuration → [BACKLOG/MODEL-01_dynamic_parameters.md](BACKLOG/MODEL-01_dynamic_parameters.md) → **v1.9.1**
  - UI для настройки (temperature, top_p, num_ctx, etc.)
  - Per-tenant/per-user configs
  - Validation через Ollama API
  - Templates для быстрых настроек
  - **Оценка:** 8-12 часов

- [ ] **QUOTA-01** Usage Quotas System → [BACKLOG/QUOTA-01_usage_quotas.md](BACKLOG/QUOTA-01_usage_quotas.md) → **v1.9.2**
  - Per-tenant/per-user limits
  - Token-based quotas
  - Request quotas (daily, monthly)
  - Storage quotas
  - Quota monitoring & alerts
  - **Оценка:** 10-14 часов

- [ ] **MODEL-02** Model Preloading & Warming → [BACKLOG/MODEL-02_model_preloading.md](BACKLOG/MODEL-02_model_preloading.md) → **v1.9.3**
  - Keep models in memory
  - Configurable warm-up strategy
  - Health checks для loaded models
  - Automatic unload of unused models
  - **Оценка:** 6-8 часов

- [ ] **METRICS-01** Prometheus Metrics Export → [BACKLOG/METRICS-01_prometheus_export.md](BACKLOG/METRICS-01_prometheus_export.md) → **v1.9.4**
  - Metrics endpoint `/metrics`
  - Request counters, latency histograms
  - Model usage statistics
  - Quota metrics
  - Grafana dashboard template
  - **Оценка:** 4-6 часов

- [ ] **RATE-02** Advanced Rate Limiting → [BACKLOG/RATE-02_advanced_rate_limiting.md](BACKLOG/RATE-02_advanced_rate_limiting.md) → **v1.9.5**
  - Per tenant, per model limits
  - Sliding window algorithm
  - Rate limit headers в response
  - Burst allowance
  - **Оценка:** 8-10 часов

---

## 🔮 Future / Deferred

**Задачи для будущих версий** (без конкретного релиза)

### Scalability (отложено из 1.4.0)

- [ ] **SCALE-01** Redis Rate Limiting
  - Redis backend для distributed rate limiting
  - Multi-instance support
  - Shared state management
  - Fallback to in-memory если Redis недоступен

- [ ] **SCALE-04** Model Fallback Strategy
  - Automatic fallback если модель недоступна
  - Model compatibility mapping (gpt-4 → llama3, etc.)
  - Configurable fallback chains
  - Health-based model selection

### Advanced Features

- [ ] **OPS-02** Advanced Logging (отложено, после BUG-01)
  - Structured request/response logging
  - Automatic log rotation
  - Log aggregation support (syslog, fluentd)
  - Configurable log levels per component

- [ ] **TODO-06** Redis Token Blacklist
  - Миграция in-memory blacklist в Redis
  - Для distributed sessions
  - Файл: internal/auth/jwt/jwt.go

- [ ] **TODO-07** PostgreSQL Stubs Implementation
  - Заменить заглушки реальными реализациями
  - Файл: internal/storage/postgresql/stubs.go

- [ ] **TODO-08** Test Helpers Enhancement
  - JSON парсинг и проверка error.code
  - Детальное сравнение JSON объектов
  - HTTP requests с JSON body
  - Файл: internal/test/helpers.go

### Legacy Cleanup

- [ ] **TODO-09** Auth Middleware Cleanup
  - Удалить legacy middleware (уже реализовано в новых файлах)
  - Файл: internal/api/middleware/auth.go (можно удалить)

### ⏸️ Postponed Tasks (API Extensions - отложено)

**Причина отложения:** Низкий приоритет, зависимость от Ollama capabilities

- [ ] **API-08** Fine-tuning API Emulation → [BACKLOG/API-08_finetuning_emulation.md](BACKLOG/API-08_finetuning_emulation.md)
  - OpenAI `/v1/fine_tuning/jobs` API эмуляция
  - Job simulation и tracking  
  - Model aliasing (ft:model:suffix → model)
  - File upload API compatibility
  - **Оценка:** 8-10 часов
  - **Статус:** ⏸️ Отложено - низкий приоритет

- [ ] **API-09** Image Generation Support → [BACKLOG/API-09_image_generation.md](BACKLOG/API-09_image_generation.md)
  - OpenAI `/v1/images/generations` API
  - Support для image models (если Ollama добавит)
  - Image format conversion (PNG, JPEG, WebP, base64)
  - **Оценка:** 6-8 часов
  - **Статус:** ⏸️ BLOCKED - Ollama не поддерживает image generation

### Ideas for Discussion

- WebSocket для TUI (вместо HTTP polling)
- Plugin system для custom extensions
- GraphQL API как альтернатива REST
- Built-in proxy для других LLM providers (Anthropic, Cohere)
- Audit log для всех операций
- Advanced caching (Redis, Memcached)
- gRPC API для high-performance clients

---

## 📊 Краткая статистика Roadmap

| Version | Задач | Время (ч) | Приоритет | Статус | Версионирование |
|---------|-------|-----------|-----------|--------|-----------------|
| **1.2.0** | 4 | 20-30 | HIGH | ✅ **Завершено** | - |
| **1.3.0** | 4 | 45-57 | HIGH | ✅ **Завершено** | - |
| **1.4.0** ⭐ | 5 | 18-24 | **MEDIUM** | 🎯 **Текущая** | v1.4.1-1.4.5 |
| **1.5.0** | 5 | 11-16 | MEDIUM | 📋 Запланирована | v1.5.1-1.5.5 |
| **1.6.0** | 3 | 22-28 | LOW | 📋 Запланирована | v1.6.1-1.6.3 |
| **1.7.0** | 2 | 14-18 | LOW | 📋 Запланирована | v1.7.1-1.7.2 |
| **Future** | 9+ | TBD | DEFERRED | 📋 Отложено | - |
| **ИТОГО** | **32+** | **130-173** | - | - | - |

**Прогресс:** ✅ 2 версии завершены | 🎯 1 в разработке (MEDIUM) | 📋 3 запланированы

**Новый подход к версионированию:**  
🔄 1 фича = 1 patch версия (например: 1.4.0 → 1.4.1 → 1.4.2 → ... → 1.5.0)

---

## 🎯 Рекомендуемая последовательность

### ✅ Завершено (Version 1.2.0) - Мониторинг и управление

**Реализовано:**

- TUI Request Monitor (полный мониторинг запросов)
- WebUI Metrics Visualization (4 графика с Chart.js)
- Advanced Logs Features (фильтрация, поиск, экспорт)
- Enhanced API Key Management (edit, revoke, enable)

### ✅ Завершено (Version 1.3.0) - User Experience & Multi-Tenancy 🎉

**Реализовано:**

1. ✅ **DB-01** Database Abstraction Layer (SQLite + PostgreSQL)
2. ✅ **AUTH-05** User Authentication & Multi-Tenancy (JWT, RBAC)
3. ✅ **WEBUI-03** Interactive Chat Interface (ChatGPT-like UI) 💬
4. ✅ **WEBUI-04** User Dashboard (personal cabinet)
5. ✅ **BUILD-01** Cross-platform Build System (Linux, Windows, ARM64)

**Итого:** ~51 часов работы (в рамках оценки 45-57ч)

**Достигнутые Killer Features:**

- ✅ ChatGPT-like интерфейс для локальных моделей
- ✅ Полноценная user authentication (JWT + Bootstrap system)
- ✅ Multi-tenancy с командной работой
- ✅ Personal dashboard с аналитикой
- ✅ Granular API key management (personal + tenant)
- ✅ Profile management
- ✅ Tenants & members management
- ✅ Usage statistics monitoring
- ✅ Auth guard для защиты всех страниц

### 🎯 Текущая (Version 1.4.0) - WebUI Enhancements & Code Quality

**Последовательность реализации:**

1. **WEBUI-06** Model Copy Button → **v1.4.1** (быстрая задача, 1-2ч)
2. **TODO-01** TUI Admin API Key Config → **v1.4.2** (устранение hardcode, 2ч)
3. **TODO-02** Build Version Info → **v1.4.3** (реальная версия, 1-2ч)
4. **WEBUI-05** Enhanced Models Info → **v1.4.4** (расширенная информация, 8-10ч)
5. **WEBUI-07** MCP Catalog → **v1.4.5** (новая страница, 6-8ч)

**Итого:** ~18-24 часа работы

**Фокус:** Улучшение UX, устранение технического долга, расширение функционала админки

### Далее (Version 1.5.0) - Operations & Maintenance

**Последовательность реализации:**

1. ✅ **Enhanced Logs System** → **v1.5.1-v1.5.5** (COMPLETED!)
   - v1.5.1: Enhanced Logs System (основная реализация)
   - v1.5.2: SSE Authentication Fix
   - v1.5.3: Enhanced Log Parsing
   - v1.5.4: SSE Stream Auth Fix
   - v1.5.5: Logs Stream File Selection
2. ✅ **Modelfile Display Enhancement** → **v1.5.6** (COMPLETED!)
   - LICENSE удаление из Modelfile
   - 20 строк с прокруткой
3. ✅ **TODO-03** Password Change → **v1.5.7** (backend для UI, 3-4ч) **DONE!**
4. ✅ **TODO-04** Tenant Membership Check + Members → **v1.5.8** (проверка прав + UI, 5-6ч) **DONE!**
5. **TODO-05** Error Type Checking → **v1.5.9** (улучшение ошибок, 1-2ч)
6. **OPS-01** Backup & Restore → **v1.5.10** (автобекап, 3-4ч)

**Итого:** ~11-16 часов работы (4/6 завершено → 66% complete!)

**Фокус:** Исправление багов, завершение TODO из кода, операционная готовность

### Потом (Version 1.6.0) - Observability

1. **OBSERV-01** OpenTelemetry Integration → **v1.6.1** (10-12ч)
2. **OBSERV-02** Performance Monitoring → **v1.6.2** (6-8ч)
3. **REPORT-01** Scheduled Reports → **v1.6.3** (6-8ч)

**Итого:** ~22-28 часов работы

**Фокус:** Production monitoring, distributed tracing, reporting

### В будущем (Version 1.7.0) - API Extensions

1. **API-08** Fine-tuning API Emulation → **v1.7.1** (8-10ч)
2. **API-09** Image Generation Support → **v1.7.2** (6-8ч)

**Итого:** ~14-18 часов работы

**Фокус:** Расширение OpenAI API совместимости

---

## 📝 Критерии готовности для каждой версии

### Обязательно

- ✅ Все задачи version завершены
- ✅ Unit tests покрывают новый код (>90%)
- ✅ Integration tests проходят
- ✅ Документация обновлена
- ✅ Manual testing выполнен
- ✅ Performance не деградировал
- ✅ Backward compatibility сохранена

### Желательно

- 📋 Beta testing с реальными пользователями
- 📋 Security audit для критичных изменений
- 📋 Migration guide (если breaking changes)
- 📋 Release notes подготовлены

---

## 🔄 Процесс обновления Roadmap

1. **Ежемесячный review** - приоритизация на основе feedback
2. **Flexible approach** - можем переставлять задачи между версиями
3. **User-driven** - приоритет задачам по запросам пользователей
4. **Quality first** - не жертвуем качеством ради скорости

---

## 📞 Обратная связь

Если у вас есть предложения по Roadmap:

1. Создайте issue в репозитории
2. Опишите use case
3. Укажите приоритет
4. Добавим в следующий review

---

**Последнее обновление:** 2025-10-11  
**Версия Roadmap:** 1.5  
**Статус:** Active Development 🚀 (Version 1.5.0 ✅ Completed! → Planning v1.6.0-1.10.0)

**Изменения в v1.5:**

- 📱 **Version 1.8.0**: Smart Chat & Content (Vision OCR, Web Fetcher, File Upload, Export, WebSocket)
- 🔐 **Version 1.9.0**: Enterprise Auth (OIDC/Keycloak, LDAP, Audit, RBAC)
- ⚙️ **Version 1.10.0**: Model Management Pro (Dynamic Params, Quotas, Preloading, Prometheus, Rate Limiting)
- 📝 Создано 15 новых детальных BACKLOG файлов
- 🎯 Фокус на enterprise features и advanced model management
- 🚫 TUI development остановлена, фокус на WebUI
- 💡 Roadmap построен на основе user feedback и production needs
