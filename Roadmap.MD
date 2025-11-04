# 🗺️ Roadmap: AIGateway Platform

> **Текущая версия:** 2.4.6 ✅ | **В работе:** v2.4.7 (REFACTOR-01 🔄)  
> **Следующие версии:**  
>
> - 🤖 **v2.5.0** - Agentic AI & MCP Integration (HIGH Priority)  
> - 🎨 **v2.6.0** - HTMX Migration & TUI Removal (MEDIUM Priority) **← BIG Release!**  
> - 🔴 **v2.7.0** - Redis Integration & Sessions (MEDIUM Priority)  
> **План развития:** Version 2.0.0 - 2.7.0+  
> **Статус:** Active Development (9 versions ✅ Complete! 1 🔄 In Progress | Next: REFACTOR-01 → Agentic AI + HTMX + Redis)  
> **Последнее обновление:** 2025-10-30  
> **Новое:** 🔧 **Migration System Refactoring в v2.4.7!** Разбиваем sqlite.go (4560+ строк) на отдельные SQL файлы с embed.FS. После этого - 🗑️ **TUI Removal** в v2.6.0 (HTMX migration + полное удаление TUI). Codebase станет значительно чище!

---

## 📋 История завершенных версий

Для сокращения Roadmap завершенные версии v1.2.0-v1.5.0 перенесены в [CHANGELOG.md](CHANGELOG.md).

**Краткая сводка:**

- ✅ **v1.2.0** - Enhanced Monitoring & Management (4 задачи)
- ✅ **v1.3.0** - User Experience & Multi-Tenancy (5 задач)
- ✅ **v1.4.0** - WebUI Enhancements & Code Quality (5 задач)
- ✅ **v1.5.0** - Operations & Maintenance (12 задач)
- ✅ **v1.12.0** - User Experience & Performance (3 задачи)

---

## 📊 Version 1.6.0 - Observability ✅ **COMPLETED!**

**Фокус:** Observability и мониторинг production  
**Приоритет:** LOW → MEDIUM  
**Оценка времени:** 22-28 часов  
**Фактическое время:** ~24 часа  
**Статус:** 🎉 Завершено 100% (3/3 задачи)  
**Версионирование:** v1.6.1, v1.6.2, v1.6.3

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

## ⚙️ Version 1.9.0 - Model Management Pro ✅ **COMPLETED!**

**Фокус:** Продвинутое управление моделями + Performance Monitoring + CPU Cache Optimization  
**Приоритет:** MEDIUM → HIGH  
**Оценка времени:** 20-28 часов (3 из 7 задач) + 16 часов (Performance Optimization)  
**Фактическое время:** ~24 часа + ~18 часов (Performance)  
**Статус:** 🎉 Завершено 100% (4/4 выполненных задач + BONUS Performance!)  
**Версионирование:** v1.9.1, v1.9.2, v1.9.3, **v1.9.4** ✨

### Задачи

- [x] **MODEL-01** Dynamic Model Parameters Configuration → [BACKLOG/MODEL-01_dynamic_parameters.md](BACKLOG/MODEL-01_dynamic_parameters.md) → **v1.9.1** ✅
  - **Chat UI - красивая панель над строкой ввода** (Primary UX)
  - Model selector с localStorage persistence
  - Collapsible parameters panel (sliders + presets)
  - Per-tenant/per-user configs (Backend готов, Admin UI опционально)
  - Параметры основаны на официальном Ollama API
  - Quick presets (Creative, Balanced, Precise, Coding)
  - **Оценка:** 10-14 часов → **Выполнено**

- [x] **UI-01** Compact UI & Context Manager → **v1.9.2** ✅
  - Компактный дизайн панели моделей (Cursor-style)
  - Context Window tracking с визуальными индикаторами
  - Автоматическая суммаризация при заполнении контекста
  - Real-time оценка токенов
  - **Оценка:** 4-6 часов → **Выполнено**

- [x] **PERF-01** Performance Monitoring with MoniGo + GPU Monitoring → [BACKLOG/PERF-01_monigo_integration.md](BACKLOG/PERF-01_monigo_integration.md) → **v1.9.3** ✅
  - Интеграция библиотеки MoniGo для real-time мониторинга
  - NVIDIA GPU Monitoring через nvidia-smi (БЕЗ CGO!)
  - Реорганизация Admin Panel (Models, Logs, System отдельно)
  - Performance dashboard на отдельном порту (9091)
  - Quick Stats Cards с автообновлением
  - Unified GPU card дизайн для multi-GPU
  - **Оценка:** 6-8 часов → **Выполнено**

- [x] **CACHE-OPT-01** CPU Cache-Friendly Data Structures → **v1.9.4** ✅ 🚀
  - **Phase 1**: False sharing elimination (StatsOptimized с padding)
    - GlobalStats migration → cache line padding
    - APIKey race condition fix → atomic operations
    - Benchmarks: 6.4x speedup на Stats
  - **Phase 2**: Hot/Cold Split & Caching
    - APIKeyCache Layer (Ristretto-based, 22ns Get, 100x faster vs DB)
    - APIKeyDBAuthOptimized middleware (95%+ cache hit rate)
    - APIKeyUsageThreadSafe (atomic + mutex, 20% faster + thread-safe)
  - **Documentation**: Comprehensive cache optimization guide
    - `docs/CACHE_OPTIMIZATION.md` (Technical deep-dive)
    - `docs/CACHE_OPTIMIZATION_SUMMARY.md` (Executive summary)
    - `PERFORMANCE_IMPROVEMENTS.md` (High-level report)
    - `MIGRATION_APPLIED.md` (Phase status)
    - `PHASE2_COMPLETED.md` (Detailed results)
  - **Cursor Rules**: Automated cache-friendly linting
  - **Performance Impact**:
    - Authentication: 20x faster (с cache)
    - Stats tracking: 6.4x faster (parallel)
    - Usage tracking: 1.2x faster + thread-safe
    - Server throughput: +50-87% improvement
    - Memory overhead: ~6 MB для 10k keys (acceptable)
  - **Оценка:** 16-20 часов → **Выполнено (~18 часов)** ✨

### 🎯 Достижения Version 1.9.0

- ✅ Dynamic model parameters в Chat UI
- ✅ Compact UI дизайн (Cursor-style)
- ✅ Context window tracking + auto-summarization
- ✅ MoniGo Performance Dashboard
- ✅ NVIDIA GPU monitoring (multi-GPU support)
- ✅ Admin Panel reorganization
- ✅ localStorage persistence для всех настроек
- ✅ **CPU Cache Optimization** (v1.9.4) 🚀
  - Cache line padding (StatsOptimized)
  - In-memory API key cache (Ristretto)
  - Thread-safe usage tracking
  - Comprehensive documentation
  - **Performance**: 6.4x-100x speedup!

**Оставшиеся задачи перенесены:**

- METRICS-01, QUOTA-01 → **v1.11.0 (Enterprise Suite)** - для полной observability
- MODEL-02, RATE-02 → **v1.12.0 (UX & Performance)** - оптимизация производительности

---

## 📱 Version 1.10.0 - Content Foundation (RAG Prerequisites)

**Фокус:** Foundation layer для работы с контентом (подготовка к RAG v1.13)  
**Приоритет:** HIGH  
**Оценка времени:** 32-44 часа  
**Фактическое время:** ~38 часов (все 4 задачи завершены!)  
**Статус:** 🎉 ЗАВЕРШЕНО 100% (4/4 задач)  
**Версионирование:** v1.10.0-v1.10.4

**Стратегия:** Реализуем базовые компоненты работы с файлами, изображениями и веб-контентом с заделом на RAG систему (v1.13.0). Все extractors и storage будут переиспользованы в RAG.

### Задачи

- [x] **FILE-STORAGE-01** Universal File Storage & Document Processing → [BACKLOG/FILE-STORAGE-01_universal_storage.md](BACKLOG/FILE-STORAGE-01_universal_storage.md) → **v1.10.1**
  - **Universal Storage Layer** (Local FS + S3/MinIO) 🔄 *Будет использован в RAG*
  - **Document Extractors** (PDF, DOCX, TXT, CSV) 🔄 *Основа для RAG processing*
  - File metadata management в PostgreSQL
  - Upload API + validation (size, type)
  - Basic integration с чатом (загрузка → extraction → контекст)
  - **Примечание:** Архитектура спроектирована для RAG (v1.13)
  - **Оценка:** 10-14 часов

- [x] **WS-01** WebSocket Real-time Updates → [BACKLOG/WS-01_websocket_realtime.md](BACKLOG/WS-01_websocket_realtime.md) → **v1.10.2** ✅
  - Real-time updates вместо polling ✅
  - Live chat streaming ✅
  - System notifications push ✅
  - Auto-reconnect logic ✅
  - Progress updates для file processing ✅
  - **Примечание:** Перенесена раньше для поддержки async operations
  - **Оценка:** 8-12 часов → **Выполнено (~6 часов)**

- [x] **IMAGE-01** Image Upload & OCR Processing → [BACKLOG/IMAGE-01_image_processing.md](BACKLOG/IMAGE-01_image_processing.md) → **v1.10.3** ✅
  - ✅ Vision Interface (OCREngine, OCROptions, OCRResult)
  - ✅ OllamaOCR implementation (LLaVA, BakLLaVA, Llama3.2-Vision)
  - ✅ ImageProcessor (thumbnail, resize, format conversion)
  - ✅ ImageHandler API (upload, download, thumbnail endpoints)
  - ✅ WebSocket integration для upload/OCR progress
  - ✅ Unit tests + benchmarks (all passing)
  - 📋 WebUI drag & drop (deferred to future)
  - **Примечание:** OCR engine готов для расширения в RAG v1.13.0
  - **Оценка:** 8-10 часов → **Выполнено (~8 часов)**

- [x] **WEB-FETCH-01** Web Content Fetcher → [BACKLOG/WEB-FETCH-01_content_fetcher.md](BACKLOG/WEB-FETCH-01_content_fetcher.md) → **v1.10.4** ✅
  - ✅ HTTP client + HTML parser (goquery) 🔄 *Будет web source в RAG*
  - ✅ Text extraction и metadata (Open Graph, Twitter Card)
  - ✅ URL validation и SSRF security checks
  - ✅ Rate limiting per domain
  - ✅ **Chat Integration** - автодетектирование URLs в сообщениях
  - ✅ **Auto-fetch** - автоматическое извлечение контента для LLM
  - ✅ Context enrichment для streaming и non-streaming режимов
  - ✅ Database migration v31 (web_fetches, web_fetch_rate_limits)
  - ✅ Unit tests + benchmarks (all passing)
  - **Примечание:** Будет основой для web data sources в RAG
  - **Оценка:** 6-8 часов → **Выполнено (~8 часов)**

### 🎯 Связь с v1.13.0 (RAG System)

Компоненты из v1.10.0 которые будут переиспользованы в RAG:

| v1.10.0 Компонент | Использование в v1.13.0 RAG |
|-------------------|----------------------------|
| **FILE-STORAGE-01**: Storage Layer | → `internal/rag/storage/` (Local + S3) |
| **FILE-STORAGE-01**: Extractors | → `internal/rag/extractors/` (PDF, DOCX, CSV) |
| **IMAGE-01**: OCR Engine | → `internal/rag/extractors/image.go` |
| **WEB-FETCH-01**: HTML Parser | → `internal/rag/sources/web.go` |
| **WS-01**: Progress Updates | → Real-time RAG job status |

**Преимущества:**

- ✅ Избегаем дублирования кода
- ✅ v1.10.0 дает immediate value (чат с файлами)
- ✅ v1.13.0 расширяет существующую инфраструктуру
- ✅ Постепенная миграция (можно начать с v1.10, потом добавить RAG)

---

## 🔐 Version 1.11.0 - Enterprise Suite

**Фокус:** Полноценная корпоративная платформа (Auth + Observability + Quotas)  
**Приоритет:** HIGH  
**Оценка времени:** 88-120 часов  
**Статус:** ✅ Завершено (8/8 задач)  
**Versioning:** v1.11.1-v1.11.8

**Стратегия:** Объединяем корпоративную аутентификацию, мониторинг и контроль ресурсов в единый enterprise-ready релиз.

### Задачи

- [x] **OIDC-01** Keycloak SSO Integration → [BACKLOG/OIDC-01_keycloak_sso.md](BACKLOG/OIDC-01_keycloak_sso.md) → **v1.11.1** ✅
  - OAuth2/OIDC flow
  - Token validation & refresh
  - UI для настройки через админку
  - User auto-provisioning
  - Fallback to local auth
  - **Оценка:** 16-24 часа → **Выполнено**

- [x] **OIDC-02** Auto-tenant Provisioning from OIDC Groups → [BACKLOG/OIDC-02_auto_tenant_groups.md](BACKLOG/OIDC-02_auto_tenant_groups.md) → **v1.11.2** ✅
  - Claims parsing (groups, roles)
  - Automatic tenant creation/mapping (direct + prefix modes)
  - Group sync on login
  - Admin groups role assignment
  - Orphaned memberships cleanup
  - **Оценка:** 8-12 часов → **Выполнено**

- [x] **LDAP-01** LDAP/Active Directory Integration → [BACKLOG/LDAP-01_ldap_integration.md](BACKLOG/LDAP-01_ldap_integration.md) → **v1.11.3** ✅
  - LDAP bind authentication (OpenLDAP, Active Directory, FreeIPA)
  - User/Group search с настраиваемыми фильтрами
  - Auto-provisioning users при первом логине
  - Tenant provisioning из LDAP groups (reuse OIDC-02)
  - TLS/LDAPS support с StartTLS и certificate validation
  - Admin detection на основе LDAP groups
  - **Оценка:** 12-16 часов → **Выполнено**

- [x] **AUDIT-01** Enhanced Audit Logging → [BACKLOG/AUDIT-01_enhanced_audit_logging.md](BACKLOG/AUDIT-01_enhanced_audit_logging.md) → **v1.11.4** ✅
  - Security events (login, permissions)
  - Compliance logs (24 event types)
  - Query API с filters + pagination
  - Export to CSV
  - Retention policy (90 days default, daily cleanup)
  - Statistics dashboard (24h metrics)
  - Admin UI (preview + dedicated page)
  - **Оценка:** 8-12 часов → **Выполнено**

- [x] **RBAC-01** Custom Roles & Permissions → [BACKLOG/RBAC-01_custom_roles_permissions.md](BACKLOG/RBAC-01_custom_roles_permissions.md) → **v1.11.5** ✅
  - Fine-grained permissions (23+ system permissions)
  - System + Custom roles (global + tenant-scoped)
  - Wildcard permission matching (*:*, api_keys:*,*:create)
  - RBAC middleware (RequirePermission, RequireAnyPermission, RequireAllPermissions, RequireRole)
  - Admin UI для role management (dedicated RBAC page)
  - **Оценка:** 10-15 часов → **Выполнено за ~12 часов**

- [x] **METRICS-01** Prometheus Metrics Export → [BACKLOG/METRICS-01_prometheus_export.md](BACKLOG/METRICS-01_prometheus_export.md) → **v1.11.6** ✅
  - Metrics endpoint `/metrics` (configurable)
  - HTTP metrics (request rate, duration histograms, response size, active connections)
  - API usage metrics (tokens by api_key/model, requests by model/status)
  - Model metrics (latency histograms, errors, loaded models count)
  - System metrics (goroutines, memory, DB connections, entities counts)
  - Auth metrics (attempts/duration by type: jwt/oidc/ldap/api_key)
  - Quota metrics (future-ready: usage/limits/exceeded)
  - Grafana dashboard template with 11 panels
  - Complete setup documentation (Prometheus/Grafana/Docker Compose)
  - **Оценка:** 4-6 часов → **Выполнено за ~5 часов**

- [x] **QUOTA-01** Usage Quotas System → [BACKLOG/QUOTA-01_usage_quotas.md](BACKLOG/QUOTA-01_usage_quotas.md) → **v1.11.7** ✅
  - Per-user и per-tenant quota definitions (scope + target_id)
  - Token quotas (daily/monthly limits с automatic tracking)
  - Request quotas (daily/monthly + concurrent limiting)
  - Storage quotas (max_file_size, max_storage_bytes, max_conversations) - future-ready
  - Model restrictions (allowed_models list)
  - Auto-reset logic (daily: 24h, monthly: calendar month)
  - Quota Service (CheckQuota, RecordUsage, auto-reset)
  - Quota Middleware (429 Too Many Requests при exceed)
  - Prometheus integration (usage/limit/exceeded metrics)
  - Admin API (CRUD для quotas, usage tracking)
  - User API (GET /api/quota/me для own stats)
  - Database migration v43 (quotas, quota_usage tables)
  - **Оценка:** 10-14 часов → **Выполнено за ~12 часов**

- [x] **UI-SYNC-01** WebUI Functionality Restoration → **v1.11.8** ✅
  - **Задача:** Синхронизация функционала между /web_old/ (старый) и /web/ (новый дизайн с theme.css)
  - **Цель:** Восстановить весь функционал при сохранении нового дизайна
  - **Стратегия:** Page-by-page анализ и портирование функциональности
  - **Оценка:** 16-20 часов → **Выполнено**

  **Детализация по страницам:**

  - [x] **1. chat.html** (8/10 приоритет) - 3-4 часа ✅
    - HTML идентичен в обеих версиях
    - JS файлы (chat.js, model-panel.js, context-manager.js, app.js) идентичны
    - Добавлен theme.css в head
    - Проверить context-manager.js (отслеживание контекста)
    - Проверить streaming responses
    - Проверить интеграцию с web-fetch (auto URL extraction)
    - Применить theme.css стили к chat interface
    - Проверить сохранение/загрузку conversations

  - [x] **2. dashboard.html** (9/10 приоритет) - 2 часа ✅
    - Восстановлена полная структура из web_old/
    - JS файл (dashboard.js) идентичен
    - Добавлен theme.css и navigation-placeholder

  - [x] **3. admin.html** (10/10 приоритет - КРИТИЧНО) - 4-5 часов ✅
    - User Management: действия (enable/disable/reset password/delete)
    - API Keys Management: отображение owner_username, статус, действия
    - Models: accordion для детальной информации
    - RBAC Tab: полная интеграция (roles, permissions, user roles)
    - Audit Tab: полная интеграция (events, filters, stats, pagination)
    - System Tab: changelogs rendering
    - Logs Tab: real-time log streaming

  - [x] **4. api-keys.html** (7/10 приоритет) - 1 час ✅
    - Восстановлена структура из web_old/ (Personal/Tenant tabs, modals)
    - JS файл (apikeys.js) идентичен
    - Добавлен theme.css и navigation-placeholder

  - [x] **5. files.html** (6/10 приоритет) - 1-2 часа ✅
    - Восстановлена структура из web_old/ (drag-drop, filters, grid view)
    - JS файл (files.js) идентичен
    - Добавлен theme.css и navigation-placeholder

  - [x] **6. tenants.html** (7/10 приоритет) - 1-2 часа ✅
    - Восстановлена структура из web_old/ (grid, modals, members)
    - JS файл (tenants.js) идентичен
    - Добавлен theme.css и navigation-placeholder

  - [x] **7. profile.html** (5/10 приоритет) - 1 час ✅
    - Восстановлена структура из web_old/
    - JS файл (profile.js) идентичен
    - Добавлен theme.css и navigation-placeholder

  - [x] **8. usage.html** (6/10 приоритет) - 1 час ✅
    - Файл уже был идентичен web_old/
    - Добавлен theme.css и navigation-placeholder

  - [x] **9. about.html** (3/10 приоритет) - 0.5 часа ✅
    - HTML идентичен web_old/
    - JS файл (about.js) идентичен
    - Добавлен theme.css

  - [x] **10. mcp.html** (4/10 приоритет) - 0.5 часа ✅
    - HTML идентичен web_old/
    - JS файл (mcp.js) идентичен
    - Добавлен theme.css

  - [x] **11. admin-audit.html** (Standalone) - 0.5 часа ✅
    - Интегрировано в admin.html как таб
    - Standalone версия не требуется

  - [x] **12. admin-rbac.html** (Standalone) - 0.5 часа ✅
    - Интегрировано в admin.html как таб
    - Standalone версия не требуется

  - [x] **13. JS Components** - 1 час ✅
    - components/navbar.js - используется на всех страницах
    - Все JS файлы идентичны между web/ и web_old/

  **Чек-лист для каждой страницы:**
  - [x] Функциональность восстановлена (все действия работают)
  - [x] theme.css применен (градиенты, карточки, кнопки)
  - [x] Responsive design работает (mobile/tablet/desktop)
  - [x] Console errors исправлены
  - [x] API endpoints корректны
  - [x] Element IDs совпадают между HTML и JS

  **Итоговый результат:**
  - ✅ Новый дизайн (theme.css) на всех страницах
  - ✅ Весь функционал из /web_old/ портирован
  - ✅ Consistency между страницами (navigation, стили, UX)
  - ✅ Zero regression - все работает как раньше + лучше выглядит
  - ✅ Основные страницы синхронизированы (chat, dashboard, admin, api-keys, files, tenants, profile, usage)

### 🎯 Enterprise Features Complete

После v1.11.0 платформа будет иметь:

✅ **Authentication & Authorization:**

- SSO (Keycloak/OIDC)
- LDAP/Active Directory
- Custom RBAC
- Auto-provisioning

✅ **Observability & Compliance:**

- Audit logging
- Prometheus metrics
- Security events tracking
- Export capabilities

✅ **Resource Management:**

- Token quotas
- Request rate limits
- Storage quotas
- Quota alerts

---

## ⚡ Version 1.12.0 - User Experience & Performance ✅ **COMPLETED!**

**Фокус:** Улучшение UX и производительности системы  
**Приоритет:** MEDIUM  
**Оценка времени:** 18-28 часов  
**Фактическое время:** ~20 часов  
**Статус:** 🎉 Завершено 100% (3/3 задач)  
**Versioning:** v1.12.1, v1.12.2, v1.12.3

**Стратегия:** Компактный релиз фокусирующийся на user experience, экспорте данных и оптимизации производительности.

### Задачи

- [x] **MODEL-02** Model Preloading & Warming → [BACKLOG/MODEL-02_model_preloading.md](BACKLOG/MODEL-02_model_preloading.md) → **v1.12.1** ✅
  - Keep models in memory (preload on startup)
  - Configurable warm-up strategy (health check loop)
  - Health checks для loaded models (periodic dummy requests)
  - Automatic unload of unused models (timeout-based)
  - Priority-based preloading (usage tracking)
  - Admin API endpoints (GET /loaded, POST /preload)
  - **Оценка:** 6-8 часов → **Выполнено**

- [x] **RATE-02** Advanced Rate Limiting → [BACKLOG/RATE-02_advanced_rate_limiting.md](BACKLOG/RATE-02_advanced_rate_limiting.md) → **v1.12.2** ✅
  - Per tenant, per model, per user, per API key limits (multi-scope)
  - Sliding window algorithm (more accurate than fixed window)
  - Rate limit headers в response (RFC 6585 compliance)
  - In-memory cache для performance (<1ms overhead)
  - Database schema для flexible configuration
  - **Оценка:** 8-10 часов → **Выполнено**

- [x] **EXPORT-01** Conversation Export/Import → [BACKLOG/EXPORT-01_conversation_export_import.md](BACKLOG/EXPORT-01_conversation_export_import.md) → **v1.12.3** ✅
  - Export to JSON, Markdown, Text formats
  - Bulk export для всех conversations
  - Import для восстановления истории (с опциями merge/preserve)
  - Access control и validation
  - Metadata export (tokens, dates, model used)
  - **Примечание:** Перенесена из v1.10 для логической группировки с UX
  - **Оценка:** 4-6 часов → **Выполнено**

### 🎯 Performance & UX Improvements

После v1.12.0:

✅ **Performance:**

- Model preloading для faster responses
- Advanced rate limiting для stability
- Optimized resource usage

✅ **User Experience:**

- Conversation export/import
- Better data portability
- History backup & restore

---

## 🧠 Version 1.13.0 → 2.0.0 - RAG System ✅ **COMPLETED!**

**Фокус:** Полноценная RAG (Retrieval-Augmented Generation) система + Chat Export/Import UI  
**Приоритет:** HIGH  
**Оценка времени:** 54-72 часа (экономия 6-8ч благодаря v1.10.0)  
**Фактическое время:** ~68 часов  
**Статус:** 🎉 Завершено 100% (5/5 задач + Chat Export/Import UI)  
**Версионирование:** v1.13.1-v1.13.5 → **v2.0.0 MAJOR RELEASE**  
**Dependencies:** ✅ Использовано v1.10.0 (FILE-STORAGE-01, IMAGE-01, WEB-FETCH-01)

**Стратегия:** Расширяем существующие компоненты из v1.10.0 добавляя векторный поиск, chunking, embeddings и orchestration. Переиспользуем file storage, extractors и web fetcher.

### Задачи

- [x] **RAG-01** Foundation & Data Sources Management → [BACKLOG/RAG-01_system_architecture.md](BACKLOG/RAG-01_system_architecture.md) → **v2.0.0** ✅
  - ✅ Database schema (sources, documents, chunks, jobs)
  - ✅ Data sources CRUD API (7 endpoints)
  - ✅ ~~File storage (Local + S3/MinIO)~~ *Переиспользуем из v1.10.0*
  - ✅ Basic in-memory queue
  - ✅ Server integration & routes
  - 📋 **WebUI: Data Sources Management Page** (осталось)
    - Grid view со списком sources
    - Create/Edit/Delete modals
    - Source type selector (file/api/database/web)
    - Configuration forms per source type
    - Status indicators & sync controls
    - Tags & search filtering
  - **Оценка:** 12-16 часов (backend ✅ done, WebUI 📋 pending ~4-6 часов)

- [x] **RAG-02** Document Processing Pipeline → **v2.0.0** ✅
  - ✅ ~~PDF/DOCX/CSV extractors~~ *Переиспользуем из v1.10.0*
  - ✅ Semantic chunking engine
  - ✅ PostgreSQL job queue
  - ✅ Worker pool (async processing)
  - **WebUI: Progress Tracking Dashboard**
    - Real-time job status (WebSocket из v1.10.0)
    - Processing queue visualization
    - Document processing logs
    - Chunk preview & statistics
    - Error handling & retry controls
  - **Оценка:** 12-16 часов (меньше т.к. extractors готовы)

- [x] **RAG-03** Embeddings & Vector Search → **v2.0.0** ✅
  - ✅ Ollama embeddings integration (nomic-embed-text, mxbai-embed-large)
  - ✅ pgvector setup + HNSW indexing
  - ✅ Batch embedding processing
  - ✅ Vector search API
  - **WebUI: Search Testing Interface**
    - Query input с similarity threshold slider
    - Top-K results selector
    - Search results preview (chunks + metadata)
    - Similarity score visualization
    - Source filtering controls
    - Export search results
  - **Оценка:** 12-16 часов

- [x] **RAG-04** External Data Sources → **v2.0.0** ✅
  - ✅ REST API source implementation
  - ✅ PostgreSQL database source
  - ✅ Credentials encryption (AES-256)
  - ✅ Connection testing
  - ✅ Scheduled auto-sync
  - **Оценка:** 10-14 часов

- [x] **RAG-05** RAG Integration & Chat UI → **v2.0.0** ✅
  - ✅ RAG Orchestrator
  - ✅ Reranking logic
  - ✅ Context assembly
  - ✅ Chat API integration
  - **WebUI: RAG Chat Integration**
    - ✅ Source selector в chat UI
    - ✅ Multi-select sources для query
    - ✅ RAG toggle switch (enable/disable per conversation)
    - ✅ Top-K chunks и Min Score sliders
    - ✅ Reranking checkbox
  - **Оценка:** 8-10 часов

- [x] **BONUS** Chat Export/Import UI → **v2.0.0** ✅ **(ADDED)**
  - ✅ Export dropdown в chat header
  - ✅ Export as JSON/Markdown/Text
  - ✅ Import modal с file upload
  - ✅ Preserve timestamps/IDs options
  - ✅ JSON validation перед import
  - ✅ Auto-reload после успешного import
  - **Оценка:** 4-6 часов (BONUS feature from v1.12.3 backend)

### 🎯 Ключевые возможности

**Backend:**

- 📄 Multi-format файлы (PDF, DOCX, CSV, TXT, изображения)
- 🔌 Внешние API с Bearer token аутентификацией
- 🗄️ PostgreSQL data sources с параметризованными запросами
- 🧠 Semantic chunking с учетом контекста (512 tokens, 10% overlap)
- 🔍 Vector search через pgvector (HNSW indexing)
- ⚡ Async processing с worker pool
- 🔐 AES-256 encryption для credentials

**WebUI:**

- 🎨 **Data Sources Page** - полное управление источниками (create/edit/delete/sync)
- 📊 **Processing Dashboard** - real-time мониторинг обработки документов
- 🔬 **Search Testing** - интерактивное тестирование vector search
- 💬 **Chat Integration** - source selector + RAG toggle в чате
- 📈 **Analytics** - статистика использования RAG (queries, tokens, sources)

### 🏗️ Технический стек

- **Vector DB:** pgvector (PostgreSQL extension) или Qdrant
- **Embeddings:** Ollama (nomic-embed-text 768d или bge-m3 1024d)
- **File Storage:** Local FS или MinIO/S3
- **Queue:** PostgreSQL или in-memory
- **Extractors:** PDF, DOCX, CSV parsers
- **Chunking:** Semantic + Fixed-size hybrid

### 📈 Use Cases

1. **CSV Analysis:** "У меня в CSV есть клиенты с балансом >10000₽, дай список"
2. **PDF Knowledge Base:** Загрузка документации → вопросы по содержимому
3. **API Integration:** Реал-тайм данные из внешних систем
4. **Database Queries:** Прямые запросы к корп. БД с контекстом
5. **Multi-source:** Комбинирование данных из разных источников

---

## 🏷️ Version 2.1.0 - AIGateway Rebranding

**Фокус:** Ребрендинг проекта в **AIGateway Platform**  
**Приоритет:** MEDIUM  
**Оценка времени:** 8-12 часов  
**Статус:** 📋 Запланировано (0/4 задач)  
**Версионирование:** v2.1.0

**Стратегия:** Комплексное переименование проекта для отражения расширенных возможностей beyond простого Ollama proxy. Подготовка к multi-model architecture.

### Задачи

- [x] **REBRAND-01** Documentation & README → **v2.1.0**
  - 📝 README.md - новое описание AIGateway Platform
  - 📝 Architecture.MD - обновление архитектурных диаграмм
  - 📝 Contributing guide - обновление названия
  - 📝 Все ссылки на "AIGateway Platform" → "AIGateway"
  - **Оценка:** 2-3 часа

- [x] **REBRAND-02** Code & Configuration → **v2.1.0**
  - 🔧 go.mod - module path rename
  - 🔧 Package imports по всему проекту
  - 🔧 Config files (dev.yaml, production.yaml)
  - 🔧 Docker Compose & Kubernetes yamls
  - 🔧 Environment variables (AIGATEWAY_*)
  - **Оценка:** 3-4 часа

- [x] **REBRAND-03** WebUI Branding → **v2.1.0**
  - 🎨 Logo & favicon (AIGateway branding)
  - 🎨 Page titles & headers
  - 🎨 Navigation labels
  - 🎨 About System page
  - 🎨 Footer copyright
  - **Оценка:** 2-3 часа

- [x] **REBRAND-04** GitHub & Release → **v2.1.0**
  - 📦 GitHub repository rename
  - 📦 Docker Hub organization/image rename
  - 📦 Release notes для v2.1.0
  - 📦 Migration guide для existing users
  - 📦 Update badges в README
  - **Оценка:** 1-2 часа

### 🎯 Цели ребрендинга

1. **Позиционирование:** Не просто proxy, а полноценная AI infrastructure platform
2. **Масштабируемость:** Название позволяет добавлять новые LLM providers
3. **Enterprise Appeal:** Более профессиональное название для корпоративных клиентов
4. **Community:** Привлечение более широкой аудитории разработчиков

### 📋 Breaking Changes

- ⚠️ **Module path:** `github.com/user/aigateway` → `github.com/user/aigateway`
- ⚠️ **Environment variables:** `OLLAMA_PROXY_*` → `AIGATEWAY_*`
- ⚠️ **Docker image:** `ollama-proxy:latest` → `aigateway:latest`
- ℹ️ **Migration guide** будет предоставлен для smooth transition

---

## 🔐 Version 2.2.0 - Invitation-Only Registration System

**Фокус:** Контролируемая регистрация пользователей через систему приглашений  
**Приоритет:** HIGH  
**Оценка времени:** 6-8 часов  
**Статус:** ✅ Завершено 100% (1/1 задача)  
**Версионирование:** v2.2.0  
**Dependencies:** None (независимая функциональность)  
**Завершено:** 2025-10-28

**Стратегия:** Реализация enterprise-grade системы управления доступом через пригласительные ссылки. Админы контролируют кто может зарегистрироваться в системе.

### Задачи

- [x] **AUTH-03** Invitation-Only Registration System → [BACKLOG/AUTH-03_invitation_system.md](BACKLOG/AUTH-03_invitation_system.md) → **v2.2.0** ✅ **ЗАВЕРШЕНО**
  - **Backend API:**
    - POST /api/admin/invitations - создание пригласительной ссылки
    - GET /api/admin/invitations - список всех приглашений (активные/использованные/отозванные)
    - DELETE /api/admin/invitations/:id - отзыв пригласительной ссылки
    - GET /api/invitations/:token/validate - проверка валидности токена приглашения
    - POST /api/auth/register - регистрация с опциональным токеном приглашения
  - **Database Schema:**
    - Таблица invitations: id, token (UUID), created_by_user_id, email (optional), expires_at, used_at, used_by_user_id, revoked_at, max_uses, current_uses
    - Индексы: token (unique), created_by_user_id, status
  - **Configuration:**
    - `auth.registration.mode`: "open" | "invitation_only" | "disabled"
    - `auth.registration.require_email_verification`: true/false
    - `auth.invitations.default_expiry_days`: 7
    - `auth.invitations.max_uses_default`: 1
  - **Business Logic:**
    - Админ генерирует уникальную ссылку с токеном (UUID v4)
    - Опционально: привязка к конкретному email
    - Опционально: срок действия (expires_at)
    - Single-use по умолчанию (max_uses = 1)
    - После успешной регистрации токен становится использованным
    - Админ может отозвать неиспользованную ссылку
  - **WebUI Components:**
    - Admin Panel → "Ссылки-приглашения" (новая вкладка `/admin-invitations.html`)
    - Таблица приглашений с фильтрами (Active/Used/Expired/Revoked)
    - Кнопка "Create Invitation" с формой (email опционально, expiry date)
    - Copy invitation link to clipboard
    - Revoke button для активных приглашений
    - Statistics: total created, used, expired, revoked
    - Register page: проверка токена перед показом формы
  - **Security:**
    - Rate limiting на создание приглашений (защита от спама)
    - Token должен быть cryptographically secure (UUID v4)
    - Validation токена перед регистрацией
    - Audit log для всех операций с приглашениями
  - **Оценка:** 6-8 часов

### 🎯 Ключевые возможности

**Backend:**

- 🔐 **Registration Modes** - гибкое переключение между открытой и закрытой регистрацией
- 🎫 **Invitation Management** - полный CRUD для пригласительных ссылок
- 📧 **Email Restriction** - опциональная привязка приглашения к email
- ⏰ **Expiration Control** - настраиваемый срок действия приглашений
- 🔄 **Multi-use Support** - возможность создавать многоразовые приглашения
- 🚫 **Revocation** - отзыв неиспользованных приглашений
- 📊 **Analytics** - статистика по использованию приглашений

**WebUI:**

- 📋 **Invitations Dashboard** - управление всеми приглашениями
- ➕ **Quick Create** - быстрое создание приглашения с copy-to-clipboard
- 🔍 **Filters & Search** - поиск по статусу, email, дате
- 📈 **Statistics Cards** - visual metrics (success rate, active invitations)
- ✅ **Registration Flow** - валидация токена перед показом формы регистрации

### 🏗️ Configuration Example

```yaml
auth:
  registration:
    mode: "invitation_only"  # "open" | "invitation_only" | "disabled"
    require_email_verification: false
    
  invitations:
    enabled: true
    default_expiry_days: 7    # Default expiry for new invitations
    max_uses_default: 1       # Default max uses (single-use)
    allow_email_restriction: true
    
    # Rate limits
    rate_limit:
      creation_per_admin_hour: 10
      validation_per_ip_minute: 10
```

### 📊 Use Cases

1. **Enterprise Deployment:** Закрытая регистрация для корпоративных пользователей
2. **Beta Testing:** Контролируемое приглашение тестировщиков
3. **Email-Restricted:** Приглашение только конкретного email адреса
4. **Time-Limited:** Приглашения с коротким сроком действия (24ч)
5. **Multi-Use Links:** Одно приглашение для команды (max_uses > 1)

---

## 🎯 Version 2.3.0 - Model Registry & Multi-Provider Foundation

**Фокус:** Universal Model Registry + vLLM Integration  
**Приоритет:** MEDIUM  
**Оценка времени:** 20-28 часов  
**Статус:** ✅ Завершена (3/4 задач, VLLM-01 отложена)  
**Версионирование:** v2.3.1-v2.3.4  
**Dependencies:** ⚠️ Требует v2.1.0 (Rebranding complete)

**Стратегия:** Создание универсальной архитектуры для поддержки множественных model providers. Foundation для HuggingFace, OpenAI, Anthropic integration.

### Задачи

- [x] **REGISTRY-01** Model Registry Core → [BACKLOG/REGISTRY-01_model_registry.md](BACKLOG/REGISTRY-01_model_registry.md) → **v2.3.1** ✅
  - ✅ Model metadata storage (DB schema: model_providers, model_registry)
  - ✅ Model capabilities tracking (chat/embeddings/vision/function-calling)
  - ✅ Provider abstraction layer (Provider interface, OllamaProvider, vLLMProvider stub)
  - ✅ Model registration API (POST /api/admin/registry/models)
  - ✅ Model discovery endpoint (GET /api/admin/registry/models)
  - ✅ Provider management API (CRUD для providers)
  - ✅ Auto-discovery mechanism (DiscoverModels при старте)
  - ✅ Health check integration (HealthCheckAll providers)
  - ✅ Configuration (model_registry в dev.yaml)
  - ✅ Seed data (default Ollama provider)
  - **Фактически:** 8 часов
  - **Завершено:** 2025-10-28

- [x] **REGISTRY-02** Smart Model Router → [BACKLOG/REGISTRY-02_smart_router.md](BACKLOG/REGISTRY-02_smart_router.md) → **v2.3.2** ✅
  - ✅ Router middleware для model requests (ModelRouterMiddleware)
  - ✅ Provider selection logic (automatic routing by model_id)
  - ✅ Fallback chains (automatic failover при provider down)
  - ✅ Load balancing (latency + load-based selection)
  - ✅ Request tracing (X-Provider-* headers + structured logs)
  - ✅ Capabilities matching при fallback
  - ✅ Health checks перед маршрутизацией
  - ✅ Model-Provider caching (5min TTL)
  - **Фактически:** 5 часов
  - **Завершено:** 2025-10-28

- [ ] **VLLM-01** vLLM Backend Integration → [BACKLOG/VLLM-01_integration.md](BACKLOG/VLLM-01_integration.md) → **v2.3.3**
  - vLLM client implementation
  - OpenAI-compatible API mapping
  - Multi-GPU support configuration
  - Model loading/unloading API
  - Performance benchmarking
  - Docker Compose integration
  - **Оценка:** 6-8 часов
  - **Documentation:** vLLM installation guide для Linux 2x GPU

- [x] **REGISTRY-03** Model Management WebUI → [BACKLOG/REGISTRY-03_webui.md](BACKLOG/REGISTRY-03_webui.md) → **v2.3.4** ✅
  - ✅ Model Registry dashboard (admin-registry.html)
  - ✅ Provider status cards с real-time health indicators
  - ✅ Model registration forms (modals для providers и models)
  - ✅ Model health monitoring (auto-refresh каждые 30 сек)
  - ✅ Performance metrics display (latency, throughput, total requests)
  - ✅ Model discovery UI (кнопка + progress)
  - ✅ Filters & Search (provider, status, health, keyword)
  - ✅ Statistics cards (total models, active, providers, healthy)
  - ✅ Integration с admin panel (вкладка Model Registry)
  - **Фактически:** 3 часа
  - **Завершено:** 2025-10-28

### 🎯 Ключевые возможности

**Backend:**

- 🗄️ **Universal Registry** - централизованное управление всеми моделями
- 🔀 **Smart Routing** - автоматический выбор оптимального provider'а
- 🚀 **vLLM Support** - fast inference для HuggingFace моделей
- ⚡ **Multi-GPU** - распределение моделей на несколько GPU
- 🔄 **Fallback Chains** - automatic failover между providers
- 📊 **Provider Metrics** - performance tracking per provider

**WebUI:**

- 📋 **Model Registry Page** - unified view всех доступных моделей
- 🎛️ **Provider Dashboard** - мониторинг Ollama, vLLM статусов
- ⚙️ **Configuration** - model assignment к providers
- 📈 **Performance Charts** - latency, throughput per provider

### 🏗️ Технический стек

- **Model Providers:**
  - Ollama (existing) - local models, GGUF format
  - vLLM (new) - HuggingFace models, GPU inference
  - Future: TGI, LocalAI, OpenAI, Anthropic

- **Architecture:**
  - Provider abstraction interface
  - Model registry database (SQLite/PostgreSQL)
  - Smart router middleware
  - Health check system

### 📈 Use Cases

1. **HuggingFace Models:** Запуск моделей которые не поддерживаются Ollama
2. **GPU Optimization:** Использование vLLM для fast inference
3. **Model Fallback:** Автоматический переход на backup provider
4. **Multi-Model:** Разные модели на разных providers (Ollama + vLLM)
5. **Load Balancing:** Распределение нагрузки между GPU/providers

---

## 🖥️ Version 2.4.0 - Desktop Client Support

**Фокус:** Server-side поддержка для Desktop приложения (Wails)  
**Приоритет:** MEDIUM → HIGH  
**Оценка времени:** 12-16 часов + 3-4 часа (REFACTOR-01)  
**Фактическое время:** ~14 часов  
**Статус:** 🔄 **IN PROGRESS** (4/5 задач - осталась REFACTOR-01)  
**Версионирование:** v2.4.1-v2.4.7  
**Dependencies:** v2.2.0+ (API Keys system must be working)  
**Related:** [Roadmap-desktop.md](Roadmap-desktop.md) - Desktop app development plan  
**Обновлено:** 2025-10-30

**Стратегия:** Подготовка server-side инфраструктуры для поддержки тонкого desktop клиента на Wails. Фокус на auto-generated API keys, device management, и WebSocket streaming.

### Задачи

- [x] **DESKTOP-01** Auto-Generated API Keys System → [BACKLOG/DESKTOP-01_auto_api_keys.md](BACKLOG/DESKTOP-01_auto_api_keys.md) → **v2.4.1** ✅
  - Endpoint для auto-creation API keys: POST /api/auth/devices/register
  - Device metadata в API keys (os, hostname, app_version)
  - Device naming convention: "Desktop App - {OS} - {Date}"
  - Automatic expiry для device keys (опционально, 90 days)
  - Device fingerprinting для duplicate detection
  - Last seen timestamp update на каждый request (prepared for future integration)
  - **Оценка:** 3-4 часа → **Completed: 2025-10-28**

- [x] **DESKTOP-02** Device Management API → [BACKLOG/DESKTOP-02_device_management.md](BACKLOG/DESKTOP-02_device_management.md) → **v2.4.2** ✅
  - GET /api/auth/devices - список всех устройств пользователя
  - GET /api/auth/devices/:id - информация об устройстве
  - DELETE /api/auth/devices/:id - revoke device key
  - PATCH /api/auth/devices/:id - переименование устройства
  - WebUI страница /profile-devices.html для device management
  - Self-revoke prevention и owner verification
  - **Оценка:** 3-4 часа → **Completed: 2025-10-28**

- [x] **DESKTOP-03** WebSocket Streaming для Desktop → [BACKLOG/DESKTOP-03_websocket_streaming.md](BACKLOG/DESKTOP-03_websocket_streaming.md) → **v2.4.3** ✅
  - WebSocket endpoint: /ws/chat?token={api_key} с API key authentication
  - API key validation через bcrypt (existing keys)
  - Device tracking: last_seen_at updates при WebSocket activity
  - Chat request/response message types (chat_request, chat_chunk, chat_done, chat_error)
  - Streaming chat responses через WebSocket с Ollama integration
  - Ping/Pong для keep-alive (уже реализовано в v1.10.2)
  - Per-user message routing (Hub.SendToUser)
  - Completed: 2025-10-28

- [x] **DESKTOP-04** Desktop-Specific WebUI Features → [BACKLOG/DESKTOP-04_webui_devices.md](BACKLOG/DESKTOP-04_webui_devices.md) → **v2.4.4** ✅
  - Device details modal (клик на card/icon) с full device info
  - Bulk revoke для multiple devices (checkbox selection + bulk actions bar)
  - Enhanced UI/UX (View Details button, smooth animations, protection от self-revoke)
  - Visual indicators (status badges, last seen color coding)
  - Responsive modal design (max-width: 700px, max-height: 90vh)
  - **Completed:** 2025-10-28

- [x] **REFACTOR-01** Migration System Refactoring + Rollback → [BACKLOG/REFACTOR-01_migrations.md](BACKLOG/REFACTOR-01_migrations.md) → **v2.4.7** ✅
  - **Проблема:** `sqlite.go` огромен (4560+ строк) и постоянно растет с каждой версией
  - **Решение:** Разбить на отдельные SQL файлы используя Go 1.16+ `embed.FS` + добавить rollback систему
  - **Реализовано:**
    - ✅ SQLite: 142 миграционных файла (71 up + 71 down) в `internal/storage/sqlite/migrations/`
    - ✅ PostgreSQL: 142 миграционных файла (71 up + 71 down) в `internal/storage/postgresql/migrations/`
    - ✅ Автоматический loader через embed.FS с сортировкой по версиям
    - ✅ Все CRUD методы портированы в PostgreSQL (12 файлов)
    - ✅ Конвертация placeholders (? → $1, $2...)
    - ✅ Успешная компиляция обоих пакетов
  - **Структура:**
    - `internal/storage/{sqlite,postgresql}/migrations/` - директории с SQL файлами
    - Именование: `001_initial_schema.up.sql` (forward) + `001_initial_schema.down.sql` (rollback)
    - Loader в `migrations.go` с автоматическим чтением up/down из embedded FS
    - Rollback метод для отката миграций до любой версии
  - **Преимущества:**
    - ✅ Легче добавлять новые миграции (просто создать up/down файлы)
    - ✅ Легче ревьюить в Git (one file per migration)
    - ✅ Можно использовать SQL форматтеры/linters
    - ✅ Уменьшился размер `sqlite.go` с 4560 до ~300 строк
    - ✅ **Rollback support** - возможность отката миграций
    - ✅ **PostgreSQL полностью реализован** - production-ready!
  - **Completed:** 2025-11-03

### 🎯 Ключевые возможности

**Authentication:**

- 🔑 **One-time Password Flow** - Username/Password → JWT → API Key → Discard JWT
- 🖥️ **Device Recognition** - Каждый desktop получает unique API key
- 🔒 **Granular Control** - Revoke access для конкретного устройства
- 📊 **Activity Tracking** - Last seen, IP address, usage metrics
- 🔄 **Auto-renewal** - Опционально, продление ключей для активных devices

**Streaming:**

- ⚡ **WebSocket Support** - Real-time chat streaming для desktop
- 🔄 **Fallback to SSE** - Server-Sent Events если WebSocket недоступен
- 📡 **Binary Protocol** - Эффективная передача данных
- 🔌 **Auto-reconnect** - Graceful handling connection drops

**Management:**

- 📱 **Multi-Device Support** - Пользователь может иметь 10+ devices
- 🗑️ **Easy Revocation** - One-click device removal из web UI
- 🔔 **Notifications** - Уведомления о новых устройствах
- 📈 **Usage Analytics** - Статистика по устройствам

### 🏗️ Технический стек

**Backend:**

- Existing API Keys system (расширение)
- WebSocket library (gorilla/websocket)
- Device fingerprinting (user-agent parsing)
- GeoIP для location tracking (опционально)

**Database:**

Расширение таблицы `api_keys`:

```sql
ALTER TABLE api_keys ADD COLUMN device_info JSONB;
ALTER TABLE api_keys ADD COLUMN last_seen_at TIMESTAMP;
ALTER TABLE api_keys ADD COLUMN last_ip VARCHAR(45);
ALTER TABLE api_keys ADD COLUMN auto_generated BOOLEAN DEFAULT false;
```

**WebUI:**

- Devices Management page (React/Svelte)
- Device cards с status indicators
- Activity timeline component
- Revoke confirmation modal

### 📈 Use Cases

1. **Desktop App Login:** User вводит credentials один раз → автоматически создается API key
2. **Multi-Device:** User работает с home PC + office PC + laptop - каждое устройство имеет свой key
3. **Security Incident:** User видит неизвестное устройство → revokes его key немедленно
4. **Device Lost:** Laptop украден → revoke ключа из web UI блокирует access
5. **Usage Tracking:** Admin видит что device не использовался 90 days → auto-revoke

### 🔗 Desktop Client Integration

После реализации v2.4.0, desktop client (см. [Roadmap-desktop.md](Roadmap-desktop.md)) сможет:

- ✅ Использовать POST /api/auth/devices/register для получения API key
- ✅ Подключаться к /ws/chat для streaming
- ✅ Показывать список всех user devices
- ✅ Self-revoke при logout

---

## 🤖 Version 2.5.0 - Agentic AI & MCP Integration

**Фокус:** Server-side Agent API для поддержки autonomous AI в Desktop Client  
**Приоритет:** HIGH (Required for Desktop v0.8.0)  
**Оценка времени:** 28-35 часов  
**Статус:** 📋 Запланировано (0/5 задач)  
**Версионирование:** v2.5.1-v2.5.5  
**Dependencies:**

- v2.4.3+ (WebSocket infrastructure must be working)
- Desktop Client v0.5.0+ (Terminal integration)
- Desktop Client v0.8.0 (Agentic Mode)

**Related:** [Roadmap-desktop.md](Roadmap-desktop.md) Version 0.8.0 - Agentic Mode

**Стратегия:** Реализация server-side инфраструктуры для autonomous AI agent capabilities. AI сможет планировать задачи, использовать tools (file operations, terminal, MCP servers) и требовать approval для опасных операций.

### 🧠 Vision: Autonomous AI Agent

Desktop client получит возможность использовать AI как autonomous agent:

```
User: "Refactor this module and add tests"
         ↓
AI Agent (server-side):
  1. Анализирует код
  2. Планирует задачи (5 steps)
  3. Использует tools:
     - file.read() → читает файлы
     - file.write() → пишет новый код
     - terminal.execute() → запускает тесты
     - mcp.github → создает PR (опционально)
  4. Требует approval для file.write(), terminal.execute()
  5. Применяет изменения после approval
         ↓
User: Reviews diff → Approves → Changes applied
```

### Задачи

- [ ] **AGENT-01** Agent Planning & Orchestration API → **v2.5.1**
  - **Planning Endpoint:**
    - POST /api/agent/plan - Task decomposition
    - Request: `{task: "Refactor module", context: {...}}`
    - Response: `{plan: [{step: 1, action: "file.read", ...}], estimated_tools: 5}`
  - **Agent State Management:**
    - In-memory agent sessions (sessionID → AgentState)
    - Session lifecycle (create, update, complete, cancel)
    - Task tree structure (parent → subtasks)
  - **Task Decomposition Logic:**
    - Analyze user request
    - Break into atomic steps
    - Determine required tools per step
    - Generate execution plan
  - **Models:**
    - AgentSession, AgentPlan, AgentTask, AgentStep
  - **Integration:**
    - Use existing Ollama client for planning (use strongest model)
    - Structured output для consistent planning
  - **Оценка:** 6-8 часов

- [ ] **AGENT-02** Tool Execution Framework → **v2.5.2**
  - **Tool Registry:**
    - Central registry для all available tools
    - Tool categories: filesystem, terminal, mcp, network
    - Tool metadata (name, description, parameters schema)
    - Tool capabilities advertisement
  - **Tool Execution Endpoint:**
    - POST /api/agent/tools/execute
    - Request: `{tool: "file.read", parameters: {path: "..."}}`
    - Response: `{success: true, result: {...}, execution_time_ms: 50}`
  - **Built-in Tools Implementation:**
    - `file.read(path)` - read file content
    - `file.write(path, content)` - write/overwrite file
    - `file.create(path, content)` - create new file (fail if exists)
    - `file.delete(path)` - delete file
    - `file.list(directory, recursive)` - list files
    - `file.search(pattern, directory)` - search files by pattern
  - **Tool Result Caching:**
    - Cache read operations (TTL 60s)
    - Cache list operations (TTL 30s)
  - **Tool Security:**
    - Path validation (prevent ../../../etc/passwd)
    - File size limits (max 10 MB for read)
    - Rate limiting per tool (100 calls/min)
  - **Models:**
    - Tool, ToolExecution, ToolResult, ToolError
  - **Оценка:** 8-10 часов

- [ ] **AGENT-03** Terminal Tool Integration → **v2.5.3**
  - **Terminal Execution Tool:**
    - `terminal.execute(command, cwd, timeout)`
    - Spawn process with timeout (default 60s)
    - Capture stdout, stderr, exit code
    - Stream output in real-time (через WebSocket)
  - **Security Features:**
    - **Dangerous Command Detection:**
      - Patterns: `rm -rf`, `sudo`, `format`, `dd`, `mkfs`, `> /dev/sda`
      - Network downloads: `curl | bash`, `wget | sh`
      - Git operations: `git push`, `git commit`
      - Требуют explicit approval
    - **Command Whitelist:**
      - Safe commands: `ls`, `cat`, `echo`, `pwd`, `which`
      - Auto-approve без user interaction
    - **Sandbox Environment (optional):**
      - Restrict to project directory
      - No access to system directories
  - **Process Management:**
    - Track running processes (pid → ProcessInfo)
    - Kill process on timeout
    - Kill process on user cancel
  - **Output Streaming:**
    - WebSocket channel для real-time output
    - Buffered output (max 100 KB)
    - Truncate if output exceeds limit
  - **Shell Selection:**
    - Support bash, powershell, zsh
    - Auto-detect based on OS
    - User-configurable в settings
  - **Models:**
    - TerminalExecution, ProcessInfo, CommandOutput
  - **Оценка:** 6-8 часов

- [ ] **AGENT-04** MCP Server Integration & Tool Catalog → **v2.5.4**
  - **MCP Catalog API:**
    - GET /api/mcp/catalog - list all available MCP servers
    - Response: `[{id: "github-mcp", name: "GitHub", tools: [...], status: "enabled"}]`
    - Filter by category (filesystem, web, database, etc.)
    - Search MCP servers
  - **MCP Tool Discovery:**
    - Query MCP servers для available tools
    - Parse tool schemas (input/output parameters)
    - Cache tool schemas (TTL 1 hour)
  - **MCP Tool Invocation:**
    - POST /api/mcp/invoke
    - Request: `{mcp_server: "github-mcp", tool: "create_issue", parameters: {...}}`
    - Response: `{success: true, result: {...}, mcp_server_response: {...}}`
  - **MCP Server Management:**
    - Enable/Disable MCP servers per user/tenant
    - Rate limiting per MCP server (100 calls/min)
    - Tool execution logs (audit trail)
  - **MCP Configuration:**
    - Store MCP configs (API keys, base URLs) encrypted
    - Per-user/per-tenant MCP settings
    - Sync from existing MCP registry (v1.11.0+)
  - **Security:**
    - Whitelist only MCP servers from our catalog
    - Validate tool parameters against schema
    - Dangerous tool warnings (delete operations, etc.)
  - **Integration:**
    - Use existing MCP infrastructure (if available)
    - Or implement minimal MCP client protocol
  - **Models:**
    - MCPServer, MCPTool, MCPToolInvocation, MCPToolResult
  - **Оценка:** 6-8 часов

- [ ] **AGENT-05** Approval Flow & Rollback System → **v2.5.5**
  - **Approval Flow API:**
    - POST /api/agent/approval/request
    - Request: `{session_id: "...", tool: "file.write", parameters: {...}, reason: "..."}`
    - Response: `{approval_id: "...", status: "pending", timeout_at: "..."}`
  - **Approval Management:**
    - GET /api/agent/approval/:id - check approval status
    - POST /api/agent/approval/:id/approve - approve action
    - POST /api/agent/approval/:id/deny - deny action
    - Approval timeout (30 seconds → auto-deny)
  - **Approval Rules:**
    - **RED (require approval):**
      - File: delete, overwrite existing files
      - Terminal: sudo, rm -rf, dangerous commands
      - Git: push, commit
      - Network: curl, wget
    - **YELLOW (auto-approve with logging):**
      - File: create new files, read
      - Terminal: ls, cat, echo, pwd
      - MCP: query operations (whitelisted)
    - **GREEN (always allowed):**
      - File: read
      - Terminal: safe commands (ls, pwd, echo)
  - **Rollback System:**
    - **File Snapshots:**
      - Snapshot file before modification (git-like)
      - Store snapshots in `/tmp/agent-snapshots/{session_id}/{file_hash}`
      - TTL 24 hours
    - **Rollback API:**
      - POST /api/agent/rollback
      - Request: `{session_id: "...", steps: ["step_3", "step_5"]}`
      - Response: `{success: true, restored_files: [...], restored_count: 2}`
    - **Undo Stack:**
      - Track all modifications per session
      - Store reverse operations (write → restore snapshot)
      - Max 50 operations per session
  - **Approval History:**
    - Store all approval requests (approved/denied)
    - Audit trail для security review
    - GET /api/agent/approval/history - view history
  - **Models:**
    - ApprovalRequest, ApprovalRule, FileSnapshot, RollbackOperation
  - **Оценка:** 8-10 часов

### 🎯 Ключевые возможности

**Agent Planning:**

- 🧠 **Task Decomposition** - разбивает сложные задачи на шаги
- 📊 **Progress Tracking** - отслеживает выполнение каждого шага
- 🔄 **Adaptive Planning** - корректирует план на основе результатов
- 🎯 **Goal-oriented** - фокусируется на достижении цели

**Tool Execution:**

- 📁 **File Operations** - read, write, create, delete, search
- 💻 **Terminal Commands** - execute, stream output, kill processes
- 🔌 **MCP Integration** - use external tools from catalog
- 🔒 **Security First** - approval flow для dangerous operations

**Approval Flow:**

- ✅ **User Control** - user подтверждает опасные действия
- ⏱️ **Timeout Protection** - auto-deny если no response in 30s
- 📝 **Audit Trail** - все действия логируются
- ↩️ **Rollback** - откат изменений одним кликом

**Safety & Security:**

- 🚫 **Dangerous Command Detection** - блокирует `rm -rf`, `sudo`, etc.
- 🛡️ **Path Validation** - prevent directory traversal attacks
- 📊 **Rate Limiting** - max 100 tools/min per session
- 🔐 **MCP Whitelist** - only trusted MCP servers

### 🏗️ Технический стек

**Backend:**

```go
internal/
  agent/
    planner.go          // Task planning & decomposition
    orchestrator.go     // Agent session management
    tools/
      registry.go       // Tool registry
      filesystem.go     // File operations
      terminal.go       // Terminal execution
      mcp.go           // MCP integration
    approval/
      manager.go        // Approval flow
      rules.go         // Approval rules
    rollback/
      snapshots.go      // File snapshots
      undo.go          // Undo operations
  models/
    agent.go           // Agent models
    tools.go           // Tool models
    approval.go        // Approval models
```

**API Endpoints:**

```
POST   /api/agent/plan                    # Task planning
POST   /api/agent/tools/execute           # Execute tool
GET    /api/mcp/catalog                   # List MCP servers
POST   /api/mcp/invoke                    # Invoke MCP tool
POST   /api/agent/approval/request        # Request approval
POST   /api/agent/approval/:id/approve    # Approve action
POST   /api/agent/approval/:id/deny       # Deny action
POST   /api/agent/rollback                # Rollback changes
GET    /api/agent/approval/history        # Approval history
```

**WebSocket Messages (для Desktop Client):**

```json
// Agent → Client
{
  "type": "agent_thinking",
  "session_id": "...",
  "payload": {
    "current_step": 2,
    "total_steps": 5,
    "message": "Reading file src/main.go..."
  }
}

{
  "type": "agent_tool_use",
  "session_id": "...",
  "payload": {
    "tool": "file.read",
    "parameters": {"path": "src/main.go"}
  }
}

{
  "type": "agent_approval_needed",
  "session_id": "...",
  "payload": {
    "approval_id": "...",
    "tool": "file.write",
    "reason": "Overwriting existing file",
    "diff": "+45 lines, -12 lines",
    "timeout_at": "2025-10-29T12:00:30Z"
  }
}

{
  "type": "agent_completed",
  "session_id": "...",
  "payload": {
    "success": true,
    "tools_used": 8,
    "files_modified": 3,
    "summary": "Refactoring completed successfully"
  }
}
```

**Database Schema:**

```sql
-- Agent sessions
CREATE TABLE agent_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    task TEXT NOT NULL,
    plan JSONB,
    status VARCHAR(50) DEFAULT 'active',
    tools_used INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Tool executions
CREATE TABLE tool_executions (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    result JSONB,
    success BOOLEAN,
    execution_time_ms INTEGER,
    executed_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE
);

-- Approval requests
CREATE TABLE approval_requests (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    reason TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    approved_by UUID REFERENCES users(id),
    timeout_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE
);

-- File snapshots (for rollback)
CREATE TABLE file_snapshots (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES agent_sessions(id),
    file_path TEXT NOT NULL,
    content_hash VARCHAR(64),
    snapshot_path TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE
);

-- MCP tool invocations
CREATE TABLE mcp_tool_invocations (
    id UUID PRIMARY KEY,
    session_id UUID REFERENCES agent_sessions(id),
    user_id UUID NOT NULL REFERENCES users(id),
    mcp_server VARCHAR(100) NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    parameters JSONB,
    result JSONB,
    success BOOLEAN,
    execution_time_ms INTEGER,
    invoked_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (session_id) REFERENCES agent_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX idx_agent_sessions_user ON agent_sessions(user_id);
CREATE INDEX idx_agent_sessions_status ON agent_sessions(status);
CREATE INDEX idx_tool_executions_session ON tool_executions(session_id);
CREATE INDEX idx_approval_requests_session ON approval_requests(session_id);
CREATE INDEX idx_approval_requests_status ON approval_requests(status);
CREATE INDEX idx_file_snapshots_session ON file_snapshots(session_id);
CREATE INDEX idx_mcp_tool_invocations_session ON mcp_tool_invocations(session_id);
CREATE INDEX idx_mcp_tool_invocations_user ON mcp_tool_invocations(user_id);
```

### 📈 Use Cases

**1. Code Refactoring:**

```
User: "Refactor the UserService to use dependency injection"
Agent:
  1. Reads UserService.go (file.read)
  2. Analyzes dependencies
  3. Generates new code with DI
  4. Requests approval для file.write
  5. User reviews diff → Approves
  6. Applies changes
  7. Runs tests (terminal.execute go test)
  8. Reports success
```

**2. Bug Fix Automation:**

```
User: "Fix the panic in handleRequest function"
Agent:
  1. Reads stack trace
  2. Identifies problematic code (file.read)
  3. Proposes fix
  4. Requests approval
  5. Applies fix (file.write)
  6. Adds test case (file.create)
  7. Runs tests
  8. Commits changes (git.commit - requires approval)
```

**3. Project Setup:**

```
User: "Initialize a new Go project with tests"
Agent:
  1. Creates directory structure (file.create)
  2. Writes main.go (file.write)
  3. Writes go.mod (file.create)
  4. Creates test files (file.create)
  5. Runs go mod tidy (terminal.execute)
  6. Runs tests (terminal.execute)
  7. Reports success
```

**4. MCP Integration:**

```
User: "Create GitHub issue for this bug"
Agent:
  1. Analyzes bug details
  2. Uses MCP GitHub tool (mcp.invoke github.create_issue)
  3. Creates issue with description
  4. Returns issue URL
```

### 🔗 Desktop Client Integration

После реализации v2.5.0, desktop client v0.8.0 сможет:

- 🤖 **Enable Agentic Mode** - toggle в Settings
- 📋 **View Task Plans** - tree view с progress
- ✅ **Approve/Deny Tools** - approval UI с diff viewer
- 📊 **Track Agent Progress** - real-time updates
- ↩️ **Rollback Changes** - one-click undo
- 🔌 **Use MCP Tools** - browse catalog, enable/disable tools
- 📝 **View Agent History** - audit trail всех actions

### ⚠️ Риски и митigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Agent executes malicious command** | Critical | Approval flow + command whitelist |
| **Agent infinite loop** | High | Max 50 tools per session, timeout 5 min |
| **File corruption** | Medium | Snapshots before modification, rollback |
| **MCP server compromise** | High | Whitelist only our catalog, rate limiting |
| **Approval timeout abuse** | Medium | 30s timeout → auto-deny |
| **Resource exhaustion** | Medium | Rate limiting, max concurrent sessions |

### 📊 Acceptance Criteria

- ✅ Agent может планировать tasks из 5+ steps
- ✅ File operations работают (read, write, delete)
- ✅ Terminal commands выполняются safely
- ✅ MCP tools доступны через catalog
- ✅ Approval flow требует confirmation для dangerous ops
- ✅ Rollback восстанавливает files correcty
- ✅ WebSocket messages отправляются в real-time
- ✅ Rate limiting работает (100 tools/min)
- ✅ Database migrations применяются без ошибок
- ✅ All endpoints have proper authentication
- ✅ Security: path validation prevents directory traversal
- ✅ Security: dangerous commands detected and blocked

### 🔄 Зависимости

**Requires from Desktop Client:**

- Desktop v0.5.0+ (Terminal integration для local command execution)
- Desktop v0.8.0 (Agent UI, approval dialogs, progress tracking)

**Provides for Desktop Client:**

- Agent API endpoints для task planning
- Tool execution infrastructure
- Approval flow system
- MCP integration layer
- WebSocket messages для real-time updates

---

## 🎨 Version 2.6.0 - HTMX Migration & TUI Removal (WebUI Modernization)

**Фокус:** Миграция WebUI на HTMX + удаление устаревшего TUI  
**Приоритет:** MEDIUM  
**Оценка времени:** 23-31 час  
**Статус:** 📋 Запланировано (0/4 задач)  
**Версионирование:** v2.6.1-v2.6.4  
**Dependencies:** None (независимая модернизация + cleanup)

**Стратегия:**

1. Миграция WebUI страниц с Vanilla JavaScript на HTMX (CRUD операции)
2. Полное удаление TUI (Terminal UI) - больше не используется
3. Desktop client остается на Svelte (не трогаем)
4. Chat page остается на Vanilla JS (сложный state)

**Why this is a BIG release:**

- 🎨 HTMX Migration - современный WebUI
- 🗑️ TUI Removal - ~2000+ lines legacy code удалено
- 📦 12 BACKLOG files cleanup
- 🧹 Codebase становится чище и проще поддерживать

### 🎯 Vision: Simplified WebUI with HTMX

**Текущая проблема:**

- ~3000+ lines JavaScript для CRUD операций
- Manual DOM manipulation
- Дублирование логики (fetch → render → update)
- Сложная синхронизация state

**После HTMX:**

- ~80% меньше JavaScript кода
- Declarative HTML attributes
- Server-side HTML rendering
- Simplified maintenance

**Пример трансформации:**

```javascript
// ❌ БЫЛО: 50+ lines JavaScript
async loadModels() {
    const response = await api.get('/api/admin/registry/models');
    this.renderModelsTable(response.models);
}

renderModelsTable(models) {
    const tbody = document.getElementById('models-tbody');
    tbody.innerHTML = models.map(m => `
        <tr><td>${m.name}</td>...</tr>
    `).join('');
}
```

```html
<!-- ✅ СТАЛО: 1 HTML attribute -->
<div id="models-table" 
     hx-get="/api/ui/registry/models" 
     hx-trigger="load">
    Loading...
</div>
```

### Задачи

- [ ] **HTMX-01** HTMX Core Setup & CRUD Pages Migration → **v2.6.1**
  
  **Подзадачи:**
  
  **1. HTMX Infrastructure Setup (2-3 часа)**
  - Добавить htmx.min.js (14KB gzipped) в /web/js/
  - Создать Go HTML templates infrastructure:
    - `web/templates/layouts/base.html` - base layout
    - `web/templates/partials/` - HTML fragments для HTMX
    - `web/templates/components/` - reusable UI blocks
  - Создать UI handlers package: `internal/api/handlers/ui/`
  - Helper functions для template rendering
  
  **2. Model Registry Page Migration (3-4 часа)**
  - **Current:** `web/admin-registry.html` + `web/js/admin-registry.js` (400+ lines)
  - **Target:** HTMX-based with server-rendered HTML
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/registry_ui_handler.go
  GET  /api/ui/registry/models          → HTML table rows
  GET  /api/ui/registry/models/search   → HTML filtered results
  GET  /api/ui/registry/models/new-form → HTML modal form
  GET  /api/ui/registry/models/:id/edit → HTML edit form
  POST /api/ui/registry/providers/new   → HTML provider form
  ```
  
  **Frontend Changes:**

  ```html
  <!-- Models Table -->
  <div id="models-table" 
       hx-get="/api/ui/registry/models" 
       hx-trigger="load"
       hx-swap="innerHTML">
      <div class="loading">Loading models...</div>
  </div>
  
  <!-- Search Input -->
  <input type="search" name="q"
         hx-get="/api/ui/registry/models/search" 
         hx-trigger="keyup changed delay:300ms"
         hx-target="#models-table"
         placeholder="Search models...">
  
  <!-- Add Model Button -->
  <button hx-get="/api/ui/registry/models/new-form"
          hx-target="#modal-content"
          class="btn btn-primary">
      ➕ Add Model
  </button>
  
  <!-- Model Row (server-rendered) -->
  <tr>
      <td>{{ .Name }}</td>
      <td>{{ .Provider }}</td>
      <td>
          <button hx-get="/api/ui/registry/models/{{ .ID }}/edit"
                  hx-target="#modal-content">
              ✏️ Edit
          </button>
          <button hx-delete="/api/admin/registry/models/{{ .ID }}"
                  hx-target="closest tr"
                  hx-swap="outerHTML swap:1s"
                  hx-confirm="Delete model {{ .Name }}?">
              🗑️ Delete
          </button>
      </td>
  </tr>
  ```
  
  **Templates:**
  - `web/templates/partials/registry/models_table.html`
  - `web/templates/partials/registry/model_form.html`
  - `web/templates/partials/registry/provider_form.html`
  
  **Экономия:** ~300 lines JS → ~50 lines HTML
  
  **3. API Keys Page Migration (3-4 часа)**
  - **Current:** `web/api-keys.html` + `web/js/apikeys.js` (250+ lines)
  - **Target:** HTMX inline editing + auto-refresh
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/apikeys_ui_handler.go
  GET  /api/ui/api-keys                    → HTML keys list
  GET  /api/ui/api-keys/personal           → HTML personal keys
  GET  /api/ui/api-keys/tenant/:id         → HTML tenant keys
  GET  /api/ui/api-keys/:id/edit           → HTML inline edit form
  POST /api/ui/api-keys/create-form        → HTML create modal
  ```
  
  **Frontend Changes:**

  ```html
  <!-- API Keys List -->
  <div id="api-keys-list" 
       hx-get="/api/ui/api-keys/personal" 
       hx-trigger="load">
      Loading...
  </div>
  
  <!-- Create API Key -->
  <form hx-post="/api/admin/keys"
        hx-target="#api-keys-list"
        hx-swap="beforeend"
        hx-on::after-request="this.reset()">
      <input name="name" required>
      <select name="models[]" multiple>
          <!-- Models list -->
      </select>
      <button type="submit">Create</button>
  </form>
  
  <!-- API Key Card (inline edit) -->
  <div class="api-key-card" id="key-{{ .ID }}">
      <div hx-get="/api/ui/api-keys/{{ .ID }}/edit" 
           hx-trigger="click from:.edit-btn"
           hx-swap="outerHTML">
          <h4>{{ .Name }}</h4>
          <p>Created: {{ .CreatedAt }}</p>
          <button class="edit-btn">✏️ Edit</button>
      </div>
      
      <button hx-delete="/api/admin/keys/{{ .ID }}"
              hx-target="closest .api-key-card"
              hx-swap="outerHTML swap:1s"
              hx-confirm="Delete {{ .Name }}?">
          🗑️ Delete
      </button>
  </div>
  
  <!-- Edit Form (server returns this) -->
  <form hx-put="/api/admin/keys/{{ .ID }}"
        hx-target="this"
        hx-swap="outerHTML">
      <input name="name" value="{{ .Name }}">
      <button type="submit">💾 Save</button>
      <button hx-get="/api/ui/api-keys/{{ .ID }}" 
              hx-swap="outerHTML">❌ Cancel</button>
  </form>
  ```
  
  **Templates:**
  - `web/templates/partials/apikeys/keys_list.html`
  - `web/templates/partials/apikeys/key_card.html`
  - `web/templates/partials/apikeys/key_edit_form.html`
  
  **Экономия:** ~200 lines JS → ~30 lines HTML
  
  **4. Tenants Page Migration (2-3 часа)**
  - **Current:** `web/tenants.html` + `web/js/tenants.js` (200+ lines)
  - **Target:** HTMX grid with modals
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/tenants_ui_handler.go
  GET  /api/ui/tenants               → HTML tenants grid
  GET  /api/ui/tenants/:id           → HTML tenant details
  GET  /api/ui/tenants/:id/members   → HTML members list
  ```
  
  **Frontend Changes:**

  ```html
  <!-- Tenants Grid -->
  <div id="tenants-grid" 
       hx-get="/api/ui/tenants" 
       hx-trigger="load"
       class="grid">
      Loading...
  </div>
  
  <!-- Create Tenant -->
  <form hx-post="/api/tenants" 
        hx-target="#tenants-grid"
        hx-swap="beforeend">
      <input name="name" required>
      <button>Create</button>
  </form>
  
  <!-- Tenant Card -->
  <div class="tenant-card" 
       hx-get="/api/ui/tenants/{{ .ID }}"
       hx-trigger="click"
       hx-target="#tenant-details">
      <h3>{{ .Name }}</h3>
      <p>{{ .MembersCount }} members</p>
  </div>
  ```
  
  **Templates:**
  - `web/templates/partials/tenants/tenants_grid.html`
  - `web/templates/partials/tenants/tenant_card.html`
  - `web/templates/partials/tenants/tenant_details.html`
  
  **Экономия:** ~150 lines JS → ~25 lines HTML
  
  **Acceptance Criteria:**
  - ✅ HTMX library integrated (14KB gzipped)
  - ✅ Template infrastructure setup
  - ✅ Model Registry page works with HTMX
  - ✅ API Keys page works with HTMX
  - ✅ Tenants page works with HTMX
  - ✅ All existing functionality preserved
  - ✅ No breaking changes for JSON API (dual endpoints)
  - ✅ ~650 lines JS code removed
  
  **Оценка:** 10-14 часов

- [ ] **HTMX-02** Live Updates & Real-time Features → **v2.6.2**
  
  **Подзадачи:**
  
  **1. GPU Monitor Real-time Updates (1-2 часа)**
  - **Current:** `web/js/gpu-monitor.js` - polling every 2s
  - **Target:** HTMX polling with auto-refresh
  
  **Backend (existing endpoint returns HTML):**

  ```go
  // internal/api/handlers/ui/gpu_ui_handler.go
  GET /api/ui/system/gpu → HTML GPU stats cards
  ```
  
  **Frontend Changes:**

  ```html
  <div class="gpu-stats" 
       hx-get="/api/ui/system/gpu" 
       hx-trigger="every 2s"
       hx-swap="innerHTML">
      <!-- GPU stats auto-refresh -->
  </div>
  ```
  
  **Template:**
  - `web/templates/partials/system/gpu_stats.html`
  
  **Экономия:** ~80 lines JS → 3 lines HTML
  
  **2. Audit Logs Pagination (2-3 часа)**
  - **Current:** `web/js/admin-audit.js` - manual pagination
  - **Target:** HTMX infinite scroll
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/audit_ui_handler.go
  GET /api/ui/audit?page=1              → HTML audit table
  GET /api/ui/audit/load-more?page=2    → HTML next rows
  ```
  
  **Frontend Changes:**

  ```html
  <table id="audit-logs">
      <tbody hx-get="/api/ui/audit?page=1" 
             hx-trigger="load"
             hx-swap="innerHTML">
          Loading...
      </tbody>
  </table>
  
  <!-- Load More Trigger -->
  <tr hx-get="/api/ui/audit/load-more?page=2" 
      hx-trigger="revealed"
      hx-swap="outerHTML">
      <td colspan="5">Loading more...</td>
  </tr>
  ```
  
  **Templates:**
  - `web/templates/partials/audit/audit_rows.html`
  - `web/templates/partials/audit/load_more_trigger.html`
  
  **Экономия:** ~40 lines JS → 5 lines HTML
  
  **3. Usage Stats Auto-refresh (1-2 часа)**
  - **Current:** `web/js/usage.js` - manual refresh button
  - **Target:** HTMX periodic polling
  
  **Backend (existing endpoint returns HTML):**

  ```go
  // internal/api/handlers/ui/usage_ui_handler.go
  GET /api/ui/usage/stats → HTML usage cards
  ```
  
  **Frontend Changes:**

  ```html
  <div class="usage-stats" 
       hx-get="/api/ui/usage/stats" 
       hx-trigger="load, every 30s"
       hx-swap="innerHTML">
      <!-- Stats auto-refresh every 30s -->
  </div>
  ```
  
  **Template:**
  - `web/templates/partials/usage/stats_cards.html`
  
  **Экономия:** ~30 lines JS → 3 lines HTML
  
  **Acceptance Criteria:**
  - ✅ GPU stats auto-refresh без JavaScript
  - ✅ Audit logs infinite scroll работает
  - ✅ Usage stats auto-update
  - ✅ No memory leaks (HTMX cleans up)
  - ✅ ~150 lines JS code removed
  
  **Оценка:** 4-7 часов

- [ ] **HTMX-03** Advanced Forms & Search → **v2.6.3**
  
  **Подзадачи:**
  
  **1. User Management Forms (2-3 часа)**
  - **Current:** `web/js/admin.js` - user CRUD modals
  - **Target:** HTMX inline validation + forms
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/users_ui_handler.go
  GET  /api/ui/admin/users                → HTML users table
  GET  /api/ui/admin/users/:id/edit       → HTML edit form
  POST /api/ui/admin/users/validate-email → HTML validation result
  ```
  
  **Frontend Changes:**

  ```html
  <!-- Users Table -->
  <table id="users-table" 
         hx-get="/api/ui/admin/users" 
         hx-trigger="load">
      Loading...
  </table>
  
  <!-- Create User Form with Inline Validation -->
  <form hx-post="/api/admin/users"
        hx-target="#users-table tbody"
        hx-swap="beforeend">
      
      <input name="username" 
             hx-post="/api/ui/admin/users/validate-username"
             hx-trigger="blur"
             hx-target="next .error"
             required>
      <span class="error"></span>
      
      <input type="email" name="email"
             hx-post="/api/ui/admin/users/validate-email"
             hx-trigger="blur"
             hx-target="next .error"
             required>
      <span class="error"></span>
      
      <button type="submit">Create User</button>
  </form>
  
  <!-- User Row -->
  <tr>
      <td>{{ .Username }}</td>
      <td>{{ .Email }}</td>
      <td>
          <button hx-get="/api/ui/admin/users/{{ .ID }}/edit"
                  hx-target="closest tr"
                  hx-swap="outerHTML">
              ✏️ Edit
          </button>
          <button hx-delete="/api/admin/users/{{ .ID }}"
                  hx-target="closest tr"
                  hx-swap="outerHTML swap:1s"
                  hx-confirm="Delete user?">
              🗑️ Delete
          </button>
      </td>
  </tr>
  ```
  
  **Templates:**
  - `web/templates/partials/admin/users_table.html`
  - `web/templates/partials/admin/user_form.html`
  - `web/templates/partials/admin/user_row.html`
  
  **Экономия:** ~100 lines JS → ~20 lines HTML
  
  **2. RBAC Management UI (2-3 часа)**
  - **Current:** `web/js/admin-rbac.js` - roles & permissions
  - **Target:** HTMX role assignment
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/rbac_ui_handler.go
  GET /api/ui/rbac/roles               → HTML roles list
  GET /api/ui/rbac/roles/:id/form      → HTML role edit form
  GET /api/ui/rbac/users/:id/roles     → HTML user roles
  ```
  
  **Frontend Changes:**

  ```html
  <!-- Roles List -->
  <div id="roles-list" 
       hx-get="/api/ui/rbac/roles" 
       hx-trigger="load">
      Loading...
  </div>
  
  <!-- Assign Role to User -->
  <form hx-post="/api/admin/rbac/users/{{ .UserID }}/roles"
        hx-target="#user-roles"
        hx-swap="beforeend">
      <select name="role_id">
          <!-- Roles dropdown -->
      </select>
      <button>Assign Role</button>
  </form>
  ```
  
  **Templates:**
  - `web/templates/partials/rbac/roles_list.html`
  - `web/templates/partials/rbac/role_form.html`
  - `web/templates/partials/rbac/user_roles.html`
  
  **Экономия:** ~80 lines JS → ~15 lines HTML
  
  **3. Dashboard Stats Cards (1-2 часа)**
  - **Current:** `web/js/dashboard.js` - stats loading
  - **Target:** HTMX parallel loading
  
  **Backend (новые endpoints):**

  ```go
  // internal/api/handlers/ui/dashboard_ui_handler.go
  GET /api/ui/dashboard/stats     → HTML all stats cards
  GET /api/ui/dashboard/recent    → HTML recent conversations
  ```
  
  **Frontend Changes:**

  ```html
  <!-- Stats Grid (loads in parallel) -->
  <div class="stats-grid">
      <div hx-get="/api/ui/dashboard/stats/conversations" 
           hx-trigger="load">...</div>
      <div hx-get="/api/ui/dashboard/stats/tenants" 
           hx-trigger="load">...</div>
      <div hx-get="/api/ui/dashboard/stats/apikeys" 
           hx-trigger="load">...</div>
      <div hx-get="/api/ui/dashboard/stats/requests" 
           hx-trigger="load">...</div>
  </div>
  
  <!-- Recent Conversations -->
  <div id="recent-conversations" 
       hx-get="/api/ui/dashboard/recent" 
       hx-trigger="load">
      Loading...
  </div>
  ```
  
  **Templates:**
  - `web/templates/partials/dashboard/stat_card.html`
  - `web/templates/partials/dashboard/recent_conversations.html`
  
  **Экономия:** ~50 lines JS → ~10 lines HTML
  
  **Acceptance Criteria:**
  - ✅ User forms with inline validation
  - ✅ RBAC role assignment UI
  - ✅ Dashboard stats parallel loading
  - ✅ All validation errors display inline
  - ✅ Form submission без page reload
  - ✅ ~230 lines JS code removed
  
  **Оценка:** 5-8 часов

- [ ] **HTMX-04** TUI Removal & Legacy Cleanup → **v2.6.4**
  
  **Подзадачи:**
  
  **1. TUI Application Removal (1-2 часа)**
  - **Delete TUI Command:**
    - Remove entire `cmd/tui/` directory (5 files):
      - `main.go`, `logs_reader.go`, `metrics_parser.go`
      - `scroll_helper.go`, `time_helper.go`
  
  **2. TUI Handlers Removal (1-2 часа)**
  - **Delete TUI-only handlers:**
    - `internal/api/handlers/stats.go` - StatsHandler (только для TUI)
    - `internal/api/handlers/requests.go` - RequestsHandler (TUI-04)
    - ⚠️ Keep `config.go` - используется в admin panel
  
  - **Remove from Router:**

    ```go
    // internal/api/router/router.go
    // DELETE these fields:
    statsHandler          *handlers.StatsHandler
    requestsHandler       *handlers.RequestsHandler
    
    // DELETE these routes:
    GET /api/stats                    // TUI stats endpoint
    GET /api/requests                 // Request monitoring
    GET /api/requests/stats
    GET /api/requests/:id
    DELETE /api/requests
    
    // DELETE these methods:
    setupStatsRoutes()
    setupRequestMonitoringRoutes()
    ```
  
  **3. Config Cleanup (30 минут)**
  - **Remove TUI Config:**

    ```go
    // internal/config/config.go
    // DELETE TUIConfig struct (lines ~409-425)
    // DELETE TUI field from Config struct
    // DELETE TUI defaults from setConfigDefaults()
    ```
  
  - **Update config files:**

    ```yaml
    # configs/dev.yaml, configs/production.yaml.example
    # DELETE entire tui: section
    ```
  
  **4. BACKLOG Cleanup (15 минут)**
  - **Delete TUI BACKLOG files (11 files):**
    - `BACKLOG/TUI-01_api_keys_management.md`
    - `BACKLOG/TUI-02_basic_infrastructure.md`
    - `BACKLOG/TUI-03_dashboard_screen.md`
    - `BACKLOG/TUI-04_request_monitor.md`
    - `BACKLOG/TUI-05_model_manager_screen.md`
    - `BACKLOG/TUI-06_api_keys_manager_screen.md`
    - `BACKLOG/TUI-07_configuration_viewer.md`
    - `BACKLOG/TUI-08_logs_viewer.md`
    - `BACKLOG/TUI-09_server_control.md`
    - `BACKLOG/TUI-10_realtime_communication.md`
    - `BACKLOG/TUI-11_advanced_features.md`
    - `BACKLOG/TODO-01_tui_admin_key_config.md`
  
  **5. Documentation Updates (30 минут)**
  - **Update documentation:**
    - Remove TUI from `README.md` (installation, usage sections)
    - Remove TUI from `Makefile` (build-tui, run-tui commands)
    - Remove TUI from `Architecture.MD`
    - Remove TUI from `docker/Dockerfile`, `docker-compose.yml`
    - Remove TUI mentions from `CHANGELOG.md`, `Plan.md`
  
  **6. Dependencies Cleanup (15 минут)**
  - **Check and remove TUI-only dependencies:**

    ```bash
    # Check if these are TUI-only:
    github.com/charmbracelet/bubbletea
    github.com/charmbracelet/lipgloss
    github.com/charmbracelet/bubbles
    
    # Run go mod tidy
    go mod tidy
    ```
  
  **7. Request Storage Decision (30 минут)**
  - **Investigate `internal/request/storage.go`:**
    - Check if used by WebUI admin panel
    - If TUI-only → delete entire package
    - If used → keep but remove TUI references
  
  **8. Final Cleanup (30 минут)**
  - Remove TUI references from code comments
  - Update API documentation
  - Check for broken imports
  - Run tests: `go test ./...`
  - Run linter: `golangci-lint run`
  
  **Files to DELETE (summary):**

  ```
  cmd/tui/                               # 5 files (TUI app)
  internal/api/handlers/stats.go         # StatsHandler
  internal/api/handlers/requests.go      # RequestsHandler
  internal/request/                      # If TUI-only
  BACKLOG/TUI-*.md                       # 11 BACKLOG files
  BACKLOG/TODO-01_tui_admin_key_config.md
  ```
  
  **Lines to MODIFY:**

  ```
  internal/config/config.go              # Remove TUIConfig
  internal/api/router/router.go          # Remove TUI routes/handlers
  configs/dev.yaml                       # Remove tui: section
  configs/production.yaml.example        # Remove tui: section
  README.md                              # Remove TUI docs
  Makefile                               # Remove TUI commands
  Architecture.MD                        # Remove TUI architecture
  docker/Dockerfile                      # Remove TUI build (if any)
  go.mod                                 # go mod tidy
  ```
  
  **Acceptance Criteria:**
  - ✅ All TUI files deleted
  - ✅ No TUI references in router
  - ✅ No TUI config sections
  - ✅ All BACKLOG TUI tasks deleted
  - ✅ Documentation updated
  - ✅ `go build ./...` successful
  - ✅ `go test ./...` passes
  - ✅ No import errors
  - ✅ Makefile commands work (without TUI)
  
  **Why removing TUI:**
  - ❌ **Not used** - WebUI заменяет все функции TUI
  - ❌ **Maintenance burden** - ~2000+ lines code + 11 BACKLOG tasks
  - ❌ **Deprecated** - TUI не развивался с v1.4.0
  - ✅ **WebUI Superior** - better UX, more features, actively developed
  
  **Оценка:** 4-6 часов

### 🎯 Ключевые возможности после v2.6.0

**Code Reduction:**

- ✅ **~1030 lines JavaScript removed** (Model Registry: 300, API Keys: 200, Tenants: 150, GPU: 80, Audit: 40, Usage: 30, Users: 100, RBAC: 80, Dashboard: 50)
- ✅ **~150 lines HTML templates added** (но проще чем JS)
- ✅ **~2000+ lines Go code removed** (TUI app, handlers, config)
- ✅ **12 BACKLOG files deleted** (TUI-01 to TUI-11 + TODO-01)
- ✅ **88% reduction** в migrated WebUI pages
- ✅ **100% TUI removal** - cleaner codebase

**Benefits:**

- ✅ **Faster Development** - CRUD pages за 30 минут вместо 2 часов
- ✅ **Easier Maintenance** - less code = fewer bugs
- ✅ **Better UX** - instant updates, no loading spinners
- ✅ **No Breaking Changes** - JSON API остается для Desktop client

**What Stays on Vanilla JS:**

- ❌ **Chat Page** - too complex state (Monaco, streaming, file attachments)
- ❌ **Desktop Client** - Svelte (separate stack)

### 🏗️ Технический стек

**Frontend:**

- **HTMX** v1.9.12 (14KB gzipped)
- **Extensions:** None needed (core HTMX sufficient)
- **CSS:** Existing theme.css (no changes)
- **JavaScript:** Minimal (only for complex interactions)

**Backend:**

```
internal/api/handlers/ui/
  ├── registry_ui_handler.go     # Model Registry HTML
  ├── apikeys_ui_handler.go      # API Keys HTML
  ├── tenants_ui_handler.go      # Tenants HTML
  ├── users_ui_handler.go        # Users HTML
  ├── rbac_ui_handler.go         # RBAC HTML
  ├── dashboard_ui_handler.go    # Dashboard HTML
  ├── gpu_ui_handler.go          # GPU Stats HTML
  ├── audit_ui_handler.go        # Audit Logs HTML
  └── usage_ui_handler.go        # Usage Stats HTML

web/templates/
  ├── layouts/
  │   └── base.html              # Base layout
  ├── partials/
  │   ├── registry/              # Model Registry fragments
  │   ├── apikeys/               # API Keys fragments
  │   ├── tenants/               # Tenants fragments
  │   ├── admin/                 # Admin fragments
  │   ├── rbac/                  # RBAC fragments
  │   ├── dashboard/             # Dashboard fragments
  │   ├── system/                # System fragments
  │   └── audit/                 # Audit fragments
  └── components/
      ├── stat_card.html         # Reusable stat card
      ├── table.html             # Reusable table
      └── modal.html             # Reusable modal
```

### 📈 Use Cases

**1. Admin adds new model:**

```
User: Clicks "Add Model" button
      ↓
HTMX: hx-get="/api/ui/registry/models/new-form"
      ↓
Server: Returns HTML form in modal
      ↓
User: Fills form → Submits
      ↓
HTMX: hx-post="/api/admin/registry/models"
      ↓
Server: Creates model → Returns HTML row
      ↓
HTMX: Appends row to table (beforeend swap)
      ↓
Result: New model appears instantly
```

**2. User searches API keys:**

```
User: Types "prod" in search
      ↓
HTMX: Waits 300ms → hx-get="/api/ui/api-keys/search?q=prod"
      ↓
Server: Filters keys → Returns HTML rows
      ↓
HTMX: Replaces table body
      ↓
Result: Filtered results instantly
```

**3. GPU stats auto-refresh:**

```
Page Load
      ↓
HTMX: hx-trigger="load, every 2s"
      ↓
Server: Returns HTML GPU cards
      ↓
HTMX: Updates div every 2 seconds
      ↓
Result: Live GPU monitoring
```

### 🔄 Dual Endpoint Strategy

**JSON API** (existing, не трогаем):

```
GET  /api/admin/registry/models  → JSON (for Desktop, API clients)
POST /api/admin/registry/models  → JSON (creates model)
```

**HTML API** (новые для HTMX):

```
GET /api/ui/registry/models          → HTML (for WebUI)
GET /api/ui/registry/models/search   → HTML (filtered)
```

**Benefits:**

- ✅ Desktop client не затронут
- ✅ API clients продолжают работать
- ✅ WebUI получает instant updates
- ✅ No breaking changes

### 📊 Performance Impact

**Before (Vanilla JS):**

- Request: Client → JSON API
- Processing: Client parses JSON → Renders HTML → Updates DOM
- Total: ~150ms (network + parsing + rendering)

**After (HTMX):**

- Request: Client → HTML API
- Processing: Server renders HTML → Client swaps DOM
- Total: ~80ms (network + DOM swap)

**Improvement:** ~47% faster page updates

### 🔒 Security Considerations

**CSRF Protection:**

```go
// All POST/PUT/DELETE require CSRF token
admin.Use(middleware.CSRFProtection())
```

**XSS Prevention:**

```go
// All HTML templates auto-escaped
{{ .UserInput }}  // Automatically escaped
```

**Authorization:**

```go
// All UI endpoints require same auth as JSON API
admin.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
admin.Use(middleware.RequireAdmin(r.db, r.logger))
```

### 📝 Migration Checklist

**Phase 1 (HTMX-01):**

- [ ] Add htmx.min.js to project
- [ ] Setup template infrastructure
- [ ] Create UI handlers package
- [ ] Migrate Model Registry page
- [ ] Migrate API Keys page
- [ ] Migrate Tenants page
- [ ] Test all CRUD operations
- [ ] Verify JSON API still works

**Phase 2 (HTMX-02):**

- [ ] Migrate GPU monitoring
- [ ] Migrate Audit logs pagination
- [ ] Migrate Usage stats
- [ ] Test auto-refresh features
- [ ] Verify no memory leaks

**Phase 3 (HTMX-03):**

- [ ] Migrate User management forms
- [ ] Migrate RBAC UI
- [ ] Migrate Dashboard stats
- [ ] Test inline validation
- [ ] Final cleanup old JS files

**Phase 4 (HTMX-04) - TUI Removal:**

- [ ] Delete `cmd/tui/` directory (5 files)
- [ ] Delete `internal/api/handlers/stats.go`
- [ ] Delete `internal/api/handlers/requests.go`
- [ ] Remove TUI routes from router
- [ ] Remove TUIConfig from config
- [ ] Delete 12 TUI BACKLOG files
- [ ] Update README.md (remove TUI)
- [ ] Update Makefile (remove TUI commands)
- [ ] Update Architecture.MD
- [ ] Run `go mod tidy`
- [ ] Test: `go build ./...` passes
- [ ] Test: `go test ./...` passes

---

## 🔮 Version 2.7.0+ - Future Versions

**Задачи для будущих версий** (планируются после v2.5.0 и v2.6.0)

### Version 2.7.0 - Redis Integration & Distributed Sessions

**Фокус:** Redis для session management, rate limiting и caching  
**Приоритет:** MEDIUM (для production deployment)  
**Оценка времени:** 12-16 часов  
**Статус:** 📋 Планируется (после v2.6.0)  
**Версионирование:** v2.7.1-v2.7.3  
**Dependencies:** None (опциональная интеграция)

**Стратегия:** Добавление Redis как опционального backend для:

- Distributed rate limiting (multi-instance Kubernetes)
- Centralized session management (JWT token blacklist)
- Response caching для performance
- Active sessions tracking

**Задачи:**

- [ ] **REDIS-01** Redis Connection Pool & Configuration → **v2.7.1**
  - Redis client setup (go-redis)
  - Connection pool configuration
  - Health checks & auto-reconnect
  - Fallback to in-memory если Redis unavailable
  - Config: `redis.enabled`, `redis.url`, `redis.pool_size`
  - **Оценка:** 2-3 часа

- [ ] **REDIS-02** Distributed Rate Limiting → **v2.7.2**
  - Migrate rate limiter to Redis backend
  - Multi-instance synchronization
  - Sliding window algorithm в Redis
  - Per-key, per-tenant, per-user limits
  - Graceful degradation to in-memory
  - **Use Case:** Kubernetes deployment с 3+ replicas
  - **Оценка:** 4-6 часов

- [ ] **REDIS-03** Centralized Session Management → **v2.7.3**
  - JWT token blacklist в Redis (refresh token revocation)
  - Active sessions tracking (user → [session_ids])
  - TTL для automatic token expiration
  - Force logout API (admin kills all user sessions)
  - Session management UI в WebUI
  - **Use Case:** Админ может отозвать все сессии скомпрометированного пользователя
  - **Оценка:** 6-8 часа

**Benefits:**

- ✅ **Horizontal Scaling** - multi-instance deployment
- ✅ **Centralized Control** - single source of truth для sessions
- ✅ **Better Security** - immediate token revocation
- ✅ **Performance** - faster rate limiting checks

---

### Version 2.8.0+ - Multi-Provider Expansion

**Фокус:** Поддержка дополнительных LLM providers (OpenAI, Anthropic, Cohere)  
**Приоритет:** LOW (после Redis и Agentic AI)  
**Статус:** 📋 Рассматривается

**Potential Providers:**

- OpenAI API (gpt-4, gpt-3.5-turbo) - для fallback
- Anthropic Claude (claude-3-opus, claude-3-sonnet)
- Cohere (command-r+, embed-v3)
- Google Gemini (gemini-1.5-pro)

**Архитектура:** Использовать существующий Provider Interface из v2.3.0

---

### Technical Debt & Quality of Life

**Задачи без конкретной версии:**

- [ ] **TODO-07** PostgreSQL Production Readiness
  - Завершить реализацию PostgreSQL storage
  - Заменить заглушки в `internal/storage/postgresql/stubs.go`
  - Full parity с SQLite functionality
  - Migration guide SQLite → PostgreSQL
  - **Приоритет:** HIGH (для production)
  - **Оценка:** 8-12 часов
  - **Когда:** Перед production deployment

- [ ] **TODO-08** Test Helpers Enhancement
  - JSON парсинг и проверка error.code
  - Детальное сравнение JSON объектов
  - HTTP requests с JSON body helpers
  - Файл: `internal/test/helpers.go`
  - **Приоритет:** LOW (quality of life)
  - **Оценка:** 2-3 часа

---

### Ideas for Distant Future

**Концепции для обсуждения** (не запланированы):

- **Plugin System** - динамическая загрузка расширений
  - Может быть полезно в далеком будущем
  - MCP + Provider Interface достаточно пока
  
- **Alternative Protocols**
  - WebSocket для TUI (вместо HTTP polling)
  - GraphQL API (избыточно для текущих use cases)
  - gRPC API (нет необходимости, REST + WebSocket достаточно)

---

## 📊 Краткая статистика Roadmap

| Version | Задач | Время (ч) | Приоритет | Статус | Заметки |
|---------|-------|-----------|-----------|--------|---------|
| **v1.2-v1.5** | 26 | ~150 | - | ✅ Завершено | См. CHANGELOG |
| **v1.6.0** | 3 | 22-28 | MEDIUM | ✅ Завершено | Observability |
| **v1.9.0** | 4 | 36-48 | HIGH | ✅ Завершено | Model Management + Cache Opt |
| **v1.10.0** | 4 | 32-44 | HIGH | ✅ Завершено | Content Foundation |
| **v1.11.0** | 8 | 88-120 | HIGH | ✅ Завершено | Enterprise Suite |
| **v1.12.0** | 3 | 18-28 | MEDIUM | ✅ Завершено | UX & Performance |
| **v2.0.0** | 5 | 54-72 | HIGH | ✅ Завершено | RAG System (ex-v1.13.0) |
| **v2.1.0** | 4 | 8-12 | MEDIUM | ✅ Завершено | AIGateway Rebranding |
| **v2.2.0** | 1 | 6-8 | HIGH | ✅ Завершено | Invitation System |
| **v2.3.0** | 3 | 16-22 | MEDIUM | ✅ Завершено | Model Registry |
| **v2.4.0** | 5 | 16-22 | HIGH | 🔄 В работе | Desktop Client Support + Refactoring |
| **v2.5.0** | 5 | 28-35 | HIGH | 📋 Запланировано | Agentic AI & MCP |
| **v2.6.0** | 4 | 23-31 | MEDIUM | 📋 Запланировано | HTMX + TUI Removal |
| **v2.7.0** | 3 | 12-16 | MEDIUM | 📋 Планируется | Redis Integration |
| **Future** | 2 | 10-15 | LOW | 📋 Tech Debt | PostgreSQL + Test Helpers |
| **ИТОГО** | **90** | **519-671** | - | - | - |

**Прогресс:** ✅ 9 major версий завершены (v1.6 - v2.3) | 🔄 1 в работе (v2.4 - REFACTOR-01) | 📋 3 запланированы (v2.5, v2.6, v2.7)

**Подход к версионированию:**  
🔄 1 фича = 1 patch версия (например: v1.10.1, v1.10.2, ... → v1.11.0)

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

## 🔨 Декомпозиция сложных задач

### OIDC-01: Keycloak SSO Integration (16-24ч) → v1.11.1

**Phase 1: OAuth2 Flow Implementation (6-8ч)**

- OAuth2 authorization code flow
- Token endpoint integration
- PKCE support для security
- State parameter validation

**Phase 2: Token Management (4-6ч)**

- JWT token validation (signature, expiry)
- Refresh token flow
- Token revocation
- Session management

**Phase 3: User Provisioning (3-4ч)**

- Auto-create users from OIDC claims
- User attribute mapping (email, name, picture)
- Tenant auto-assignment
- Fallback to local auth

**Phase 4: Admin UI (3-6ч)**

- OIDC provider configuration page
- Client ID/Secret management
- Endpoint URL configuration
- Test connection button
- Enable/disable toggle

---

### LDAP-01: LDAP/Active Directory Integration (12-16ч) → v1.11.3

**Phase 1: LDAP Connection (3-4ч)**

- LDAP bind (simple auth + SASL)
- Connection pooling
- TLS/LDAPS support
- Health check & reconnect logic

**Phase 2: User Search & Sync (4-5ч)**

- LDAP search filters
- User attribute mapping (CN, mail, memberOf)
- Incremental sync (track changes)
- Pagination для больших directories

**Phase 3: Group Management (2-3ч)**

- Group search & parsing
- User-group membership resolution
- Nested groups support
- Group-to-tenant mapping

**Phase 4: Admin UI & Testing (3-4ч)**

- LDAP configuration form
- Test connection/search
- Manual sync trigger
- Scheduled sync configuration
- Sync status dashboard

---

### QUOTA-01: Usage Quotas System (10-14ч) → v1.11.7

**Phase 1: Quota Storage & Models (2-3ч)**

- Database schema (quota_configs, quota_usage)
- Quota models (token, request, storage limits)
- Per-tenant & per-user quotas
- Time-based quotas (daily, monthly)

**Phase 2: Quota Enforcement (3-4ч)**

- Middleware для quota checks
- Token counting (before/after request)
- Request counting
- Storage size tracking
- Quota exceeded error handling

**Phase 3: Quota Monitoring (2-3ч)**

- Usage calculation & aggregation
- Near-limit warnings (80%, 90%, 95%)
- Quota reset logic (daily/monthly)
- Usage history tracking

**Phase 4: Admin UI & Alerts (3-4ч)**

- Quota configuration page
- Per-tenant quota dashboard
- Usage charts (integration с METRICS-01)
- Email/webhook alerts для quota violations
- Quota templates (starter, pro, enterprise)

---

### FILE-STORAGE-01: Universal File Storage (10-14ч) → v1.10.1

**Phase 1: Storage Foundation (3-4ч)**

- StorageBackend interface
- Local filesystem implementation
- S3/MinIO client setup
- Configuration loading

**Phase 2: Document Extractors (4-5ч)**

- PDF extractor (pdftotext or go-fitz)
- DOCX extractor (ZIP + XML parsing)
- TXT extractor (encoding detection)
- CSV parser
- Extractor registry pattern

**Phase 3: Database Integration (2-3ч)**

- Files table migration
- Metadata CRUD operations
- Full-text search index
- Access control checks

**Phase 4: API & Chat Integration (1-2ч)**

- Upload/download endpoints
- File listing API
- Chat integration (file → context)
- WebUI file picker

---

### RAG-04: External Data Sources (10-14ч) → v1.13.4

**Phase 1: REST API Source (4-5ч)**

- HTTP client с auth (Bearer, Basic, API Key)
- Request template engine (Jinja-like)
- Response parsing (JSON, XML)
- Pagination support
- Rate limiting per source

**Phase 2: PostgreSQL Source (3-4ч)**

- Connection pool management
- Parameterized query execution
- Row-based chunking
- Column selection для embedding
- Query result caching

**Phase 3: Credentials Management (2-3ч)**

- AES-256 encryption для credentials
- Secure key storage (env var or secret manager)
- Credential rotation support
- Connection string parsing

**Phase 4: Sync & Scheduling (1-2ч)**

- Manual sync trigger
- Scheduled sync (cron-like)
- Incremental sync detection
- Sync status tracking
- Error retry logic

---

## 📋 Рекомендации по имплементации

### Порядок разработки для v1.11.0 (Enterprise Suite)

1. **Start with OIDC-01** - базовая SSO функциональность
2. **Then RBAC-01** - установить permission framework
3. **Then LDAP-01** - второй auth provider
4. **Then AUDIT-01** - логирование для compliance
5. **Then METRICS-01** - observability для всего выше
6. **Finally QUOTA-01** - resource control с метриками

### Критические зависимости

- **OIDC-02** зависит от **OIDC-01** (need claims parsing)
- **QUOTA-01** зависит от **METRICS-01** (need usage tracking)
- **RATE-02** (v1.12) зависит от **QUOTA-01** (shared enforcement logic)
- **RAG-02** зависит от **FILE-STORAGE-01** (need extractors)
- **RAG-05** зависит от **WS-01** (need progress updates)

### Performance considerations

- **OIDC token validation**: Cache public keys (60min TTL)
- **LDAP sync**: Use incremental sync, не full sync каждый раз
- **Quota checks**: Redis cache для fast lookups
- **File extraction**: Async processing, не блокировать upload

---

**Последнее обновление:** 2025-10-30  
**Версия Roadmap:** 2.3  
**Статус:** Active Development 🚀 (10 Versions ✅ Complete! → Planning v2.5.0 Agentic AI + v2.6.0 HTMX Migration)
