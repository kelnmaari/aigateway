# 🗺️ Roadmap: Ollama-OpenAI Proxy

> **Текущая версия:** 1.9.3 ✅  
> **План развития:** Version 1.9.0 - 1.13.0+  
> **Статус:** Active Development (Version 1.9.0 Complete! - 3/3 задачи)  
> **Последнее обновление:** 2025-01-15  
> **Новое:** 🧠 v1.13.0 RAG System добавлена! v1.10.0 реорганизована как Foundation layer для RAG. Roadmap: v1.10 (Content Foundation) → v1.11 (Enterprise Auth) → v1.12 (Model Pro II) → v1.13 (RAG)

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

**Фокус:** Продвинутое управление моделями + Performance Monitoring  
**Приоритет:** MEDIUM → HIGH  
**Оценка времени:** 20-28 часов (3 из 7 задач)  
**Фактическое время:** ~24 часа  
**Статус:** 🎉 Завершено 100% (3/3 выполненных задач)  
**Версионирование:** v1.9.1, v1.9.2, v1.9.3

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

### 🎯 Достижения Version 1.9.0

- ✅ Dynamic model parameters в Chat UI
- ✅ Compact UI дизайн (Cursor-style)
- ✅ Context window tracking + auto-summarization
- ✅ MoniGo Performance Dashboard
- ✅ NVIDIA GPU monitoring (multi-GPU support)
- ✅ Admin Panel reorganization
- ✅ localStorage persistence для всех настроек

**Оставшиеся задачи перенесены:**

- QUOTA-01, MODEL-02, METRICS-01, RATE-02 → **v1.12.0 (Model Management Pro II)**

---

## 📱 Version 1.10.0 - Content Foundation (RAG Prerequisites)

**Фокус:** Foundation layer для работы с контентом (подготовка к RAG v1.13)  
**Приоритет:** HIGH  
**Оценка времени:** 38-52 часа  
**Статус:** 📋 Запланировано (0/5 задач)  
**Версионирование:** v1.10.1-v1.10.5

**Стратегия:** Реализуем базовые компоненты работы с файлами, изображениями и веб-контентом с заделом на RAG систему (v1.13.0). Все extractors и storage будут переиспользованы в RAG.

### Задачи

- [ ] **FILE-STORAGE-01** Universal File Storage & Document Processing → [BACKLOG/FILE-STORAGE-01_universal_storage.md](BACKLOG/FILE-STORAGE-01_universal_storage.md) → **v1.10.1**
  - **Universal Storage Layer** (Local FS + S3/MinIO) 🔄 *Будет использован в RAG*
  - **Document Extractors** (PDF, DOCX, TXT, CSV) 🔄 *Основа для RAG processing*
  - File metadata management в PostgreSQL
  - Upload API + validation (size, type)
  - Basic integration с чатом (загрузка → extraction → контекст)
  - **Примечание:** Архитектура спроектирована для RAG (v1.13)
  - **Оценка:** 10-14 часов

- [ ] **IMAGE-01** Image Upload & OCR Processing → [BACKLOG/IMAGE-01_image_processing.md](BACKLOG/IMAGE-01_image_processing.md) → **v1.10.2**
  - Image storage (переиспользует FILE-STORAGE-01) 🔄 *Будет в RAG*
  - Multimodal OCR через Ollama (LLaVA, BakLLaVA) 🔄 *Будет в RAG*
  - Drag & drop upload в чат
  - Thumbnail generation
  - Image → text extraction для контекста
  - **Примечание:** OCR engine будет расширен для RAG
  - **Оценка:** 8-10 часов

- [ ] **WEB-FETCH-01** Web Content Fetcher → [BACKLOG/WEB-FETCH-01_content_fetcher.md](BACKLOG/WEB-FETCH-01_content_fetcher.md) → **v1.10.3**
  - HTTP client + HTML parser (goquery) 🔄 *Будет web source в RAG*
  - Text extraction и metadata
  - URL validation и security checks
  - Rate limiting
  - Basic summarization через LLM
  - **Примечание:** Будет основой для web data sources в RAG
  - **Оценка:** 6-8 часов

- [ ] **EXPORT-01** Conversation Export/Import → [BACKLOG/EXPORT-01_conversation_export_import.md](BACKLOG/EXPORT-01_conversation_export_import.md) → **v1.10.4**
  - Export to JSON, Markdown, PDF
  - Bulk export для всех conversations
  - Import для восстановления истории
  - Date range filtering
  - **Оценка:** 4-6 часов

- [ ] **WS-01** WebSocket Real-time Updates → [BACKLOG/WS-01_websocket_realtime.md](BACKLOG/WS-01_websocket_realtime.md) → **v1.10.5**
  - Real-time updates вместо polling
  - Live chat streaming
  - System notifications push
  - Auto-reconnect logic
  - Progress updates для file processing 🔄 *Нужно для RAG*
  - **Оценка:** 8-12 часов

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

## 🔐 Version 1.11.0 - Enterprise Auth

**Фокус:** Корпоративная аутентификация  
**Приоритет:** HIGH  
**Оценка времени:** 58-78 часов  
**Статус:** 📋 Запланировано (0/5 задач)  
**Версионирование:** v1.11.1-v1.11.5

### Задачи

- [ ] **OIDC-01** Keycloak SSO Integration → [BACKLOG/OIDC-01_keycloak_sso.md](BACKLOG/OIDC-01_keycloak_sso.md) → **v1.11.1**
  - OAuth2/OIDC flow
  - Token validation & refresh
  - UI для настройки через админку
  - User auto-provisioning
  - Fallback to local auth
  - **Оценка:** 16-24 часа

- [ ] **OIDC-02** Auto-tenant Provisioning from OIDC Groups → [BACKLOG/OIDC-02_auto_tenant_groups.md](BACKLOG/OIDC-02_auto_tenant_groups.md) → **v1.11.2**
  - Claims parsing (groups, roles)
  - Automatic tenant creation/mapping
  - Group sync (manual/scheduled)
  - Admin groups role assignment
  - **Оценка:** 8-12 часов

- [ ] **LDAP-01** LDAP/Active Directory Integration → [BACKLOG/LDAP-01_ldap_integration.md](BACKLOG/LDAP-01_ldap_integration.md) → **v1.11.3**
  - LDAP bind + search
  - User/group sync
  - UI для настройки через админку
  - TLS/LDAPS support
  - **Оценка:** 12-16 часов

- [ ] **AUDIT-01** Enhanced Audit Logging → [BACKLOG/AUDIT-01_enhanced_audit_logging.md](BACKLOG/AUDIT-01_enhanced_audit_logging.md) → **v1.11.4**
  - Security events (login, permissions)
  - Compliance logs
  - Query API с filters
  - Export to CSV
  - Retention policy
  - **Оценка:** 8-12 часов

- [ ] **RBAC-01** Custom Roles & Permissions → [BACKLOG/RBAC-01_custom_roles_permissions.md](BACKLOG/RBAC-01_custom_roles_permissions.md) → **v1.11.5**
  - Fine-grained permissions
  - Per-tenant role templates
  - Permission inheritance
  - RBAC middleware
  - Admin UI для role management
  - **Оценка:** 10-14 часов

---

## ⚡ Version 1.12.0 - Model Management Pro II

**Фокус:** Расширенное управление моделями (продолжение v1.9.0)  
**Приоритет:** MEDIUM  
**Оценка времени:** 28-38 часов  
**Статус:** 📋 Запланировано (0/4 задач)  
**Версионирование:** v1.12.1-v1.12.4

### Задачи

- [ ] **QUOTA-01** Usage Quotas System → [BACKLOG/QUOTA-01_usage_quotas.md](BACKLOG/QUOTA-01_usage_quotas.md) → **v1.12.1**
  - Per-tenant/per-user limits
  - Token-based quotas
  - Request quotas (daily, monthly)
  - Storage quotas
  - Quota monitoring & alerts
  - **Оценка:** 10-14 часов

- [ ] **MODEL-02** Model Preloading & Warming → [BACKLOG/MODEL-02_model_preloading.md](BACKLOG/MODEL-02_model_preloading.md) → **v1.12.2**
  - Keep models in memory
  - Configurable warm-up strategy
  - Health checks для loaded models
  - Automatic unload of unused models
  - **Оценка:** 6-8 часов

- [ ] **METRICS-01** Prometheus Metrics Export → [BACKLOG/METRICS-01_prometheus_export.md](BACKLOG/METRICS-01_prometheus_export.md) → **v1.12.3**
  - Metrics endpoint `/metrics`
  - Request counters, latency histograms
  - Model usage statistics
  - Quota metrics
  - Grafana dashboard template
  - **Оценка:** 4-6 часов

- [ ] **RATE-02** Advanced Rate Limiting → [BACKLOG/RATE-02_advanced_rate_limiting.md](BACKLOG/RATE-02_advanced_rate_limiting.md) → **v1.12.4**
  - Per tenant, per model limits
  - Sliding window algorithm
  - Rate limit headers в response
  - Burst allowance
  - **Оценка:** 8-10 часов

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
| **v1.10.0** | 5 | 38-52 | HIGH | 📋 Запланировано | Foundation для RAG |
| **v1.11.0** | 5 | 58-78 | HIGH | 📋 Запланировано | - |
| **v1.12.0** | 4 | 28-38 | MEDIUM | 📋 Запланировано | - |
| **v1.13.0** | 5 | 54-72 | HIGH | 📋 Запланировано | ⚠️ Требует v1.10.0 |
| **Future** | 9+ | TBD | DEFERRED | 📋 Отложено | - |
| **ИТОГО** | **22** | **198-268** | - | - | Экономия 6-8ч |

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

**Последнее обновление:** 2025-01-15  
**Версия Roadmap:** 2.1  
**Статус:** Active Development 🚀 (Version 1.9.0 ✅ Complete! → Planning v1.10.0-v1.13.0)
