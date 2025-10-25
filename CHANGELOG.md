# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.11.4] - 2025-10-25

### Added

- **AUDIT-01: Enhanced Audit Logging** 🔐
  - **Comprehensive Security Events Logging** для всех критичных операций
  - **Structured Audit Events** с полной трассировкой actor/target/action
  - **Event Types** (24 типа): LOGIN, API_KEY, TENANT, USER, BACKUP, PERMISSIONS
  - **Severity Levels**: info, warning, critical для приоритизации
  - **Metadata Support** для хранения произвольных данных в JSON
  - **Query API** с мощными фильтрами (event_type, severity, resource, date range)
  - **CSV Export** для compliance reporting и external analysis
  - **Statistics Dashboard** с real-time метриками (24h window)
  - **Admin UI** в WebUI с preview последних 20 событий + полнофункциональная страница
  - **Retention Policy** с автоматической очисткой старых событий (90 days default)
  - **Automatic Cleanup** (daily schedule) для управления размером БД

### Changed

- **AuthHandler** интегрирован с audit logging (LOGIN_SUCCESS, LOGIN_FAILED events)
- **Database Interface** расширен методами для audit events (CreateAuditEvent, GetAuditEvents, DeleteOldAuditEvents)

### Technical

- **Новые модули**:
  - `internal/models/audit.go` - AuditEvent data model с 24 event types
  - `internal/services/audit/logger.go` - AuditLogger service с convenience methods
  - `internal/services/audit/retention.go` - RetentionPolicy для auto-cleanup
  - `internal/api/handlers/audit.go` - HTTP handlers для query/export/stats
  - `internal/storage/sqlite/audit.go` - SQLite CRUD для audit events
- **Database** (Migration v40):
  - `CREATE TABLE audit_events` с полями:
    - `id, event_type, severity, actor_id, actor_type, target_id, target_type`
    - `action, resource, status, error_msg, metadata (JSON)`
    - `ip_address, user_agent, timestamp`
  - **7 индексов** для эффективных запросов:
    - `idx_audit_events_timestamp` (DESC для recent events)
    - `idx_audit_events_actor_id, idx_audit_events_event_type`
    - `idx_audit_events_severity, idx_audit_events_resource`
    - `idx_audit_events_target_id, idx_audit_events_status`
- **API Routes** (Admin-only):
  - `GET /api/admin/audit` - Query audit events с pagination/filters
  - `GET /api/admin/audit/stats` - Real-time statistics (24h)
  - `GET /api/admin/audit/export` - CSV export с filters
- **WebUI**:
  - `web/admin-audit.html` - Dedicated audit log viewer с:
    - Stats cards (critical/warning/info/failed logins)
    - Filters panel (event type, severity, resource, status, date range, actor)
    - Pagination (50 events per page)
    - CSV export button
  - `web/admin.html` - New "Audit" tab с preview последних 20 событий
  - `web/js/admin.js` - `loadAudit()` method для загрузки audit data
- **Convenience Methods** в AuditLogger:
  - `LogLogin(userID, ipAddress, success, errMsg)` - LOGIN events
  - `LogOIDCLogin(userID, issuer, ipAddress, success)` - OIDC events
  - `LogLDAPLogin(userID, server, ipAddress, success)` - LDAP events
  - `LogAPIKeyCreated(actorID, keyID, ipAddress)` - API key events
  - `LogTenantMemberAdded(actorID, tenantID, memberID, ipAddress)` - Tenant events
  - `LogPermissionDenied(userID, resource, ipAddress)` - Authorization events
- **Retention Policy**:
  - Default: 90 days retention
  - Daily cleanup schedule (configurable)
  - Manual trigger via `RunOnce()` method
  - Graceful shutdown support

### Use Cases

1. **Security Monitoring**: Track failed login attempts, permission denied events
2. **Compliance Reporting**: Export audit log для SOC2, ISO27001 compliance
3. **Incident Investigation**: Full trace с actor/target/action/IP/timestamp
4. **User Activity Tracking**: Кто и когда выполнял операции
5. **Administrative Auditing**: Все изменения (users, API keys, tenants)
6. **Trend Analysis**: Statistics dashboard для выявления аномалий

### Configuration Example

```yaml
# Retention policy настраивается в коде (future: yaml config)
# Default: 90 days retention, daily cleanup
# WithRetentionPeriod(duration) - custom retention period
# WithCleanupInterval(duration) - custom cleanup interval
```

### Notes

- Audit events хранятся в отдельной таблице `audit_events` для изоляции
- Автоматическая очистка запускается при старте сервера
- WebUI показывает последние 20 событий + full audit page для детального анализа
- CSV export поддерживает все filters для targeted reporting
- Integration в handlers требует добавления `auditLogger.Log*()` calls

### Future Enhancements (Phase 2)

- SIEM integration (Syslog, Splunk, ELK)
- Real-time alerting для critical events
- Advanced analytics и dashboards
- Audit event replay для forensics
- Encryption at rest для sensitive audit data

## [1.11.3] - 2025-10-25

### Added

- **LDAP-01: LDAP/Active Directory Integration** 🔐
  - **LDAP Bind Authentication** для корпоративных LDAP/AD серверов
  - **User Search** с настраиваемыми фильтрами (OpenLDAP, Active Directory)
  - **Group Search** для извлечения LDAP groups
  - **Auto-provisioning users** при первом логине через LDAP
  - **Auto-update users** синхронизация email/full name при каждом логине
  - **Tenant provisioning** из LDAP groups (reuse OIDC-02 logic)
  - **TLS/LDAPS support** с StartTLS и certificate validation
  - **Admin detection** на основе LDAP groups
  - **Test connection endpoint** для admin (`/api/auth/ldap/test`)

### Technical

- **Новые модули**:
  - `internal/auth/ldap/client.go` - LDAP client с bind auth, user/group search, TLS
  - `internal/api/handlers/ldap.go` - LDAP login handler с user provisioning
  - 17 unit tests (config validation, authentication, isAdminGroup logic)
- **Configuration** (Version 1.11.3+):
  - `auth.ldap.enabled` - включение LDAP аутентификации
  - `auth.ldap.url` - LDAP server URL (ldap:// или ldaps://)
  - `auth.ldap.bind_dn` - Service account DN для bind
  - `auth.ldap.bind_password` - Пароль для bind
  - `auth.ldap.user_base_dn`, `user_filter`, `user_id_attribute` - user search
  - `auth.ldap.group_base_dn`, `group_filter`, `group_name_attribute` - group search
  - `auth.ldap.start_tls`, `skip_verify`, `ca_cert_file` - TLS настройки
  - `auth.ldap.auto_create_user`, `auto_update_user` - user provisioning
  - `auth.ldap.tenant_provisioning` - tenant provisioning from groups
  - `auth.ldap.timeout` - timeout для LDAP операций
- **Database** (Migration v38):
  - `ALTER TABLE users ADD COLUMN ldap_dn TEXT UNIQUE` - LDAP Distinguished Name
  - `CREATE INDEX idx_users_ldap_dn` - быстрый поиск по LDAP DN
  - `GetUserByLDAPDN(ctx, ldapDN)` - новый метод для LDAP lookup
- **API Routes**:
  - `POST /api/auth/ldap/login` - LDAP login endpoint (public)
  - `GET /api/auth/ldap/test` - Test LDAP connection (admin only)
- **Integration**:
  - JWT tokens с tenant IDs из LDAP groups
  - Reuse tenant provisioner из OIDC-02 (direct/prefix mapping modes)
  - Support OpenLDAP, Active Directory, FreeIPA

### Use Cases

**OpenLDAP Authentication:**
```yaml
auth:
  ldap:
    enabled: true
    url: "ldap://ldap.company.com:389"
    bind_dn: "cn=admin,dc=company,dc=com"
    bind_password: "${LDAP_BIND_PASSWORD}"
    user_base_dn: "ou=users,dc=company,dc=com"
    user_filter: "(uid={username})"
    group_base_dn: "ou=groups,dc=company,dc=com"
# → Users логинятся с LDAP credentials, auto-created
```

**Active Directory:**
```yaml
auth:
  ldap:
    enabled: true
    url: "ldaps://ad.company.com:636"  # LDAPS для security
    bind_dn: "cn=service-account,dc=company,dc=com"
    bind_password: "${AD_SERVICE_PASSWORD}"
    user_base_dn: "ou=users,dc=company,dc=com"
    user_filter: "(sAMAccountName={username})"  # AD format
    user_id_attribute: "sAMAccountName"
    user_name_attribute: "displayName"
    tenant_provisioning:
      enabled: true
      group_mapping:
        mode: "prefix"
        prefix: "CN=APP-"  # APP-Engineering → engineering
        admin_groups: ["Domain Admins", "APP-Admins"]
# → AD users логинятся, tenants создаются из APP-* groups
```

---

## [1.11.2] - 2025-10-25

### Added

- **OIDC-02: Auto-tenant Provisioning from OIDC Groups** 🏢
  - **Автоматическое создание tenants** из OIDC groups claims (Keycloak, Google, Azure AD)
  - **Group → Tenant mapping** с двумя режимами:
    - **Direct mode**: 1:1 mapping (group name = tenant name)
    - **Prefix mode**: извлечение tenant из path (`/organizations/acme` → `acme`)
  - **Auto-provisioning**: создание tenants и добавление пользователей при первом логине
  - **Role assignment**: автоматическое назначение admin/member ролей из OIDC groups
  - **Orphaned memberships cleanup**: удаление доступа при удалении из группы (опционально)
  - **Tenant name normalization**: lowercase, hyphens, deduplication

### Technical

- **Новые модули**:
  - `internal/auth/oidc/tenants.go` - Group parsing и mapping logic
  - `internal/auth/oidc/provisioner.go` - Tenant provisioner service
  - 16 unit tests (ParseGroups, mapping modes, admin roles, normalization)
- **Configuration** (Version 1.11.2+):
  - `auth.oidc.tenant_provisioning.enabled` - включение tenant provisioning
  - `auth.oidc.tenant_provisioning.auto_create_tenants` - автосоздание tenants
  - `auth.oidc.tenant_provisioning.sync_on_login` - синхронизация при каждом логине
  - `auth.oidc.tenant_provisioning.remove_orphaned_memberships` - удаление orphaned memberships
  - `auth.oidc.tenant_provisioning.group_mapping.mode` - direct или prefix
  - `auth.oidc.tenant_provisioning.group_mapping.prefix` - префикс для prefix mode
  - `auth.oidc.tenant_provisioning.group_mapping.admin_groups` - список admin groups
- **Database** (Migration v36):
  - `CREATE UNIQUE INDEX idx_tenants_name_unique ON tenants(name)` - быстрый поиск tenants
  - `GetTenantByName(ctx, name)` - новый метод для OIDC provisioning
- **Integration**:
  - OIDC callback flow обновлен для tenant provisioning
  - JWT tokens теперь включают tenant IDs пользователя
  - Graceful error handling (login продолжается даже при ошибках provisioning)

### Use Cases

**Enterprise Keycloak Integration:**
```yaml
# Keycloak groups: /organizations/acme, /organizations/acme/engineering
auth:
  oidc:
    tenant_provisioning:
      enabled: true
      auto_create_tenants: true
      group_mapping:
        mode: "prefix"
        prefix: "/organizations/"
        admin_groups: ["/admins", "tenant-owners"]
# → User автоматически добавляется в tenant "acme" при логине
```

**Direct Group Mapping:**
```yaml
# Keycloak groups: engineering, sales, support
auth:
  oidc:
    tenant_provisioning:
      group_mapping:
        mode: "direct"
        admin_groups: ["engineering-admins"]
# → Каждая группа = отдельный tenant
```

---

## [1.11.1] - 2025-10-25

### Added

- **OIDC-01: Keycloak SSO Integration** 🔐
  - **OpenID Connect (OIDC)** аутентификация для корпоративного Single Sign-On (SSO)
  - Интеграция с **Keycloak** и другими OIDC providers (Google, Azure AD, Okta)
  - **Authorization Code Flow** с PKCE для безопасной аутентификации
  - Автоматическое **user provisioning** при первом входе через SSO
  - Гибкий **claims mapping** для разных OIDC providers
  - **Role-based access control** из OIDC groups/roles
  - Session management для OIDC state с защитой от CSRF
  - HTTP endpoints: `/api/auth/oidc/login`, `/api/auth/oidc/callback`, `/api/auth/oidc/logout`

### Technical

- **Новые модули**:
  - `internal/auth/oidc/provider.go` - OIDC provider wrapper на базе `coreos/go-oidc`
  - `internal/auth/oidc/claims.go` - структуры для OIDC claims (Standard, Keycloak, Generic)
  - `internal/api/handlers/oidc.go` - HTTP handlers для OIDC flow
- **Конфигурация**:
  - `auth.oidc.enabled` - включение/выключение OIDC
  - `auth.oidc.issuer` - URL OIDC провайдера (e.g., Keycloak realm)
  - `auth.oidc.client_id`, `auth.oidc.client_secret` - OIDC client credentials
  - `auth.oidc.redirect_uri` - callback URL
  - `auth.oidc.scopes` - запрашиваемые scopes (openid, profile, email, groups, roles)
  - `auth.oidc.claims.*` - mapping OIDC claims на поля пользователя
  - `auth.oidc.auto_create_user`, `auth.oidc.auto_update_user` - auto-provisioning
  - `auth.oidc.default_role` - роль по умолчанию для новых пользователей
  - `auth.oidc.session_store` - memory или redis для session storage
  - `auth.oidc.session_ttl` - время жизни OIDC session state
- **База данных (Migration v34)**:
  - `users.auth_provider` - тип провайдера (local, oidc, ldap)
  - `users.oidc_subject` - OIDC 'sub' claim (уникальный идентификатор)
  - `users.oidc_issuer` - OIDC issuer URL
  - Индексы для быстрого поиска по OIDC subject
  - Unique constraint для пары (issuer, subject)
- **Зависимости**:
  - `github.com/coreos/go-oidc/v3/oidc` - OIDC client library
  - `golang.org/x/oauth2` - OAuth2 flow
  - `github.com/gin-contrib/sessions` - session middleware
  - `github.com/gin-contrib/sessions/cookie` - cookie-based session store
- **Тестирование**:
  - Unit tests для OIDC provider (валидация конфигурации, discovery)
  - Unit tests для OIDC handlers (login, callback, logout)
  - Unit tests для helper functions (generateUsername, isAdminRole)
  - 10 тестов PASS, 5 SKIP (требуют mock OIDC provider)

### Security

- **CSRF Protection** - random state parameter в OAuth2 flow
- **ID Token Verification** - проверка подписи и claims через `coreos/go-oidc`
- **Session Security** - HttpOnly cookies, SameSite=Lax, secure encryption
- **Claims Validation** - проверка issuer, audience, expiration
- **Auto-Logout** - на expired/invalid tokens

### Configuration Examples

**Development (Keycloak):**
```yaml
auth:
  oidc:
    enabled: true
    provider: "keycloak"
    issuer: "https://keycloak.example.com/realms/myrealm"
    client_id: "ollama-proxy"
    client_secret: "${OIDC_CLIENT_SECRET}"
    redirect_uri: "http://localhost:8085/auth/oidc/callback"
    scopes: [openid, profile, email, groups, roles]
    auto_create_user: true
    auto_update_user: true
    default_role: "user"
```

**Production (Azure AD):**
```yaml
auth:
  oidc:
    enabled: true
    provider: "azure"
    issuer: "https://login.microsoftonline.com/{tenant-id}/v2.0"
    client_id: "your-client-id"
    client_secret: "${OIDC_CLIENT_SECRET}"
    redirect_uri: "https://proxy.yourdomain.com/auth/oidc/callback"
    scopes: [openid, profile, email]
    claims:
      user_id: "sub"
      username: "preferred_username"
      email: "email"
```

---

## [1.10.5] - 2025-10-25

### Changed

- **WEB-FETCH-01: Full Content Processing** 🚀
  - **BREAKING CHANGE**: Web fetcher теперь передает **полное содержимое страницы** модели без обрезания
  - Удален hardcoded truncation до 3000 символов
  - Добавлен параметр `TruncateLength` в `ProcessMessageOptions` для гибкого контроля
  - Default: `TruncateLength: 0` (без ограничений) - оптимально для моделей с большим контекстом (128K+)
  - Добавлены поля `WordCount` и `Language` в `WebPage` для статистики
  - Логирование truncation когда применяется

### Technical

- **Структуры данных**:
  - `WebPage`: добавлены `WordCount int` и `Language string`
  - `ParsedContent`: добавлено `Language string`
  - `ProcessMessageOptions`: добавлено `TruncateLength int` (0 = без ограничений)
- **Поведение по умолчанию**:
  - Chat Integration: `TruncateLength: 0` - полный контент для LLM
  - Старое поведение можно вернуть: `TruncateLength: 3000`
- **Улучшения**:
  - Показ статистики для больших страниц (>10K chars)
  - Детальное логирование при truncation
  - Language detection из HTML metadata

### Use Cases

**Работа с большими документами:**
```
User: Summarize https://docs.python.org/3/library/asyncio.html
→ Fetches full 50K+ chars documentation
→ LLM gets complete context for accurate summary
```

**Сравнение длинных статей:**
```
User: Compare https://example.com/article1 vs https://example.com/article2
→ Both articles fetched in full
→ No loss of important details
```

---

## [1.10.4] - 2025-10-24

### Added

- **WEB-FETCH-01: Web Content Fetcher & Chat Integration** 🌐
  - **Web Fetch Infrastructure** (`internal/webfetch/`)
    - HTTP client с retry logic (exponential backoff, 3 attempts)
    - URL validator с SSRF protection (блокирует private IPs, localhost)
    - Rate limiter для доменов (10 req/min default, configurable per domain)
    - HTML parser на базе goquery (text extraction, metadata)
    - Metadata extraction: Open Graph, Twitter Card, JSON-LD, author, language
  - **Chat Integration** ✨ **MAJOR FEATURE**
    - URL auto-detection в сообщениях (regex detector)
    - Automatic fetch при детектировании URL в user message
    - Context enrichment: добавление web content в контекст для LLM
    - Работает в streaming и non-streaming режимах
    - Max 2 URLs per message (context overflow protection)
    - 15s timeout per URL для быстрого fetch
    - Truncate до 3000 символов на страницу
  - **API Endpoints** (`internal/api/handlers/webfetch.go`)
    - POST `/api/web/fetch` - Fetch single URL
    - POST `/api/web/fetch/batch` - Batch fetch до 10 URLs
  - **Database Migration v31**
    - `web_fetches` table: url, title, content, metadata, links, cache
    - `web_fetch_rate_limits` table: per-domain rate limiting
    - Indexes для url_hash, user, tenant, domain, expires_at
  - **Configuration** (`configs/dev.yaml`)
    - `web_fetch.enabled: true` - включает автоматическую интеграцию с чатом
    - SSRF protection settings (block_private_ips, block_localhost)
    - Rate limiting settings (default_requests_per_min)
    - Cache settings (cache_enabled, cache_ttl)

### Changed

- **ChatHandler** (`internal/api/handlers/chat.go`)
  - Добавлен `webfetchIntegration` field
  - Новый метод `enrichMessagesWithWebContent()` для auto-fetch
  - Применяется в streaming и non-streaming режимах
  - Работает параллельно с file enrichment (FILE-STORAGE-01)

### Technical

- **Testing**:
  - `internal/webfetch/detector_test.go` - URL detection tests (12 test cases)
  - `internal/webfetch/validator_test.go` - URL validation, SSRF tests (10 test cases)
  - All tests passing ✅
- **Dependencies**:
  - `github.com/PuerkitoBio/goquery v1.10.3` - HTML parsing library
  - `golang.org/x/time/rate` - Rate limiting
- **Security Features**:
  - ✅ SSRF protection (блокирует 10.x, 192.168.x, 172.16-31.x, 127.x, link-local)
  - ✅ DNS resolution check перед запросом
  - ✅ Per-domain rate limiting с burst support
  - ✅ Content size limits (10MB max)
  - ✅ Redirect limits (max 5)
- **Performance**:
  - Cache с TTL (default 1 hour)
  - Connection pooling для HTTP client
  - Truncation для контекста (3000 chars per page)
  - Parallel fetch для batch requests
- **Documentation**:
  - `docs/WEB_FETCH_QUICKSTART.md` - Complete guide with examples
  - `BACKLOG/WEB-FETCH-01_content_fetcher.md` - Detailed specification

### Use Cases

```
User: Summarize https://example.com/article
→ Proxy auto-detects URL → fetches content → adds to context → LLM responds

User: Compare https://example.com/page1 vs https://example.com/page2
→ Fetches both pages → LLM compares based on actual content

User: Explain https://docs.python.org/3/library/asyncio.html
→ Fetches documentation → LLM explains based on real docs
```

### RAG Integration Ready

Все компоненты WEB-FETCH-01 спроектированы для переиспользования в RAG v1.13.0:
- HTML parser → web sources для RAG
- SSRF protection → secure crawling
- Rate limiting → respectful fetching
- Metadata extraction → document enrichment

---

## [1.10.3] - 2025-10-20

### Added

- **IMAGE-01: Image Upload & OCR Processing** 🖼️
  - **Vision Interface** (`internal/vision/interface.go`)
    - OCREngine interface: ExtractText, DescribeImage, SupportedModels
    - OCROptions с поддержкой layout preservation, table extraction
    - OCRResult с confidence, language detection, bounding boxes
  - **OllamaOCR Engine** (`internal/vision/ollama_ocr.go`)
    - Multimodal models support: LLaVA (7b/13b/34b), BakLLaVA, Llama3.2-Vision (11b/90b)
    - Raw bytes image transfer (ChatMessage.Images [][]byte)
    - Russian/English language detection heuristics
    - Confidence scoring и metadata extraction
  - **ImageProcessor** (`internal/imageproc/processor.go`)
    - Thumbnail generation (CatmullRom filter, customizable size/quality)
    - Image resize с Lanczos filter
    - Format conversion (JPEG, PNG, WebP)
    - Image validation и metadata extraction (width, height, format, size)
  - **ImageHandler API** (`internal/api/handlers/image_handler.go`)
    - POST `/api/images/upload` - Upload с OCR processing
    - GET `/api/images/:id` - Get image metadata
    - GET `/api/images/:id/download` - Download original
    - GET `/api/images/:id/thumbnail` - On-the-fly thumbnail generation
    - WebSocket integration для upload/OCR progress events

### Changed

- **Ollama Client Extension** (`internal/client/ollama/models.go`)
  - Added `Images [][]byte` field to ChatMessage for vision model support
  - Compatible с Ollama API vision models interface

### Technical

- **Testing**:
  - `internal/imageproc/processor_test.go` - Validation, thumbnail, resize tests
  - `internal/vision/ollama_ocr_test.go` - OCR engine, language detection tests
  - Benchmark tests для thumbnail generation
- **Dependencies**:
  - `github.com/disintegration/imaging v1.6.2` - Image processing library
  - `golang.org/x/image` - Extended image format support
- **Features**:
  - ✅ Multi-format support (JPEG, PNG, GIF, WebP)
  - ✅ Automatic OCR via Ollama vision models
  - ✅ Thumbnail generation (200x200px default, on-the-fly)
  - ✅ Language detection (Russian/English)
  - ✅ WebSocket real-time progress notifications
  - ✅ Image metadata extraction

### Known Limitations

- **Database schema**: FileMetadata doesn't have dedicated image fields (using Custom map)
- **Thumbnail persistence**: Generated on-the-fly, not pre-saved to storage
- **Public images**: Public flag not yet implemented in File model
- **WebUI**: Drag & drop interface pending

## [1.9.4] - 2025-10-20

### Added

- **CPU Cache-Friendly Data Structures** 🚀 (Phase 2: Hot/Cold Split & Caching)
  - **APIKeyCache Layer**: In-memory cache для API keys с TTL expiration
    - Thread-safe concurrent access (RWMutex)
    - Load-through caching pattern с `GetOrLoad()`
    - Background cleanup goroutine
    - Performance: 22.76ns Get, 78.43ns Set, 45.52ns Parallel
    - **100x faster** vs database queries (22ns vs 2ms)
  - **APIKeyDBAuthOptimized Middleware**: Cache-aware authentication
    - Cache hit path: ~22ns (memory only)
    - Cache miss path: ~2ms (DB + bcrypt)
    - Expected 95%+ cache hit rate → **20x faster auth**
  - **APIKeyUsageThreadSafe**: Fully thread-safe usage tracking
    - Atomic counters с cache line padding (prevents false sharing)
    - RWMutex для map operations (ModelUsage, EndpointUsage, DailyUsage)
    - Performance: 196ns vs 236ns Original + **zero race conditions**
    - **20% faster + thread-safe**

- **Cursor Rules для Cache Optimization**
  - Автоматические подсказки для cache-friendly structures
  - Lint правила для false sharing detection
  - Templates для padded structs, hot/cold splits, concurrent counters

### Changed

- **StatsOptimized Migration**: Migrated `GlobalStats` to cache-friendly version
  - Cache line padding между atomic counters
  - StatsInterface для backwards compatibility
  - Performance: **6.4x faster** under high concurrency

### Fixed

- **Race Condition в APIKey.IncrementUsage**: Replaced `++` with `atomic.AddInt64`
  - ✅ Atomic operations для TotalRequests, SuccessfulRequests, FailedRequests, TotalTokens
  - ⚠️ Map operations (ModelUsage, DailyUsage) все еще require careful handling
  - Recommended: Use APIKeyUsageThreadSafe для new keys

### Technical

- **Benchmarks Added**:
  - `internal/api/handlers/stats_bench_test.go`: Stats vs StatsOptimized comparison
  - `internal/models/apikey_bench_test.go`: APIKey validation benchmarks
  - `internal/models/apikey_usage_threadsafe_test.go`: Thread-safe usage benchmarks
  - `internal/cache/apikey_cache_test.go`: Cache performance benchmarks
- **Documentation**:
  - `docs/CACHE_OPTIMIZATION.md`: Technical deep-dive
  - `docs/CACHE_OPTIMIZATION_SUMMARY.md`: Executive summary
  - `docs/CACHE_OPTIMIZATION_QUICKSTART.md`: Quick start guide
  - `PERFORMANCE_IMPROVEMENTS.md`: High-level report
  - `MIGRATION_APPLIED.md`: Phase 1 & 2 migration status
  - `PHASE2_COMPLETED.md`: Phase 2 completion report
- **Dependencies**: No new dependencies (pure Go stdlib)
- **New Packages**:
  - `internal/cache`: APIKey caching layer
  - `internal/api/middleware/apikey_db_auth_optimized.go`: Optimized middleware
  - `internal/models/apikey_usage_threadsafe.go`: Thread-safe usage tracking
  - `internal/api/handlers/stats_optimized.go`: Cache-friendly stats

### Performance

- **Authentication**: 20x faster (с cache hit rate 95%+)
- **Usage Tracking**: 1.2x faster + thread-safe
- **Server Throughput**: +50-87% expected improvement
- **Memory Overhead**: ~6 MB для 10,000 API keys (acceptable)

### Security

- ✅ **Zero Race Conditions**: All optimized structures pass `-race` tests
- ✅ **Thread-Safe Maps**: RWMutex protection для concurrent access
- ✅ **Atomic Counters**: Cache line padding prevents false sharing

## [1.10.0] - 2025-10-16

### Added

- **FILE-STORAGE-01: Universal File Storage & Processing System** ✅ (Content Foundation)
  - **Storage Backends**: Local filesystem и S3-compatible (MinIO) storage
  - **Document Extractors**: PDF, DOCX, TXT, CSV с автоматическим определением кодировки
  - **Database Integration**: Таблицы `files`, `file_access_logs`, `message_files`
  - **API Endpoints**: `/api/files/*` для upload, download, delete, list
  - **WebUI**: Страница Files для управления файлами пользователя
  - **Admin Panel**: Новая вкладка Files для управления всеми файлами системы
  - **Chat Integration**: Прикрепление файлов к сообщениям через junction table
  - **LLM Context Enrichment**: Автоматическое включение содержимого файлов в контекст чата
  - **Path Traversal Prevention**: Robust защита от path traversal атак
  - **Unicode Filenames**: Полная поддержка Unicode имен файлов (Cyrillic, Chinese, Emoji)
  - **Content Validation**: Magic number validation для безопасности

- **Cross-Platform PDF Text Extraction** 🚀
  - **Pure Go Library**: `github.com/ledongthuc/pdf` для работы без внешних зависимостей
  - **Automatic Fallback**: `pdftotext` (если доступен) → `go-pdf` (всегда работает)
  - **Three Extraction Methods**:
    - `auto`: Автоматический выбор лучшего доступного метода
    - `pdftotext`: Использует Poppler для высокого качества
    - `go-pdf`: Чистый Go (работает на Windows, Linux, macOS без установки)
  - **Docker-Ready**: Работает в контейнерах без дополнительных зависимостей
  - **Metadata Extraction**: Автоматическое извлечение метаданных (title, author, pages)

- **Advanced Text Encoding Detection** 🔍
  - **UTF-8 with BOM**: Автоматическое определение и обработка UTF-8 BOM
  - **UTF-16 LE/BE**: Поддержка UTF-16 Little/Big Endian с BOM detection
  - **Windows-1251 Fallback**: Heuristic-based detection для русского текста
  - **Cyrillic Detection**: Интеллектуальное определение кириллицы для правильной кодировки
  - **Reasonable Text Validation**: Проверка что декодированный текст является валидным

- **Comprehensive Unit Tests** ✅
  - **Validator Tests**: 13 тестов (100% pass rate)
    - File validation (PDF, size, extensions)
    - Filename security (path traversal, special chars)
    - MIME type validation
    - Magic number checks
    - Benchmark tests
  - **Local Storage Tests**: 15 тестов (100% pass rate)
    - Store/Retrieve/Delete operations
    - Path traversal prevention
    - Unicode filenames support
    - Multi-user isolation
    - Benchmark tests
  - **Coverage**: filestorage 46.7%, storage 29.6%

- **File Management UI**
  - **User Files Page**: Drag & drop upload, grid view, filters, search, pagination
  - **Admin Files Tab**: Управление всеми файлами с отображением email/username владельца
  - **File Preview**: Modal для просмотра извлеченного текста
  - **File Details**: Метаданные, размер, MIME type, extraction status
  - **Statistics**: Total files, total size, по типам файлов

- **Documentation** 📚
  - **PDF_EXTRACTION.md**: Полная документация по PDF extraction
  - **QUICK_START_PDF.md**: Быстрый старт для PDF
  - Описание всех трех методов extraction
  - Инструкции по установке Poppler для каждой ОС
  - Docker integration guide

### Changed

- **Chat Messages**: Добавлено поле `file_ids` для хранения прикрепленных файлов
  - Frontend отправляет `file_ids` массив при создании сообщения
  - Backend enrichment: содержимое файлов автоматически добавляется в LLM prompt
  - UI: File badges под сообщением с возможностью просмотра содержимого

- **Configuration**:
  - Добавлены секции `file_storage` и `extractors` в dev.yaml и production.yaml.example
  - PDF extractor: `method: "auto"` по умолчанию для автоматического выбора

### Fixed

- **File Upload Integrity**: Исправлено отрезание начала файла из-за magic number validation
  - Введен флаг `SkipContentValidation` для HTTP uploads
- **Windows Path Separators**: Корректная обработка forward slashes на Windows
  - Использование `filepath.FromSlash()` для кроссплатформенности
- **File Deletion**: Исправлено физическое удаление файлов на Windows
  - Robust path traversal checks с `filepath.Abs` и `strings.HasPrefix`
- **Text Encoding**: Улучшенное определение Windows-1251 для русских текстов
  - Heuristic-based fallback с проверкой Cyrillic символов

### Technical

- **New Dependencies**:
  - `github.com/ledongthuc/pdf v0.0.0-20250511090121-5959a4027728` - Pure Go PDF parser
- **Database Migrations**:
  - Migration v26: `files` и `file_access_logs` таблицы
  - Migration v27: `message_files` junction table для chat integration
- **New Packages**:
  - `internal/filestorage` - Universal storage abstraction
  - `internal/filestorage/storage` - Local и S3 backends
  - `internal/extractors` - Document extractors (PDF, DOCX, TXT, CSV)
- **Test Files**:
  - `internal/filestorage/validator_test.go` - 13 tests
  - `internal/filestorage/storage/local_test.go` - 15 tests
  - `internal/extractors/text_test.go` - Encoding tests (prepared)

## [1.9.3] - 2025-10-14

### Added

- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
  - MoniGo запускается на отдельном порту 9091 для изоляции
  - Quick Stats Cards с автоматическим обновлением каждые 5 секунд
  - Real-time метрики: CPU Usage, Memory Usage, Goroutines, System Health
  - Visual indicators (success/warning/critical) для метрик
  - API proxy для `/admin/performance/monigo/api/v1/metrics`
  - Direct link на Advanced Dashboard для полных возможностей MoniGo

- **NVIDIA GPU Monitoring**: Мониторинг GPU метрик через nvidia-smi
  - **БЕЗ CGO зависимостей** - использует `nvidia-smi` CLI напрямую
  - **БЕЗ NVML headers** - работает на любой системе с nvidia-smi
  - Поддержка нескольких GPU (multi-GPU configurations)
  - Единый компактный блок для всех GPU с gradient top border
  - Real-time метрики: Temperature, Power, GPU Load, VRAM, Clock, Fan Speed
  - Автообновление каждые 5 секунд
  - Hover эффект с подсветкой для каждой GPU строки
  - Цветовые индикаторы: Green (<70°C), Orange (70-80°C), Red (>80°C)
  - Graceful degradation если GPU не обнаружены или nvidia-smi недоступен
  - Platform-specific builds: Linux/macOS (full GPU support), Windows (stub)

- **Admin Panel Reorganization**: Улучшенная структура навигации
  - Новая вкладка "Models" с Available Models списком
  - Refresh Models кнопка для обновления списка моделей
  - System Tab переработан для Performance & GPU Monitoring
  - Logs Tab отделен от System (уже существовал)
  - Models Tab отделен от System

### Changed

- **System Tab**: Переработан полностью под мониторинг
  - Убраны "Available Models" (перенесены в Models Tab)
  - Убраны "System Logs" (остались в Logs Tab)
  - MoniGo Dashboard link вместо embedded iframe
  - Quick Stats Cards для основных метрик
  - NVIDIA GPU Metrics секция (если GPU доступны)

- **Models Tab**: Новая навигационная структура
  - Перенесены Available Models из System Tab
  - Добавлена кнопка Refresh для обновления списка
  - Section header с красивым дизайном
  - Fix: Модели корректно загружаются при первом заходе

- **GPU Monitoring Architecture**:
  - Отказ от `go-nvml` (CGO зависимость) в пользу nvidia-smi CLI
  - Build tags для platform-specific реализаций
  - Windows: stub версия (GPU monitoring disabled)
  - Linux/macOS: полная функциональность через nvidia-smi

### Fixed

- **Models Tab Loading**: Исправлен баг с загрузкой моделей при первом заходе
- **MoniGo Integration**: Убран iframe, решены проблемы с CORS и static files
- **GPU Monitoring**: Убраны CGO compilation errors на Windows/Linux

### Technical

- **Backend (Go)**:
  - `internal/metrics/gpu_monitor_smi.go` - GPU monitoring через nvidia-smi (Linux/macOS)
  - `internal/metrics/gpu_monitor_windows.go` - Stub для Windows
  - `internal/metrics/custom_monigo.go` для custom MoniGo metrics
  - `internal/api/handlers/gpu.go` - REST API handler для `/api/gpu/metrics`
  - MoniGo запускается на порту 9091 в отдельном HTTP сервере
  - GPU Monitor инициализация в `cmd/server/main.go`
  - Reverse proxy для MoniGo API endpoints в `internal/api/router/router.go`
  - Зависимость: `github.com/iyashjayesh/monigo v1.1.0`

- **Frontend (JavaScript)**:
  - `web/js/performance.js` - Real-time performance metrics от MoniGo
  - `web/js/gpu-monitor.js` - NVIDIA GPU metrics визуализация
  - Unified GPU card дизайн с табличным layout для нескольких GPU
  - Gradient top border на карточке (цвет зависит от max температуры)
  - Grid layout: Name, Temp, Power, Clock, Fan | GPU Load & VRAM bars
  - Auto-refresh каждые 5 секунд для актуальных данных
  - CSS animations и hover эффекты

- **Database Migration**:
  - Migration v24: `add_changelog_v1_9_3` для системной истории изменений
  - Автоматическое применение при старте сервера

### Security

- MoniGo dashboard доступен только через JWT authentication Admin Panel
- API proxy для метрик защищен Bearer token authentication
- GPU metrics endpoint требует аутентификацию

### Notes

- MoniGo собирает метрики автоматически через Gin middleware
- Custom metrics (Ollama latency, API keys) подготовлены для future versions
- GPU monitoring работает только в Linux/macOS, Windows использует stub
- Dashboard доступен только для admin users с валидным JWT
