# 🗺️ Roadmap: Ollama-OpenAI Proxy

> **Текущая версия:** 1.11.7 ✅  
> **Следующая версия:** 1.11.8 (UI-SYNC-01 WebUI Functionality Restoration) 🔄  
> **План развития:** Version 1.9.0 - 1.13.0+  
> **Статус:** Active Development (Version 1.11.8 In Progress - WebUI Redesign & Functionality Sync)  
> **Последнее обновление:** 2025-10-26  
> **Новое:** 🎨 v1.11.8 UI-SYNC-01 WebUI Redesign в работе! Синхронизация функционала /web_old/ → /web/ с новым theme.css дизайном. Roadmap: v1.11 (Enterprise Auth ✅) → v1.11.8 (UI Sync 🔄) → v1.12 (UX & Performance) → v1.13 (RAG)

---

## 📋 История завершенных версий

Для сокращения Roadmap завершенные версии v1.2.0-v1.5.0 перенесены в [CHANGELOG.md](CHANGELOG.md).

**Краткая сводка:**

- ✅ **v1.2.0** - Enhanced Monitoring & Management (4 задачи)
- ✅ **v1.3.0** - User Experience & Multi-Tenancy (5 задач)
- ✅ **v1.4.0** - WebUI Enhancements & Code Quality (5 задач)
- ✅ **v1.5.0** - Operations & Maintenance (12 задач)

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

## ⚡ Version 1.12.0 - User Experience & Performance

**Фокус:** Улучшение UX и производительности системы  
**Приоритет:** MEDIUM  
**Оценка времени:** 18-28 часов  
**Статус:** 📋 Запланировано (0/3 задач)  
**Versioning:** v1.12.1-v1.12.3

**Стратегия:** Компактный релиз фокусирующийся на user experience, экспорте данных и оптимизации производительности.

### Задачи

- [ ] **MODEL-02** Model Preloading & Warming → [BACKLOG/MODEL-02_model_preloading.md](BACKLOG/MODEL-02_model_preloading.md) → **v1.12.1**
  - Keep models in memory
  - Configurable warm-up strategy
  - Health checks для loaded models
  - Automatic unload of unused models
  - Priority-based preloading
  - **Оценка:** 6-8 часов

- [ ] **RATE-02** Advanced Rate Limiting → [BACKLOG/RATE-02_advanced_rate_limiting.md](BACKLOG/RATE-02_advanced_rate_limiting.md) → **v1.12.2**
  - Per tenant, per model limits
  - Sliding window algorithm
  - Rate limit headers в response
  - Burst allowance
  - Integration с QUOTA-01 (из v1.11)
  - **Оценка:** 8-10 часов

- [ ] **EXPORT-01** Conversation Export/Import → [BACKLOG/EXPORT-01_conversation_export_import.md](BACKLOG/EXPORT-01_conversation_export_import.md) → **v1.12.3**
  - Export to JSON, Markdown, PDF
  - Bulk export для всех conversations
  - Import для восстановления истории
  - Date range filtering
  - Export с RAG metadata (если v1.13 установлена)
  - **Примечание:** Перенесена из v1.10 для логической группировки с UX
  - **Оценка:** 4-6 часов

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

## 🧠 Version 1.13.0 - RAG System

**Фокус:** Полноценная RAG (Retrieval-Augmented Generation) система  
**Приоритет:** HIGH  
**Оценка времени:** 54-72 часа (экономия 6-8ч благодаря v1.10.0)  
**Статус:** 📋 Запланировано (0/5 задач)  
**Версионирование:** v1.13.1-v1.13.5  
**Dependencies:** ⚠️ Требует v1.10.0 (FILE-STORAGE-01, IMAGE-01, WEB-FETCH-01)

**Стратегия:** Расширяем существующие компоненты из v1.10.0 добавляя векторный поиск, chunking, embeddings и orchestration. Переиспользуем file storage, extractors и web fetcher.

### Задачи

- [ ] **RAG-01** Foundation & Data Sources Management → [BACKLOG/RAG-01_system_architecture.md](BACKLOG/RAG-01_system_architecture.md) → **v1.13.1**
  - Database schema (sources, documents, chunks, jobs)
  - Data sources CRUD API
  - ~~File storage (Local + S3/MinIO)~~ ✅ *Переиспользуем из v1.10.0*
  - WebUI: Data Sources management
  - Basic in-memory queue
  - **Оценка:** 12-16 часов (меньше т.к. storage готов)

- [ ] **RAG-02** Document Processing Pipeline → **v1.13.2**
  - ~~PDF/DOCX/CSV extractors~~ ✅ *Переиспользуем из v1.10.0*
  - Semantic chunking engine (новое)
  - PostgreSQL job queue (новое)
  - Worker pool (async processing)
  - Progress tracking UI (WebSocket из v1.10.0)
  - **Оценка:** 12-16 часов (меньше т.к. extractors готовы)

- [ ] **RAG-03** Embeddings & Vector Search → **v1.13.3**
  - Ollama embeddings integration (nomic-embed-text, bge-m3)
  - pgvector setup + HNSW indexing
  - Batch embedding processing
  - Vector search API
  - Search testing UI
  - **Оценка:** 12-16 часов

- [ ] **RAG-04** External Data Sources → **v1.13.4**
  - REST API source implementation
  - PostgreSQL database source
  - Credentials encryption (AES-256)
  - Connection testing
  - Scheduled auto-sync
  - **Оценка:** 10-14 часов

- [ ] **RAG-05** RAG Integration & Chat UI → **v1.13.5**
  - RAG Orchestrator
  - Reranking logic
  - Context assembly (max 64k of 128k)
  - Chat API integration
  - WebUI: Source selector in chat sidebar
  - RAG metadata in responses
  - **Оценка:** 8-10 часов

### 🎯 Ключевые возможности

- 📄 Multi-format файлы (PDF, DOCX, CSV, TXT, изображения)
- 🔌 Внешние API с Bearer token аутентификацией
- 🗄️ PostgreSQL data sources с параметризованными запросами
- 🧠 Semantic chunking с учетом контекста (512 tokens, 10% overlap)
- 🔍 Vector search через pgvector (HNSW indexing)
- ⚡ Async processing с worker pool
- 🎨 UI-first approach для управления источниками
- 🔐 AES-256 encryption для credentials
- 📊 Analytics dashboard для RAG usage

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

| Version | Задач | Время (ч) | Приоритет | Статус | Заметки |
|---------|-------|-----------|-----------|--------|---------|
| **v1.2-v1.5** | 26 | ~150 | - | ✅ Завершено | См. CHANGELOG |
| **v1.6.0** | 3 | 22-28 | MEDIUM | ✅ Завершено | - |
| **v1.9.0** | 3 | 20-28 | HIGH | ✅ Завершено | - |
| **v1.10.0** | 4 | 32-44 | HIGH | 📋 Запланировано | Foundation для RAG |
| **v1.11.0** | 7 | 72-100 | HIGH | 📋 Запланировано | Enterprise Suite |
| **v1.12.0** | 3 | 18-28 | MEDIUM | 📋 Запланировано | UX & Performance |
| **v1.13.0** | 5 | 54-72 | HIGH | 📋 Запланировано | ⚠️ Требует v1.10.0 |
| **Future** | 9+ | TBD | DEFERRED | 📋 Отложено | - |
| **ИТОГО** | **22** | **196-272** | - | - | Оптимизирован |

**Прогресс:** ✅ 2 major версии завершены (v1.6, v1.9) | 📋 4 запланированы (v1.10-v1.13)

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

**Последнее обновление:** 2025-01-16  
**Версия Roadmap:** 2.2  
**Статус:** Active Development 🚀 (Version 1.9.0 ✅ Complete! → Planning v1.10.0-v1.13.0)
