# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.4.9] - Unreleased

### Planned

- **Go 1.18-1.25 Modernization** 🚀:
  - **Generic Pointer Helpers**:
    - New `internal/utils/ptr.go` with `Ptr[T]()` and `Value[T]()` functions
    - Replace 200+ lines of duplicated `stringPtr`, `intPtr`, `float64Ptr` helpers
    - Type-safe pointer operations across entire codebase
  - **Standard Library Integration**:
    - `slices` package (Go 1.21+): Replace manual loops with `slices.Contains`, `slices.DeleteFunc`, `slices.Index`
    - `maps` package (Go 1.21+): Safe operations with `maps.Clone`, `maps.Copy` for configs and metadata
    - Estimated code reduction: ~150 lines
  - **Generic API Response Wrapper**:
    - Type-safe `APIResponse[T any]` for all API handlers
    - Consistent error handling across endpoints
    - Better type inference and IDE support
  - **Custom Iterators** (Go 1.23):
    - Database pagination with `range over func`
    - RAG chunk processing iterator
    - Cleaner, more idiomatic pagination code
  - **Generic Type Aliases** (Go 1.24):
    - `ConfigMap[T]` for typed configuration maps
    - `Metadata[T]` for model and RAG metadata
    - Improved type safety and code clarity
  - **Generic Vector Operations**:
    - `Vector[T ~float32 | ~float64]` for RAG embeddings
    - Type-safe similarity calculations
    - Support for both float32 and float64 precision
  - **Go 1.25 New Features**:
    - `sync.WaitGroup.Go()` - convenient goroutine spawning pattern
    - `runtime/trace.FlightRecorder` - lightweight execution tracing for rare events
    - `hash.Cloner` interface - implement for all custom hash implementations
    - `net/http.CrossOriginProtection` - CSRF protection middleware
    - `testing.T.Attr()` - test attributes for better observability
    - Container-aware `GOMAXPROCS` - automatic CPU limit detection in Kubernetes
    - Experimental GreenTea GC (`GOEXPERIMENT=greenteagc`) - 10-40% GC overhead reduction
    - `go vet` analyzers: `waitgroup` (misplaced Add calls), `hostport` (IPv6 compatibility)

### Technical

- **Estimated Impact**:
  - Code reduction: ~500 lines
  - Type safety improvement: High
  - Performance: Minimal overhead (generics are zero-cost at runtime)
  - Readability: Very High improvement
- **Compatibility**: Go 1.21+ required for full feature set
- **Migration**: Incremental rollout, backward compatible

## [2.4.8] - 2025-11-04

### Added

- **Embeddings & Vector Search** 🧮:
  - Ollama embeddings integration (`mxbai-embed-large`, `nomic-embed-text`)
  - pgvector store for similarity search with HNSW index
  - Automatic index creation with configurable parameters (m=16, ef_construction=64)
  - Connection string support for direct PostgreSQL connection
  - Configurable dimensions (768, 1024) and distance metrics (cosine, l2, inner_product)
  - **RAG Worker embeddings generation**:
    - Batch embeddings for all RAG data sources (API, DB, Web)
    - Automatic vector storage in pgvector after chunk creation
    - Graceful fallback to simple mode if embedder not configured
    - `generateAndStoreEmbeddings()` method for efficient processing
  - Full vs Simple mode: seamless operation with or without embeddings
  - RAG Orchestrator integrates embedder and vector store for enhanced search

- **RAG Worker Full Implementation** 🤖:
  - Complete API Sync implementation (`internal/rag/worker/api_sync.go`)
    - REST API data synchronization (GET/POST methods)
    - JSON and plain text response parsing
    - Configurable `data_path` extraction (e.g., "data.items")
    - Custom `text_fields` selection for chunking
    - Batch processing for array responses
  - Complete Database Sync implementation (`internal/rag/worker/db_sync.go`)
    - PostgreSQL and MySQL database connectivity
    - SQL query execution with full row scanning
    - Dynamic connection string building from config + encrypted credentials
    - Automatic document and chunk creation from query results
  - Complete Web Scraping implementation (`internal/rag/worker/web_scrape.go`)
    - HTML page scraping with User-Agent spoofing
    - Text extraction from HTML (script/style tag removal)
    - HTML entities decoding (&nbsp;, &lt;, &mdash;, etc.)
    - Whitespace cleanup and normalization
    - Multiple URLs batch processing
    - Partial success handling (some URLs may fail)
  - **Chunking Strategies**:
    - **Fixed**: Fixed-size chunks with configurable overlap
    - **Sentence**: Smart sentence-based chunking (punctuation detection)
    - **Paragraph**: Paragraph-based chunking (double newline split)
  - Token counting: Approximate `tokens = len(text) / 4`
  - RAG Worker logging to separate file: `logs/rag-worker.log`

- **UI Improvements** 🎨:
  - **RAG Data Sources Checkboxes**:
    - Replaced confusing multi-select dropdown with intuitive checkboxes
    - Each source shows name and type badge (api/database/web)
    - Hover effects and visual feedback
    - No more "Hold Ctrl/Cmd" requirement
    - Scrollable list for many sources (max-height: 150px)
  - **Thinking Spoiler for Reasoning Models**:
    - Compact single-line spoiler with expand/collapse toggle (DeepSeek-R1, o1-preview)
    - Dark theme styling (blue accent, #1a1a2e background)
    - `<think>` and `<reasoning>` tag support in streaming chunks
    - Frontend extraction and rendering without breaking markdown

### Changed

- **RAG Job Processing** 🔧:
  - Job queue polling interval: 5 seconds
  - Max retry attempts: 3
  - Job locking: 5 minutes timeout
  - Status flow: `pending` → `processing` → `completed`/`failed`
  - Automatic expired job unlocking

- **RAG Data Source Configuration**:
  - **API Sources**: `url`, `method`, `headers`, `data_path`, `text_fields`
  - **Database Sources**: `connection_string`, `query`, `database_type`
  - **Web Sources**: `url` or `urls` (array)
  - **Indexing Config**: `chunk_strategy`, `max_chunk_size` (default: 500), `chunk_overlap` (default: 50)

### Fixed

- **RAG Data Source Update Config Loss** 🔧:
  - Fixed critical bug where editing RAG source would lose advanced config fields
  - **Problem**: Frontend rebuilt `config` and `indexing_config` from scratch, losing:
    - API sources: `data_path`, `text_fields`, custom `headers`
    - Database sources: `connection_string`, custom `database_type`
    - All sources: `chunk_strategy` in indexing_config
  - **Solution**: 
    - `buildConfig()` now starts with existing config when editing: `{ ...this.currentSource.config }`
    - `indexing_config` preserves existing fields: `{ ...existingIndexingConfig, chunk_size, chunk_overlap }`
    - Only form-visible fields are updated, everything else remains intact
  - **Impact**: Config changes now properly persist between edits
  - File modified: `web/js/rag-sources.js`

- **RAG Sync Deduplication** 🔄:
  - Implemented `DeleteChunksBySource` to prevent data duplication on repeated syncs
  - Each sync now deletes old chunks/vectors/documents before creating new ones
  - Atomic transaction: vectors → chunks → documents cleanup
  - Applied to all sync types: API, Database, Web Scraping
  - Prevents "chunks growing infinitely" issue
  - Implementation:
    - `internal/storage/database.go`: Added interface method
    - `internal/storage/postgresql/rag.go`: PostgreSQL implementation with transaction
    - `internal/storage/sqlite/rag.go`: SQLite implementation with transaction
    - `internal/rag/worker/api_sync.go`: Delete before sync
    - `internal/rag/worker/db_sync.go`: Delete before sync
    - `internal/rag/worker/web_scrape.go`: Delete before sync

- **RAG Chunk ID Collisions** 🔧:
  - Changed chunk ID generation from `timestamp+random` to **UUID v4**
  - Fixes `duplicate key value violates unique constraint "rag_chunks_pkey"` error
  - Prevents collisions during high-speed chunk creation (100+ chunks/sec)
  - Applies to API sync, DB sync, and web scraping workers
- **Duplicate Method Declarations**:
  - Removed duplicate `generateDocumentID`, `generateChunkID`, `randInt` from `db_sync.go`
  - Centralized ID generation in `api_sync.go`
- **Compilation Errors**:
  - Fixed method redeclaration conflicts
  - Removed duplicate `DocumentProcessor` implementation (kept `pipeline.go` version)

### Technical

- **File Structure**:
  ```
  internal/rag/worker/
    ├── worker.go           # Main worker loop and job dispatching
    ├── api_sync.go         # API synchronization logic
    ├── db_sync.go          # Database query synchronization
    └── web_scrape.go       # Web scraping logic
  
  internal/rag/processor/
    ├── pipeline.go         # Document processing pipeline (existing)
    └── worker.go           # Worker pool for batch processing
  ```

- **Documentation**:
  - New comprehensive guide: `docs/RAG_WORKER_GUIDE.md`
  - API endpoints documentation
  - Troubleshooting guide
  - Configuration examples for all source types

## [2.4.7] - 2025-11-03

### Added

- **PostgreSQL Full Support** 🐘:
  - Complete PostgreSQL database backend implementation
  - All 12 CRUD modules ported from SQLite (8,000+ lines of code)
  - 142 migration files (71 up + 71 down) with rollback support
  - Automatic placeholder conversion (? → $1, $2, $3...)
  - Production-ready with full feature parity to SQLite

### Changed

- **Migration System Refactoring** 🔧:
  - Split monolithic `sqlite.go` (4,560 lines) into separate SQL files
  - Reduced `sqlite.go` to ~300 lines (connection management only)
  - New structure: `internal/storage/{sqlite,postgresql}/migrations/`
  - File naming: `001_initial_schema.up.sql` + `001_initial_schema.down.sql`
  - Automatic loader via Go 1.16+ `embed.FS`

### Fixed

- **PostgreSQL Migration Syntax** 🔧:
  - Converted 47 changelog migrations from SQLite `INSERT OR REPLACE` to PostgreSQL `INSERT ... ON CONFLICT`
  - Fixed SQL syntax errors in `001_initial_schema.up.sql` (comments, DEFAULT values)
  - Fixed cross-platform path handling (`path.Join` for `embed.FS`, `filepath.Join` for OS filesystem)
  - Fixed `build.ps1` PowerShell script (emoji parser errors, VCS status handling)
  - All PostgreSQL CRUD operations now compile successfully
  - Converted all SQLite-specific syntax to PostgreSQL:
    - `DATETIME` → `TIMESTAMP`
    - `INTEGER PRIMARY KEY AUTOINCREMENT` → `BIGSERIAL PRIMARY KEY`
    - `randomblob()` → `gen_random_bytes()` (requires `pgcrypto` extension)
    - SQLite triggers → PL/pgSQL trigger functions
  - Added required PostgreSQL extensions: `pgcrypto` (for UUID generation), `vector` (for RAG/embeddings)

### Technical

- **Migration Loader**:
  - Regex-based version extraction from filenames
  - Automatic sorting by version number
  - Embedded FS for zero-config deployment
  - Rollback support with `.down.sql` files

- **PostgreSQL Specifics**:
  - JSONB for all JSON fields (better performance)
  - Native UUID support via `gen_random_uuid()`
  - Proper boolean types (vs INTEGER in SQLite)
  - Timestamp with timezone support
  - Full-text search capabilities (future)

- **Code Quality**:
  - 100% method coverage across all CRUD operations
  - Type-safe placeholder conversion
  - Consistent error handling patterns
  - Thread-safe transaction support

### Files Modified

- `VERSION`: 2.4.6 → 2.4.7
- `internal/storage/postgresql/`: 12 CRUD files created
- `internal/storage/postgresql/migrations/`: 142 SQL files
- `internal/storage/sqlite/migrations/`: 142 SQL files
- `Roadmap.MD`: REFACTOR-01 marked as completed

## [2.4.6] - 2025-10-29

### Fixed

- **CRITICAL: WebUI Modal Overlay Conflict** 🐛:
  - **Issue**: On "My Devices" page, all buttons become unclickable (including top menu)
  - **Root Cause**: CSS class name conflict between:
    - Device details modal (`.modal-overlay` in `profile-devices.html`)
    - Confirmation modals (`.modal-overlay` in `notifications.js`)
  - **Problem**: When confirmation modal closes, it removes `modal-show` class but device details CSS has `display: flex` always visible, leaving invisible overlay blocking clicks
  - **Solution**: Renamed device details modal class from `.modal-overlay` to `.device-modal-overlay`
  - **Impact**: All buttons and navigation now work correctly after using confirmation modals
  - **Files Modified**:
    - `web/profile-devices.html`: Renamed CSS class `.modal-overlay` → `.device-modal-overlay`
    - `web/js/profile-devices.js`: Updated `openDeviceDetails()` to use new class name
  - **Testing**: Verified on "My Devices" page with delete/rename confirmations

### Technical

- **CSS Specificity Issue**: Global class names like `.modal-overlay` should be avoided or properly scoped
- **Best Practice**: Use component-specific class names (e.g., `.device-modal-overlay`, `.tenant-modal-overlay`)
- **Future**: Consider CSS modules or scoped styles to prevent conflicts

## [2.4.5] - 2025-10-28

### Fixed

- **CRITICAL: WebSocket Panic Fix** 🐛:
  - **Issue**: Server panic "send on closed channel" when client disconnects during streaming
  - **Root Cause**: Goroutine continues streaming after client's `Send` channel closed
  - **Solution**: 
    - Added panic recovery in `sendMessage()` with `defer recover()`
    - Added 5-second timeout to detect disconnected clients
    - Stop streaming immediately on send error (graceful goroutine exit)
    - Ignore errors on final "done" message (client may be disconnected)
  - **Impact**: Server now handles client disconnections gracefully without crashes
  - **Testing**: Verified with client disconnect during active streaming

### Technical

- **File Modified**: `internal/websocket/chat_handler.go`
  - `sendMessage()`:
    - Added `defer recover()` to catch panic from closed channel
    - Added `select` with `time.After(5s)` timeout
    - Returns error instead of panicking
  - `processStreamingResponse()`:
    - Check `sendMessage()` error on every chunk
    - Exit goroutine immediately on error (stop streaming)
    - Log warning with context (request_id, chunks_sent, client_id)
- **Design Patterns**:
  - Panic recovery pattern for channel operations
  - Timeout pattern for dead client detection
  - Graceful goroutine shutdown on errors
- **Documentation**: Added `WEBSOCKET_PANIC_FIX.md` with detailed analysis

## [2.4.4] - 2025-10-28

### Added

- **Desktop-Specific WebUI Features** (DESKTOP-04): Enhanced device management experience
  - **Device Details Modal** 🔍:
    - Full-screen modal с полной информацией об устройстве
    - Клик на device card или icon открывает modal
    - Display: OS, version, hostname, API key ID, first/last seen dates
    - Usage statistics (total requests, tokens) если доступны
    - Action buttons: Rename, Revoke Access (если не current device), Close
    - Responsive design с smooth animations (fadeIn, slideIn)
  - **Bulk Device Revoke** 📦:
    - Checkbox selection на всех device cards (кроме current device)
    - Bulk actions bar появляется при selection (fixed bottom, slide up animation)
    - Selected count display (e.g. "3 devices selected")
    - Bulk revoke button с confirmation modal
    - Protection: нельзя bulk revoke current device
    - Cancel button для сброса selection
    - Success/error toast notifications после bulk operation
  - **Enhanced UI/UX**:
    - "View Details" button на каждой device card
    - event.stopPropagation() для всех кнопок (предотвращение случайного открытия modal)
    - CSS animations: fadeIn, slideIn, slideUp
    - Modal overlay с click-outside-to-close
    - Grid layout для device details (2 columns)
    - Color-coded badges: active/inactive/expired

### Changed

- **Device Cards Layout**: Добавлен checkbox слева (18x18px) для bulk selection
- **Device Icon**: Стал кликабельным для открытия details modal
- **Device Name/Hostname**: Стали кликабельными для открытия details modal
- **Actions**: Добавлена кнопка "View Details" (secondary style)

### Technical

- **Frontend (HTML/CSS/JS)**:
  - `web/profile-devices.html` - Bulk actions bar, modal CSS (animations: fadeIn, slideIn, slideUp)
  - `web/js/profile-devices.js` - Device Details Modal (180+ lines new code)
    - `openDeviceDetails(deviceId)` - открывает modal с device info
    - `renderDeviceDetailsHTML(device)` - генерирует HTML для modal
    - `enableBulkSelection()` - включает bulk selection checkboxes
    - `updateBulkActionsBar()` - обновляет bulk actions bar visibility
    - `getSelectedDeviceIds()` - возвращает массив selected device IDs
    - `bulkRevokeDevices()` - массовое удаление с confirmation
    - `clearSelection()` - сброс всех checkboxes
- **CSS Enhancements**:
  - `.modal-overlay` - full-screen overlay с backdrop
  - `.device-details-modal` - responsive modal (max-width: 700px, max-height: 90vh)
  - `.modal-header/.modal-body/.modal-footer` - структура modal
  - `@keyframes fadeIn, slideIn, slideUp` - smooth animations
  - `.btn-outline-secondary` - новый button style для "View Details"
- **Security**:
  - Current device НЕ может быть selected для bulk revoke
  - Bulk revoke проверяет наличие current device в selection
  - Each revoke operation проходит через DELETE endpoint с owner verification
- **Performance**:
  - Modal рендерится on-demand (не в DOM by default)
  - Bulk operations асинхронные (Promise.all можно добавить в future)
  - Event delegation для checkboxes
- **Backward Compatibility**:
  - Existing "Rename" и "Remove" buttons продолжают работать
  - API endpoints НЕ изменились
  - Только frontend updates (HTML/CSS/JS)

## [2.4.3] - 2025-10-28

### Added

- **WebSocket Streaming для Desktop** (DESKTOP-03): Real-time chat через WebSocket с API key authentication
  - **WebSocket Chat Endpoint**: `GET /ws/chat?token={api_key}`
    - API key authentication через query parameter
    - Bcrypt validation для безопасности (проверка всех ключей)
    - Device tracking: автоматическое обновление `last_seen_at` при WebSocket activity
    - Connection upgrade с HTTP to WebSocket protocol
  - **Chat Message Types**:
    - `chat_request` - chat запрос от desktop client (с model, messages, streaming params)
    - `chat_chunk` - streaming chunk от сервера (content, role, done flag)
    - `chat_done` - финальное сообщение (message_id, total_tokens, finish_reason)
    - `chat_error` - ошибка (error message, error code)
    - `ping`/`pong` - heartbeat для keep-alive
  - **Per-User Message Routing**: `Hub.SendToUser(userID, message)` для targeted messaging
  - **ChatHandler**: Обработчик chat requests через WebSocket
    - Ollama integration через StreamingClient
    - Async request processing (non-blocking)
    - Token counting и метрики
    - Error handling с graceful fallback
  - **WebSocket Handler Enhancements**:
    - `HandleChatWebSocket()` - новый endpoint с API key auth
    - `SetDatabase()` - DI для API key validation
    - `SetChatHandler()` - DI для chat request routing
    - Message routing по типу (chat_request, ping)
  - **Hub Updates**:
    - `userClients map[string][]*Client` - per-user client tracking
    - `SendToClient(clientID, message)` - individual client messaging
    - Automatic cleanup при disconnect

### Technical

- **Backend (Go)**:
  - `internal/websocket/chat_handler.go` - Chat handler для WebSocket streaming (267 lines)
  - `internal/websocket/handler.go` - API key auth, message routing, pong response
  - `internal/websocket/hub.go` - Per-user client tracking, SendToUser/SendToClient methods
  - `internal/api/router/router.go` - Route `/ws/chat`, ChatHandler initialization
  - Message types: `ChatRequestMessage`, `ChatChunkMessage`, `ChatDoneMessage`, `ChatErrorMessage`
- **Security**:
  - API key validation: bcrypt verification против всех активных ключей
  - Status check: только `active` ключи permitted
  - Expiration check: `auto_expire_at` validation
- **Performance**:
  - Ping/Pong heartbeat: 54s interval (existing from v1.10.2)
  - Async request processing: non-blocking goroutines
  - Efficient message routing: per-user client maps
- **Backward Compatibility**:
  - SSE endpoints (`/v1/chat/completions` с `stream=true`) продолжают работать
  - WebSocket `/ws` для metrics (без auth) не изменен
- **Documentation**: `BACKLOG/DESKTOP-03_websocket_streaming.md` с примерами интеграции

## [2.4.2] - 2025-10-28

### Added

- **Device Management API** (DESKTOP-02): Полное управление устройствами desktop client
  - **REST API Endpoints**:
    - `GET /api/auth/devices` - список устройств пользователя с фильтрацией (status, sort, order)
    - `GET /api/auth/devices/:id` - детальная информация об устройстве с usage статистикой
    - `DELETE /api/auth/devices/:id` - revoke device API key с защитой от self-revoke
    - `PATCH /api/auth/devices/:id` - обновление user-friendly имени устройства
  - **Device Management WebUI** (`/profile-devices.html`):
    - Список всех зарегистрированных устройств с карточками
    - Визуальные индикаторы: OS icons (🪟 Windows, 🍎 macOS, 🐧 Linux), статус (Active/Inactive/Expired)
    - Highlight текущего устройства с badge "This Device"
    - Фильтры: status (active/expired/all), сортировка (last_seen/created_at/name), order (asc/desc)
    - Статистика: Total Devices, Active Devices, Current Device
    - Действия: Rename device (prompt), Remove device (confirmation modal)
    - Last seen timestamp с цветовой индикацией (green - recent, red - inactive >30 days)
    - Адаптивный grid layout (CSS Grid с auto-fill/auto-fit)
    - Темная тема с CSS переменными (без Bootstrap)
  - **Security Features**:
    - Owner verification - пользователь может управлять только своими устройствами
    - Self-revoke prevention - нельзя удалить текущее устройство
    - JWT authentication для всех device management endpoints
  - **Data Models** (`DeviceInfo`, `ListDevicesResponse`, `UpdateDeviceNameRequest`, `DeviceFilters`)

### Technical

- **Backend (Go)**:
  - `internal/api/handlers/device_handler.go` - Device management handlers (380+ lines)
  - `internal/storage/sqlite/apikeys.go` - Device-specific queries (`ListDeviceAPIKeys`, `UpdateDeviceName`)
  - `internal/storage/database.go` - Interface updates для device management methods
  - `internal/models/apikey.go` - Helper methods (`ToDeviceInfo()`, `IsDeviceKey()`, `GenerateDeviceName()`)
  - Route registration в `/api/auth/devices` group с JWT middleware
- **Frontend (HTML/CSS/JS)**:
  - `web/profile-devices.html` - Device management page (297 lines, CSS Grid layout)
  - `web/js/profile-devices.js` - DeviceManager class (367 lines, follows Dashboard/ProfileManager pattern)
  - `web/js/components/navbar.js` - Добавлена ссылка "My Devices" в user dropdown
  - `web/chat.html` - Добавлены кнопки "Devices" и "Profile" в sidebar footer
  - Использует `modal.danger()` и `toast.*` из `notifications.js`
  - Emoji icons вместо Bootstrap Icons для совместимости с темной темой
- **Database**: Использует существующие device fields из migration v66 (DESKTOP-01)
- **Testing**: Unit tests для device management endpoints (pending full test suite fix)

## [2.4.1] - 2025-10-28

### Added

- **Auto-Generated API Keys System** (DESKTOP-01): Автоматическая регистрация desktop устройств
  - **Device Registration API**:
    - `POST /api/auth/devices/register` - auto-creation API keys для desktop client
    - Device metadata: OS (windows/darwin/linux), hostname, app version, fingerprint (SHA256)
    - Automatic device naming: "Desktop App - {OS} - {Date}" с fallback на hostname
    - Auto-expiry для device keys (default 90 days, настраиваемый)
    - Duplicate detection по device fingerprint (SHA256 hash device-specific info)
  - **Authentication Flow for Desktop**:
    1. Desktop app логинится username/password → получает JWT token
    2. JWT token используется для `POST /api/auth/devices/register` → получает API key
    3. API key сохраняется локально и используется для всех последующих запросов
    4. JWT token discarded (security best practice)
  - **Device Tracking**:
    - `last_seen_at` timestamp (prepared for automatic updates in middleware)
    - Device fingerprint для unique identification и duplicate prevention
    - Separate device keys отображаются как отдельные устройства в management UI
- **Data Model Extensions**:
  - `APIKey` struct: added 7 device fields (device_name, device_os, device_hostname, device_version, device_fingerprint, last_seen_at, auto_expire_at)
  - `DeviceRegistrationRequest` / `DeviceRegistrationResponse` structs
  - Helper methods: `IsDeviceKey()`, `GenerateDeviceName()`

### Technical

- **Backend (Go)**:
  - `internal/api/handlers/device_handler.go` - DeviceHandler with RegisterDevice method
  - `internal/storage/sqlite/apikeys.go` - Device-related queries (`FindAPIKeyByDeviceFingerprint`, `UpdateAPIKeyLastSeen`)
  - `internal/storage/database.go` - Interface updates для device methods
  - `internal/models/apikey.go` - Device metadata fields и validation
  - Route: `POST /api/auth/devices/register` в `authProtected` group (JWT auth required)
- **Database Migration v66** (`add_device_fields_to_api_keys`):
  - Added columns: device_name, device_os, device_hostname, device_version, device_fingerprint, last_seen_at, auto_expire_at
  - Indices: idx_api_keys_device_fingerprint, idx_api_keys_last_seen, idx_api_keys_device_os
- **PostgreSQL**: Stub implementations в `internal/storage/postgresql/stubs.go`
- **Testing**: Unit tests для device registration endpoint
- **Documentation**: API endpoints documented в `device_handler.go` Swagger comments

## [2.3.1] - 2025-10-28

### Changed

- **Admin Panel UI Refactoring**: Реорганизация табов для улучшения навигации и устранения горизонтального скролла
  - **Users & RBAC**: Объединены "Users", "Invitations" и "RBAC" в один таб с подтабами
    - Users → управление пользователями с расширенной информацией
    - Invitations → система приглашений (AUTH-03)
    - RBAC → управление ролями и разрешениями
  - **Models**: Объединены "Available Models" и "Model Registry" в один таб с подтабами
    - Available Models → список доступных моделей Ollama
    - Model Registry → управление провайдерами и зарегистрированными моделями (REGISTRY-03)
  - **System & Logs**: Объединены "Performance", "Audit" и "Logs" в один таб с подтабами
    - Performance → мониторинг CPU, RAM, GPU, MoniGo метрики
    - Audit → события безопасности и compliance мониторинг
    - Logs → просмотр системных логов с фильтрацией
  - **Итого**: Сокращение с 11 до 8 основных табов, улучшенная группировка по функциональности

### Fixed

- **Logs Tab Loading**: Исправлена ошибка `Uncaught SyntaxError: Identifier 'logsViewer' has already been declared`
  - Удалено дублирующее объявление переменной `logsViewer` в `admin.html`
  - Обновлен селектор в `logs.js` с `[data-tab="logs"]` на `[data-subtab="system-logs"]`
  - Теперь логи корректно загружаются при переключении на подтаб Logs

### Technical

- **Frontend (HTML/CSS/JS)**:
  - Добавлена система подтабов (`.sub-tabs`, `.sub-tab-btn`, `.sub-tab-pane`) в `admin.html`
  - JavaScript логика для переключения подтабов с сохранением контекста родительского таба
  - CSS стили для визуального разделения основных табов и подтабов
  - Функция `loadAuditPreview()` для автозагрузки audit событий при открытии подтаба
  - Интеграция существующего `LogsViewer` класса для работы с новой структурой табов
  - Все модальные окна и существующий функционал сохранен без изменений

## [2.3.0] - 2025-10-28

### Added

- **Model Registry & Multi-Provider Foundation** (REGISTRY-01, REGISTRY-03): Универсальная система управления моделями от разных провайдеров
  - **Model Registry Core**:
    - Централизованный реестр всех доступных моделей с метаданными
    - Поддержка множества провайдеров: Ollama, vLLM, OpenAI, Anthropic, Custom
    - Отслеживание capabilities моделей: chat, embeddings, vision, function-calling
    - Health monitoring для провайдеров и моделей с историей статусов
    - Auto-discovery механизм для автоматического обнаружения новых моделей
    - Performance метрики: latency, throughput, total requests
  - **Model Registry WebUI** (`admin-registry.html`):
    - Dashboard с provider status cards (active/inactive/error states)
    - Табы "Model Providers" и "Registered Models" для раздельного управления
    - Real-time health indicators с автообновлением каждые 30 секунд
    - Модальные формы для создания/редактирования провайдеров и моделей
    - Фильтрация по provider, status, health, keyword
    - Кнопка "Discover Models" для запуска автообнаружения моделей
    - Statistics cards: total models, active models, providers count, healthy models
    - Integration с основной admin панелью через вкладку "Model Registry"
  - **Database Schema**:
    - Таблица `model_registry`: хранение метаданных моделей
    - Таблица `model_providers`: конфигурация провайдеров с API keys
    - Поля для capabilities, health status, performance metrics
  - **Configuration** (`config.go`, `dev.yaml`):
    - Секция `model_registry` для включения/отключения функционала
    - Настройки auto-discovery и health check интервалов
    - Provider configurations для Ollama (active), vLLM (inactive)

### Technical

- **Backend (Go)**:
  - Новые модели в `internal/models/registry.go`:
    - `ModelRegistry`: описание зарегистрированной модели
    - `ModelProvider`: конфигурация провайдера моделей
    - `ModelCapabilities`: битовая маска для chat/embeddings/vision/functions
    - `ProviderType`: enum для типов провайдеров
    - `HealthStatus`: enum для статусов здоровья
  - Реализация `RegistryHandler` в `internal/api/handlers/registry_handler.go`:
    - CRUD операции для провайдеров и моделей
    - Health check endpoints для мониторинга
    - Discovery endpoint для автоматического обнаружения моделей
    - Stats endpoint для dashboard метрик
  - Реализация `ProviderManager` в `internal/providers/manager.go`:
    - Управление lifecycle провайдеров (load, health check, discovery)
    - Background loops для периодических задач (discovery, health checks)
    - Интеграция с Ollama и vLLM провайдерами
  - Storage layer (`internal/storage/sqlite/model_registry.go`):
    - CRUD методы для `model_registry` и `model_providers` таблиц
    - Support для `NULL` значений в `api_key` и `error_message` через `sql.NullString`
  - Миграции (v16, v17):
    - v16: создание таблиц `model_registry` и `model_providers`
    - v17: seed данных - Ollama провайдер и базовые модели (llama3.2, qwen2.5, etc.)
  - Router integration (`internal/api/router/router.go`):
    - Функция `setupModelRegistry` для инициализации провайдеров
    - Регистрация Model Registry API endpoints в `/api/admin/registry/*`
    - Background tasks для auto-discovery и health monitoring

- **Frontend (HTML/JS/CSS)**:
  - `web/admin-registry.html`: полноценная страница управления Model Registry
  - `web/js/admin-registry.js`: client-side логика для CRUD операций, фильтрации, real-time updates
  - Модальные окна для создания/редактирования провайдеров и моделей
  - CSS стили для status badges, health indicators, provider cards
  - Интеграция с существующей темой и dashboard стилями

- **Configuration**:
  - `configs/dev.yaml`: добавлена секция `model_registry` с настройками
  - `internal/config/config.go`: структуры `ModelRegistryConfig`, `ProviderConfig`

### Notes

- vLLM провайдер настроен, но отключен (inactive) до развертывания vLLM сервера
- Система готова к добавлению новых провайдеров через UI или конфигурацию
- Auto-discovery работает в фоновом режиме с настраиваемым интервалом
- Health checks обновляют статусы провайдеров и моделей автоматически

## [2.2.2] - 2025-10-28

### Added

- **Enhanced User Management**: Админ-панель теперь отображает расширенную информацию о пользователях
  - **Auth Provider Badge**: Отображение способа регистрации пользователя (Local, OIDC, LDAP)
    - Local: фиолетовый градиент
    - OIDC: розовый градиент
    - LDAP: синий градиент
  - **RBAC Roles Display**: Показывает роли пользователя из системы RBAC
    - Отображение до 2 ролей с badge'ами
    - Счетчик "+N" для остальных ролей
    - Иконка 🏢 для tenant-specific ролей
    - "No roles" для пользователей без ролей
  - **Tenants Display**: Отображение всех тенантов в которых участвует пользователь
    - Отображение до 2 тенантов с badge'ами
    - Иконка 👑 для Owner, 👤 для Member
    - Золотой градиент для Owner, оранжевый для Member
    - Счетчик "+N" для остальных тенантов
  - **XSS Protection**: Все пользовательские данные экранируются через `escapeHtml()` для безопасности

### Technical

- **Backend (Go)**:
  - Новые модели: `UserWithDetails`, `RoleInfo`, `TenantInfo` в `internal/models/user.go`
  - Реализован метод `GetUsersWithDetails(ctx, filters)` для SQLite с JOIN'ами к RBAC и tenants таблицам
  - Добавлен метод `ListAllTenants(ctx)` для получения списка всех tenants
  - Stub реализации для PostgreSQL в `internal/storage/postgresql/stubs.go`
  - Delegation методы в `sqliteTx` и `postgresqlTx` для поддержки транзакций
  - Обновлен `AdminUserHandler.ListUsers()` для возврата enriched данных

- **Frontend (HTML/JS/CSS)**:
  - Обновлена таблица Users в `web/admin.html` с 3 новыми колонками: AUTH PROVIDER, RBAC ROLES, TENANTS
  - Новые helper функции в `web/js/admin.js`:
    - `renderAuthProviderBadge(authProvider)` - рендеринг badge для способа аутентификации
    - `renderRolesBadges(roles)` - рендеринг RBAC ролей с ограничением отображения
    - `renderTenantsBadges(tenants)` - рендеринг тенантов с owner/member индикацией
    - `escapeHtml(text)` - защита от XSS атак
  - Добавлены CSS стили для новых badges в `web/css/style.css`:
    - `.badge-auth-local`, `.badge-auth-oidc`, `.badge-auth-ldap` (градиентные фоны)
    - `.badge-role`, `.badge-tenant`, `.badge-tenant-owner` (роли и тенанты)
    - `.badge-count` (счетчик для скрытых элементов)
  - Обновлен метод `renderUsers()` для использования новых helper функций

### Fixed

- **JavaScript Syntax Error**: Исправлена критическая ошибка в `web/js/admin.js` на строке 342
  - Незакрытая arrow function в методе `renderUsers()`
  - Добавлено корректное закрытие: `}).join('');` вместо `).join('');`

### Changed

- API endpoint `/api/admin/users` теперь возвращает `UserWithDetails` вместо базовой модели `User`
- Таблица Users расширена с 6 до 9 колонок для отображения новой информации

## [2.2.1] - 2025-10-28

### Fixed

- **RBAC Management WebUI**: Исправлена критическая ошибка загрузки пользователей во вкладке "User Roles"
  - Несоответствие ID элемента между HTML (`id="userSelector"`) и JavaScript (`$('#userSelect')`)
  - Теперь корректно отображается список всех пользователей системы (24 users)
  - Корректно загружается список всех tenants (26 tenants) при назначении ролей
  
- **RBAC Permissions Reference**: Исправлена загрузка permissions при переключении вкладок
  - Добавлен механизм callbacks для табов с явным вызовом `handlePermissionsTab()` и `handleUserRolesTab()`
  - Экспортированы функции в `window` scope для доступа из HTML
  - Теперь 35 системных permissions корректно отображаются с группировкой по ресурсам

### Added

- **Admin API Endpoint**: Новый endpoint `/api/admin/tenants` для получения списка всех tenants
  - Реализация `ListAllTenants()` в SQLite storage
  - Stub реализация для PostgreSQL
  - Интеграция в router с admin middleware и JWT auth
  - Delegation методы в transaction wrappers

### Technical

- Обновлена структура `TenantHandler` с методом `ListAllTenants()`
- Добавлена функция `loadUsersAndTenants()` в `admin-rbac.js` с параллельной загрузкой данных
- Улучшены empty state сообщения для permissions и users с actionable инструкциями
- Добавлены callbacks механизм для Bootstrap tabs в `admin-rbac.html`
- Исправлен ID HTML элемента `userSelector` → `userSelect` для соответствия с JavaScript селектором

## [2.2.0] - 2025-10-28

### Added

- **Invitation-Only Registration System (AUTH-03)**: Полная система управления приглашениями для контролируемой регистрации пользователей
  - Database schema с таблицей `invitations` (migration v57 для SQLite, v4 для PostgreSQL)
  - REST API endpoints для создания, просмотра, отзыва и валидации приглашений
  - Admin WebUI: `/admin-invitations.html` - страница управления приглашениями с фильтрацией и статистикой
  - Поддержка ограничений: по email, сроку действия, количеству использований
  - Детальная информация о пользователях: кто создал, кто использовал, кто отозвал приглашение
  - Регистрация по приглашению: обновлен `/register.html` с поддержкой invitation tokens
  
- **Invitation Management Features**:
  - Создание приглашений с настраиваемыми параметрами (email restriction, expiry, max uses)
  - Автоматическая генерация уникальных invitation links
  - Статистика приглашений (Active, Pending, Used, Expired, Revoked)
  - Фильтрация по статусу, email, создателю
  - View modal с полной информацией включая user details через LEFT JOIN
  - One-click копирование invitation links и tokens
  
- **Configuration Options**: Новые настройки в `configs/dev.yaml`
  - `auth.registration.mode`: "open" | "invitation_only" | "disabled"
  - `auth.invitations.enabled`: включение системы приглашений
  - `auth.invitations.default_expiry_days`: срок действия по умолчанию
  - `auth.invitations.max_uses_default`: количество использований
  - Rate limits для создания и валидации приглашений

### Changed

- **WebUI Improvements**:
  - Улучшен контраст текста в статистических карточках (белый текст на цветных градиентах)
  - Markdown форматирование для пользовательских сообщений в ChatUI
  - WYSIWYG-подобная панель форматирования текста при выделении (bold, italic, code, lists)
  - Сохранение переносов строк в сообщениях чата (`white-space: pre-wrap`)

- **Registration Flow**: Обновлен процесс регистрации с проверкой invitation tokens
  - Валидация токена перед показом формы регистрации
  - Автоматическое использование приглашения после успешной регистрации
  - Email restriction check для приглашений привязанных к конкретному email

### Fixed

- **Database Schema**: Исправлена ошибка с колонкой `display_name` → `full_name` в запросах с JOIN к таблице users
- **Invitation Links**: Исправлена генерация ссылок - добавлено `.html` расширение (`/register.html?invite=...`)

### Technical

- **Backend (Go)**:
  - Новые модели: `Invitation`, `InvitationWithUsers`, `UserInfo`, `InvitationStatus`
  - Storage layer: полная реализация CRUD операций для SQLite и PostgreSQL
  - Handler: `InvitationHandler` с 6 endpoint'ами (create, list, stats, details, revoke, validate)
  - AuthService: интеграция invitation token validation в процесс регистрации
  - Transaction delegation: добавлены методы в `sqliteTx` и `postgresqlTx`

- **Database Migrations**:
  - SQLite migration v57: создание таблицы `invitations` с индексами
  - PostgreSQL migration v4: аналогичная схема для PostgreSQL
  - LEFT JOIN queries для получения информации о пользователях

- **API Endpoints**:
  - `POST /api/admin/invitations` - создание приглашения
  - `GET /api/admin/invitations` - список приглашений с фильтрацией
  - `GET /api/admin/invitations/stats` - статистика
  - `GET /api/admin/invitations/:id` - детальная информация с user info
  - `DELETE /api/admin/invitations/:id` - отзыв приглашения
  - `GET /api/invitations/:token/validate` - публичная валидация токена

- **Frontend**:
  - `web/admin-invitations.html` (623 строки) - полнофункциональная админ-панель
  - `web/js/api.js` - 6 новых методов для работы с invitations API
  - `web/register.html` - поддержка `?invite=` query parameter
  - Formatting toolbar для ChatUI с keyboard shortcuts (Ctrl+B, Ctrl+I, Ctrl+K, Ctrl+L)

### Security

- **Access Control**: Все admin endpoints защищены JWT authentication + RequireAdmin middleware
- **Rate Limiting**: Настраиваемые лимиты для создания приглашений и валидации токенов
- **Token Security**: UUID v4 tokens для приглашений, проверка валидности перед использованием
- **Email Verification**: Опциональная привязка приглашения к конкретному email

## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

### Changed

- **Repository name**: `ollama-openai-proxy` → `aigateway`
- **Module path**: `ollama-openai-proxy` → `aigateway`
- **Container name**: `ollama-openai-proxy` → `aigateway`
- **Database file**: `proxy.db` → `aigateway.db` (optional rename)
- **Log files**: `proxy.log` → `aigateway.log`

- **Environment Variables (⚠️ Breaking Change)**: All `PROXY_*` → `AIGATEWAY_*`
  - Example: `PROXY_SERVER_PORT` → `AIGATEWAY_SERVER_PORT`
  - `PROXY_OLLAMA_URL` → `AIGATEWAY_OLLAMA_URL`
  - `PROXY_DATABASE_TYPE` → `AIGATEWAY_DATABASE_TYPE`
  - See [Migration Guide](docs/MIGRATION_GUIDE_v2.1.0.md) for complete mapping

- **Documentation Updates**: 40+ files rebranded
  - README.md - complete rewrite для AIGateway Platform
  - Architecture.MD - updated diagrams with RAG System, Model Registry
  - All docs/*.md files (20+ files)
  - All BACKLOG/*.md files

- **Code Changes**: Zero functional changes - pure rebranding
  - go.mod module path updated
  - All import statements across entire codebase
  - Docker Compose configuration
  - Dockerfile VERSION=2.1.0

- **WebUI Branding**: Complete frontend rebranding
  - All HTML page titles: "AIGateway Platform"
  - Navigation labels и headers
  - About System page
  - Footer copyright

### Why Rebranding?

**Reasons for transition to "AIGateway":**

1. **Expanded Scope**: No longer just an Ollama proxy
  - Multi-provider support: vLLM (v2.2.0), future: OpenAI, Anthropic
  - Model registry для unified API access

2. **RAG System**: Built-in RAG capabilities
  - Vector search, embeddings, document processing
  - Enterprise-ready data integration

3. **Enterprise Positioning**: "Gateway" better represents platform role
  - Central AI infrastructure component
  - Unified API для multiple backends

4. **Scalability**: Name allows future expansion
  - Cloud provider integration
  - Custom model support

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed steps.

**Quick Migration Checklist:**
  - Update environment variables: `PROXY_*` → `AIGATEWAY_*`
  - Update Docker image names
  - Rename database file (optional): `proxy.db` → `aigateway.db`
  - Update scripts/configs referencing old names
  - Pull new Docker images: `aigateway:2.1.0`

### Technical

- **Module path**: `aigateway` (was `ollama-openai-proxy`)
- **Import paths**: Updated throughout codebase (~150+ Go files)
- **Docker Compose**: Service name `aigateway`, volumes `aigateway_data`/`aigateway_logs`
- **Functional changes**: Zero - pure rebranding release
- **Compilation**: Verified ✅
- **Database schema**: Unchanged (backward compatible)

---

## [2.0.0] - 2025-10-27

### 🚀 Major Features

- **RAG System (Retrieval-Augmented Generation)**: Полная интеграция системы RAG для работы с внешними источниками данных
  - Поддержка множественных типов источников данных:
    - REST API с аутентификацией (Basic, Bearer Token, API Key)
    - PostgreSQL базы данных с incremental sync
    - File Upload для документов (PDF, TXT, MD, DOCX)
    - Web Scraping для веб-страниц
  - Semantic chunking с intelligent text splitting
    - Поддержка параграфов и предложений
    - Configurable chunk size и overlap
    - Token estimation для оптимального размера chunks
  - Vector embeddings через Ollama:
    - Модели: `mxbai-embed-large`, `nomic-embed-text`
    - Batch processing для производительности
    - Настраиваемые размерности векторов
  - PgVector для хранения и similarity search:
    - Cosine similarity, L2 distance, dot product
    - HNSW indexing для быстрого поиска
    - Масштабируемое хранение векторов
  - RAG Orchestrator с reranking:
    - Keyword overlap scoring
    - Metadata boosting
    - Configurable Top-K chunks retrieval
    - Min similarity score filtering
  - Context assembly для LLM:
    - Форматирование retrieved context
    - Source attribution
    - Token-aware context window management
  - Query logging для analytics:
    - Performance metrics (search time)
    - Quality scoring
    - Usage statistics по источникам

- **RAG Management WebUI**: Полнофункциональный интерфейс управления RAG
  - **User Dashboard** (`/rag-sources.html`):
    - CRUD операции для личных RAG источников
    - Real-time статус синхронизации
    - Connection testing перед созданием источника
    - Управление credentials с шифрованием
    - Statistics: chunks count, tokens, last sync
  - **Admin Panel** (`/admin-rag.html`):
    - Просмотр всех RAG источников всех пользователей
    - Фильтрация по user, type, status
    - Bulk operations (sync, delete)
    - Detailed metrics и analytics
    - Pagination для больших списков
  - **Chat Integration** (`/chat.html`):
    - RAG toggle в UI чата
    - Multi-select для выбора data sources
    - Sliders для Top K chunks и Min Score
    - Reranking checkbox
    - Source attribution в ответах
  - **RAG Disable Banner**: Информационная плашка когда RAG отключен администратором

- **Chat Export/Import UI**: Полнофункциональный интерфейс экспорта и импорта conversations
  - **Export Dropdown в Chat Header**:
    - Export as JSON (structured data с metadata)
    - Export as Markdown (readable format с форматированием)
    - Export as Text (plain text без форматирования)
    - Автоматическое скачивание файла с sanitized filename
  - **Import Modal**:
    - Upload JSON файла экспортированной conversation
    - Опция "Preserve original timestamps" (сохранить оригинальные даты)
    - Опция "Preserve original IDs" (сохранить оригинальные идентификаторы)
    - Validation JSON структуры перед импортом
    - Success notification с количеством imported messages
    - Автоматический reload для отображения импортированной conversation

- **Browser Testing Integration**: MCP browser extension для E2E тестирования
  - Chrome browser automation через Playwright
  - Accessibility snapshots для UI testing
  - Screenshot capabilities
  - Form interactions и validations
  - Network requests monitoring

### 🎨 UI/UX Improvements

- **Modal Windows Centering**: Исправлено позиционирование модальных окон
  - Модалки теперь появляются строго по центру экрана (horizontal + vertical)
  - Flexbox-based centering для надежности
  - Smooth animations с правильным transform
  - Backdrop blur эффект
  - Responsive design для mobile

- **Consistent Dashboard Styling**: Единообразный dark theme на всех страницах
  - Серый фон для content areas (`var(--bg-secondary)`)
  - Темные карточки с прозрачностью (`var(--bg-tertiary)`)
  - Улучшенная читаемость текста (explicit color definitions)
  - Убраны белые полосы и артефакты
  - Consistent borders и border-radius
  - Применено на всех страницах:
    - Dashboard, API Keys, RAG Sources, Admin RAG
    - About System, Usage, Profile, Files
    - MCP Catalog, Tenants, RBAC, Audit

- **Login Page Redesign**: Двухколоночный layout с gradient background
  - **Левая колонка**: Login form
  - **Правая колонка**: System info panel
    - System Status (Online badge с пульсацией)
    - Version info
    - Available Models list с размерами
    - RAG System status
  - Gradient background (blue → purple → pink)
  - Dynamic model loading через `/api/v1/models`

- **Error Messages Styling**: Улучшенный контраст и visibility
  - Error messages: rgba(239, 68, 68) с border и shadow
  - Success messages: rgba(16, 185, 129) с border и shadow
  - Improved padding и font-weight
  - Better readability на темном фоне

- **RBAC Page Refactoring** (`/admin-rbac.html`):
  - Удален Bootstrap, full custom CSS
  - Custom tab navigation
  - Dark theme compatibility
  - Card styles с правильными цветами
  - Fixed non-clickable buttons issue

- **Audit Log Page Refactoring** (`/admin-audit.html`):
  - Consistent styling с другими admin pages
  - Improved table layout
  - Better filters section
  - Pagination controls

### 🔐 Security & Authentication

- **JWT Token Rotation**: Refresh token rotation для enhanced security
  - Новый refresh token выдается при каждом обновлении access token
  - Старый refresh token добавляется в blacklist
  - Frontend обновляет оба токена при refresh
  - Предотвращение token replay attacks

- **Authentication Flow Fixes**: Исправлен logout loop
  - `clearSessionAndRedirect()` вместо `logout()` при refresh failure
  - Корректная очистка localStorage
  - Правильный redirect на `/login.html`
  - Предотвращение API calls с invalid tokens

- **Public API Endpoints**: `/api/models` и `/api/v1/models` теперь публичные
  - Доступны без аутентификации для Login page
  - Display available models до входа в систему

- **RAG Credentials Encryption**: AES-256 шифрование credentials
  - Configurable encryption key в RAG config
  - Безопасное хранение API keys, passwords, tokens
  - Encrypt при создании, decrypt при использовании

### 📊 Logging & Monitoring

- **Separate Error Logging**: Dedicated error log file
  - Новый config `logging.error_log_enabled`
  - Отдельный файл для errors и warnings (`logs/proxy-errors.log`)
  - Logrus hook с lumberjack rotation
  - Configurable max_size, max_backups, max_age, compress
  - Defaults наследуются из основной logging config
  - Log rotation при старте сервера

- **Audit Events Metadata Fix**: JSON serialization для metadata
  - `map[string]interface{}` теперь корректно сохраняется
  - JSON Marshal/Unmarshal в SQLite implementation
  - Исправлена ошибка "unsupported type map[string]interface{}"

### 🛠️ Technical Improvements

- **API Client Enhancements** (`web/js/api.js`):
  - Generic HTTP methods: `get()`, `post()`, `put()`, `delete()`
  - RAG-specific methods: `getRAGSources()`, `createRAGSource()`, etc.
  - RBAC methods: `getRBACRoles()`, `createRBACRole()`, etc.
  - Audit methods: `getAuditLogs()`, `exportAuditLogs()`, etc.
  - System methods: `getSystemInfo()`, `isRAGEnabled()`
  - Improved token refresh logic
  - Better error handling

- **Context Parsing Fix**: User/Tenant ID prefix stripping
  - JWT middleware добавляет "user_" и "tenant_" prefixes
  - RAG handlers теперь корректно парсят с strip префиксов
  - Исправлены UUID parsing errors

- **CSS Conflicts Resolution**: 
  - Удалено дублирующее `.modal` правило из `theme.css`
  - Высокоприоритетные селекторы в `dashboard.css`
  - `max-width: none !important` для override
  - Правильный flexbox centering

- **Navigation Component** (`web/js/components/navbar.js`):
  - Добавлена поддержка `rag-sources.html` и `admin-rag.html`
  - Правильное определение текущей страницы
  - Active state для навигационных ссылок

### 📚 Documentation

- **RAG Deployment Guide** (`docs/RAG_DEPLOYMENT_GUIDE.md`):
  - Prerequisites (PostgreSQL with pgvector, Ollama, embeddings models)
  - Configuration examples
  - Docker Compose setup
  - Kubernetes deployment
  - Troubleshooting guide

- **RAG Config Guide** (`docs/RAG_CONFIG_GUIDE.md`):
  - Detailed configuration options
  - Performance tuning
  - Security best practices
  - Enable/Disable instructions

- **RAG Testing Guide** (`docs/RAG_TESTING.md`):
  - Go unit tests coverage
  - Integration testing
  - E2E Playwright tests

- **Error Logging Guide** (`docs/ERROR_LOGGING.md`):
  - Configuration instructions
  - Usage examples
  - Best practices

### 🔧 Configuration

- **RAG Configuration** (`rag` section в config):
  - `enabled`: Enable/disable RAG subsystem
  - `vector_store`: PgVector connection settings
  - `file_storage`: Local/S3/Azure storage for documents
  - `embeddings`: Ollama embeddings configuration
  - `processing`: Worker pool settings, chunk sizes
  - `retrieval`: Top-K, similarity threshold, reranking
  - `queue`: PostgreSQL job queue settings
  - `security`: Encryption key для credentials

- **Logging Configuration Enhancements**:
  - `error_log_enabled`: Enable separate error log
  - `error_log_file_path`: Path to error log file
  - `error_log_max_size`: Max size before rotation (MB)
  - `error_log_max_backups`: Number of old error logs to keep
  - `error_log_max_age`: Days to keep old error logs
  - `error_log_compress`: Compress old error logs

### 🗃️ Database

- **RAG Schema Migrations**:
  - `rag_data_sources`: Data source definitions
  - `rag_documents`: Document metadata
  - `rag_chunks`: Text chunks с embeddings
  - `rag_jobs`: Processing job queue
  - `rag_query_logs`: Query analytics
  - PostgreSQL + SQLite support

### 🧪 Testing

- **Go Unit Tests**: Comprehensive test coverage
  - `internal/rag/chunker/semantic_test.go`: 15+ tests
  - `internal/rag/orchestrator/orchestrator_test.go`: 12+ tests
  - `internal/services/rag/datasource_service_test.go`: 13+ tests
  - `internal/api/handlers/chat_rag_test.go`: 8+ tests
  - `internal/rag/embeddings/ollama_test.go`: 10+ tests
  - `internal/rag/processor/worker_test.go`: 13+ tests

- **Playwright E2E Tests**: Browser-based UI testing
  - `tests/playwright/e2e/auth.spec.ts`: Authentication flows
  - `tests/playwright/e2e/chat.spec.ts`: Chat functionality
  - `tests/playwright/e2e/rag.spec.ts`: RAG UI interactions
  - `tests/playwright/e2e/dashboard.spec.ts`: Dashboard navigation
  - `tests/playwright/e2e/apikeys.spec.ts`: API key management
  - Multi-browser support (Chrome, Firefox, Safari)
  - Docker integration для CI/CD

### 🐛 Bug Fixes

- Fixed modal windows appearing off-center (left side instead of center)
- Fixed logout loop when refresh token is blacklisted
- Fixed `user_id` UUID parsing with "user_" prefix
- Fixed audit events metadata serialization error
- Fixed white-on-white text readability on About System page
- Fixed RBAC page non-clickable buttons
- Fixed MCP Catalog white spaces and styling issues
- Fixed modal animations with flexbox centering
- Fixed API error responses when RAG is disabled

### ⚡ Performance

- **Worker Pool для Document Processing**:
  - Parallel chunking с configurable workers
  - Metrics tracking (processed, failed, total time)
  - Graceful shutdown

- **Batch Embeddings**:
  - Batch processing для Ollama API
  - Reduced API calls overhead

- **Connection Pooling**:
  - HTTP client connection pooling для Ollama
  - Database connection pooling для PostgreSQL

### 🔄 Breaking Changes

- **Version Jump**: 1.12.3 → 2.0.0 (major release)
- **New Dependencies Required**:
  - PostgreSQL with pgvector extension для RAG
  - Ollama с embedding models для RAG
- **Configuration Changes**: Новый раздел `rag` в config files
- **Database Schema**: Новые таблицы для RAG system

### 📦 Dependencies

- Added `github.com/pgvector/pgvector-go` для vector operations
- Added Playwright для E2E testing
- Added lumberjack для log rotation
- Enhanced Ollama client для embeddings support

### 🎯 Next Steps

- WebUI для RAG analytics и query logs
- RAG performance metrics dashboard
- Advanced reranking algorithms
- Support для дополнительных embedding models
- RAG templates для common use cases

---

## [1.12.3] - 2025-10-26

### Added
- **Conversation Export/Import**: Полная система экспорта и импорта conversations
  - Export форматы: JSON, Markdown, Text
  - Single conversation export через GET `/api/conversations/{id}/export?format=json|markdown|text`
  - Bulk export через POST `/api/conversations/bulk-export` (multiple conversations at once)
  - Import из JSON через POST `/api/conversations/import`
  - Import опции:
    - Merge into existing conversation (добавить messages в существующую)
    - Preserve timestamps (сохранить оригинальные даты)
    - Preserve IDs (сохранить оригинальные IDs для recovery)
  - Metadata export (total messages, tokens used, model, dates)

### Changed
- **API**: Новые endpoints для работы с экспортом/импортом conversations
- **Models**: Новые data models для export/import operations

### Technical
- Новый сервис `internal/services/export/conversation_exporter.go`:
  - `ConversationExporter` с support для JSON/Markdown/Text форматов
  - Rich Markdown formatting с emojis и metadata
  - Bulk export с error handling для каждой conversation
- Новый сервис `internal/services/export/conversation_importer.go`:
  - `ConversationImporter` для восстановления conversations
  - Merge support (добавление messages в existing conversation)
  - Flexible options (preserve timestamps/IDs)
- Новый handler `internal/api/handlers/conversation_export.go`:
  - Export endpoints с content-type negotiation
  - Bulk export endpoint
  - Import endpoint с validation
- Data Models в `internal/models/conversation_export.go`:
  - `ConversationExport` - структура экспорта
  - `BulkExportRequest/Result` - bulk operations
  - `ImportConversationRequest` - импорт с опциями
  - `ImportResult` - результат импорта

### Security
- **Access Control**: Verify conversation ownership при export/import
- **Validation**: Strict validation для import data format
- **User Isolation**: Импорт только в свой tenant/user scope

### Performance
- **Bulk Export**: Efficient batch processing для multiple conversations
- **Error Resilience**: Продолжение export при ошибках отдельных conversations
- **Memory Efficient**: Streaming для больших conversations (future improvement)

### Use Cases
- **Backup**: Full backup conversations в JSON для restore
- **Sharing**: Export в Markdown для sharing с коллегами
- **Migration**: Import conversations с другого instance
- **Analysis**: Export в Text для text analysis
- **Recovery**: Restore deleted conversations из backup

## [1.12.2] - 2025-10-26

### Added
- **Advanced Rate Limiting**: Multi-scope rate limiting с sliding window algorithm
  - Scope support: Global, Tenant, User, API Key, Model-specific
  - Priority-based checking (API Key → User → Tenant → Model → Global)
  - Sliding window algorithm для точного подсчета requests (предотвращает burst attacks)
  - RFC 6585 compliance headers:
    - `X-RateLimit-Limit` - общий лимит
    - `X-RateLimit-Remaining` - оставшееся количество
    - `X-RateLimit-Reset` - Unix timestamp сброса
    - `X-RateLimit-Window` - временное окно (second/minute/hour/day)
    - `X-RateLimit-Scope` - какой scope сработал
    - `Retry-After` - через сколько секунд можно retry
  - 429 Too Many Requests при превышении лимита

### Changed
- **Rate Limiting**: Базовая система расширена multi-scope support
- **Database Schema**: Новые таблицы `rate_limits` и `rate_limit_usage` для гибкой настройки

### Technical
- Новый сервис `internal/services/ratelimit/sliding_window.go`:
  - `SlidingWindowLimiter` с in-memory cache для performance
  - Thread-safe operations с sync.RWMutex
  - Automatic cache cleanup (configurable)
  - Cache statistics для мониторинга
- Новый сервис `internal/services/ratelimit/advanced_service.go`:
  - `AdvancedRateLimiter` с multi-scope checking
  - Priority-based rate limit enforcement
  - Configurable default limits
  - Per-second, per-minute, per-hour, per-day windows
- Новый middleware `internal/api/middleware/advanced_rate_limit.go`:
  - RFC 6585 headers support
  - Context-aware (user, tenant, API key, model extraction)
  - Detailed error messages with retry information
- Data Models в `internal/models/rate_limit.go`:
  - `RateLimitConfig` - конфигурация rate limits
  - `RateLimitResult` - результат проверки
  - `RateLimitUsage` - tracking для persistence
- Database Migration v48:
  - `rate_limits` table с support для всех scopes
  - `rate_limit_usage` table для sliding window tracking
  - Indexes для efficient queries (scope, target_id, model_name)

### Performance
- **Sliding Window Algorithm**: Более точный чем fixed window, fair distribution
- **In-Memory Cache**: Fast lookups без DB queries на каждый request
- **Async Cleanup**: Periodic cache cleanup не блокирует requests
- **Low Latency**: < 1ms overhead на rate limit check (in-memory)

### Future Enhancements
- Admin API для управления rate limits (CRUD)
- WebUI для настройки limits через интерфейс
- Burst allowance support
- Redis backend для distributed rate limiting

## [1.12.1] - 2025-10-26

### Added
- **Model Preloading & Warming**: Механизм предзагрузки моделей для устранения cold start задержки
  - Preload моделей при старте сервера (настраиваемый список в конфигурации)
  - Health check loop для поддержания моделей в горячем состоянии
  - Автоматическая выгрузка неиспользуемых моделей через настраиваемый timeout
  - Track model usage для оптимизации preloading
  - Admin API endpoints:
    - `GET /api/admin/models/loaded` - список загруженных моделей с статусом
    - `POST /api/admin/models/:name/preload` - ручная загрузка модели

### Changed
- **Chat Handler**: Автоматический tracking использования моделей при каждом запросе
- **Configuration**: Добавлена секция `models.preload` с полной настройкой preloading

### Technical
- Новый сервис `internal/services/model/preloader.go`:
  - `ModelPreloader` с async startup и health check loops
  - Thread-safe tracking загруженных моделей
  - Graceful shutdown при остановке сервера
- Новый handler `internal/api/handlers/model_preload.go` для Admin API
- Integration в `Router` через `NewOptions.ModelPreloader`
- Integration в `ChatHandler` через `ModelPreloader` interface
- Конфигурация:
  - `models.preload.enabled` - включить/выключить preloading
  - `models.preload.on_startup` - загружать при старте
  - `models.preload.keep_warm` - поддерживать в горячем состоянии
  - `models.preload.health_check_interval` - интервал проверки (default: 5m)
  - `models.preload.warm_up_prompt` - тестовый промпт (default: "Hello")
  - `models.preload.max_loaded_models` - лимит одновременно загруженных (0 = unlimited)
  - `models.preload.unload_after` - timeout выгрузки (0 = never)

### Performance
- **First Request Latency**: Сокращение времени первого ответа с 5-30s до <1s для preloaded моделей
- **Memory Management**: LRU eviction через Ollama при достижении лимита памяти
- **Non-blocking**: Async preload не блокирует startup сервера

### Documentation
- Updated configs/dev.yaml с примером конфигурации preloading
- API documentation для Admin endpoints в Roadmap

## [1.11.9] - 2025-10-26

### Added
- **Enhanced Audit Logging**: Comprehensive audit trail for critical operations
  - User operations: creation, deletion, enable/disable (LogUserCreated, LogUserDeleted, LogUserUpdated)
  - API key operations: creation and deletion tracking (LogAPIKeyCreated, LogAPIKeyDeleted)
  - Tenant operations: creation, updates, deletion (LogTenantCreated, LogTenantUpdated, LogTenantDeleted)
  - Backup operations: creation and restoration tracking (LogBackupCreated, LogBackupRestored)
  - Performance monitoring: reduced update frequency from 5s to 10s for GPU and system metrics
  - WebUI performance: monitors now stop when not actively viewing System tab

### Technical
- Added AuditLogger integration to handlers:
  - `AdminUserHandler`: tracks user lifecycle events (create, delete, disable, enable)
  - `UserHandler`: tracks personal API key management
  - `TenantHandler`: tracks organization tenant operations
  - `BackupHandler`: tracks critical backup/restore operations
- New audit methods in `internal/services/audit/logger.go`:
  - `LogUserUpdated()` - tracks user status changes and updates
  - `LogTenantUpdated()` - tracks tenant information changes
- Updated handler constructors to accept `*audit.AuditLogger` parameter
- Router injection of `auditLogger` into all relevant handlers
- WebUI optimization: `admin.js` now stops performance/GPU monitors when switching tabs

### Security
- **Audit trail for CRITICAL operations**:
  - User deletion (data loss risk)
  - Backup restoration (overwrites current data)
  - Tenant deletion (organization data loss)
  - API key operations (security credentials)

## [1.11.7] - 2025-10-25

### Added

- **QUOTA-01: Usage Quotas System** 📊
  - **Flexible Quota System** для per-user и per-tenant limits
  - **Token Quotas**:
    - Daily token limits (`tokens_per_day`)
    - Monthly token limits (`tokens_per_month`)
    - Automatic usage tracking с prompt/completion tokens
  - **Request Quotas**:
    - Daily request limits (`requests_per_day`)
    - Monthly request limits (`requests_per_month`)
    - Concurrent request limiting (`max_concurrent`)
  - **Storage Quotas** (future-ready):
    - Max file upload size (`max_file_size`)
    - Max total storage per user/tenant (`max_storage_bytes`)
    - Max conversations count (`max_conversations`)
  - **Model Restrictions**:
    - Per-quota model allow-list (`allowed_models`)
    - Block specific models for certain users/tenants
  - **Auto-Reset Logic**:
    - Daily quota reset (24h sliding window)
    - Monthly quota reset (calendar month boundary)
    - Background reset при первом request after reset time
  - **Quota Service** (`internal/services/quota/service.go`):
    - `CheckQuota()` - проверка before request processing
    - `RecordUsage()` - tracking actual usage after request
    - `IncrementConcurrent() / DecrementConcurrent()` - concurrent tracking
    - `GetQuotaStats()` - статистика для UI display
  - **Quota Middleware** (`internal/api/middleware/quota.go`):
    - Автоматическая проверка квот для chat/completion endpoints
    - 429 Too Many Requests при quota exceeded
    - Concurrent request tracking with defer cleanup
  - **Prometheus Integration**:
    - `ollama_proxy_quota_usage` - Current usage by target_id/type
    - `ollama_proxy_quota_limit` - Quota limits
    - `ollama_proxy_quota_exceeded_total` - Exceeded events counter
    - Periodic collection (30s interval) в MetricsCollector
  - **Admin API** (`/api/admin/quotas`):
    - `GET /quotas` - List all quotas (filter by scope)
    - `POST /quotas` - Create quota
    - `GET /quotas/:id` - Get quota details
    - `PUT /quotas/:id` - Update quota
    - `DELETE /quotas/:id` - Delete quota (cascade delete usage)
    - `GET /quotas/:id/usage` - Get current usage
  - **User API** (`/api/quota/me`):
    - Get current user's quota stats with percentages
    - Tenant-scoped quota support
  - **Database Schema** (migration v43):
    - `quotas` table - quota definitions
    - `quota_usage` table - usage tracking
    - Indexes for efficient queries по scope/target_id
    - Foreign key constraints с cascade delete
  - **Data Models**:
    - `Quota` - quota definition (limits, scope, target)
    - `QuotaUsage` - current usage counters
    - `QuotaStats` - computed stats для UI (percentages, remaining)
    - `QuotaCheck` - result of quota validation

### Changed

- **Router**: Quota service и middleware инициализируются автоматически при наличии database
- **Chat Endpoints**: Quota checking применяется к `/v1/chat/completions`, `/v1/completions`
- **Metrics Collector**: Добавлен сбор quota metrics (usage/limits) каждые 30s

### Technical

- **internal/models/quota.go**: Data models для quotas
- **internal/storage/sqlite/quotas.go**: SQLite CRUD implementation
- **internal/storage/postgresql/stubs.go**: PostgreSQL stubs (v1.11.7+)
- **internal/services/quota/service.go**: Core quota logic
- **internal/api/middleware/quota.go**: Quota enforcement middleware
- **internal/api/handlers/quota.go**: Admin & user API handlers
- **internal/api/router/router.go**: Route registration
- **internal/metrics/prometheus.go**: Quota metrics integration
- **Dependencies**: No new dependencies required

### Fair Usage

- **Quota Hierarchy**: Tenant quota > User quota (tenant takes priority)
- **Unlimited Access**: No quota = unlimited (admin override possible)
- **Soft Enforcement**: Checks before request, records after (no mid-request interruption)
- **Concurrent Safety**: Mutex-protected usage updates для race-free tracking
- **Idempotent Resets**: Safe daily/monthly resets без data loss

### Use Cases

1. **Free Tier Limits**: Set daily/monthly token quotas для free users
2. **Paid Plan Enforcement**: Different quotas per subscription tier
3. **Team Quotas**: Tenant-level quotas для shared team resources
4. **Model Access Control**: Restrict expensive models to premium users
5. **Fair Usage Policy**: Prevent resource exhaustion from single user
6. **Cost Control**: Track and limit token consumption for budget management
7. **Concurrent Throttling**: Limit simultaneous requests per user/tenant

### Future Enhancements

- Soft limits vs hard limits (warnings before enforcement)
- Quota alerts/notifications (email/webhook when 80% usage)
- Time-based quotas (hourly, weekly)
- Cost-based quotas (dollar amounts instead of tokens)
- Quota templates для quick assignment
- Bulk quota operations (assign to multiple users)
- Storage quota enforcement для file uploads
- Conversation count enforcement

## [1.11.6] - 2025-10-25

### Added

- **METRICS-01: Prometheus Metrics Export** 📊
  - **Comprehensive Metrics Collection** для monitoring и observability
  - **HTTP Metrics**:
    - Request rate (by method, endpoint, status)
    - Request duration histograms (p50, p95, p99)
    - Response size histograms
    - Active connections gauge
  - **API Usage Metrics**:
    - Tokens used (by api_key, model, type: prompt/completion)
    - API requests (by model, status: success/error)
    - API cost tracking (if pricing enabled)
  - **Model Metrics**:
    - Model request duration histograms (by model)
    - Models loaded gauge
    - Model errors (by model, error_type)
  - **System Metrics**:
    - Goroutines count
    - Memory usage (alloc, sys, heap_alloc, heap_sys, heap_inuse, stack_inuse)
    - DB connections
    - API keys/users/tenants total counts
  - **Authentication Metrics** (v1.11+):
    - Auth attempts (by type: jwt/oidc/ldap/api_key, status)
    - Auth duration by type
  - **Quota Metrics** (future-ready):
    - Quota usage/limits/exceeded events
  - **Prometheus Middleware** для автоматического сбора HTTP metrics
  - **Usage Tracking Integration** экспортирует API/model metrics в Prometheus
  - **MetricsCollector** с periodic collection system metrics (30s interval)
  - **Grafana Dashboard Template** (`docs/grafana-dashboard.json`):
    - HTTP request rate & duration panels
    - Token usage rate by model
    - Model latency (p95) by model
    - Top 5 models by request rate
    - System health (goroutines, memory, models loaded)
  - **Setup Documentation** (`docs/PROMETHEUS_SETUP.md`):
    - Installation guide (Prometheus, Grafana)
    - Docker Compose setup example
    - Alerting examples
    - PromQL query examples
    - Troubleshooting guide

### Changed

- **Middleware Order**: Prometheus middleware добавлен after OpenTelemetry tracing
- **Usage Tracking**: Интегрирован с Prometheus metrics export
- **MetricsCollector**: Запускается автоматически при `metrics.enabled=true`

### Technical

- **internal/metrics/prometheus.go**: Все определения Prometheus metrics
- **internal/api/middleware/prometheus.go**: HTTP metrics middleware
- **internal/api/middleware/usage_tracking.go**: Интеграция с Prometheus
- **cmd/server/main.go**: Запуск MetricsCollector
- **Dependencies**: `github.com/prometheus/client_golang v1.17+`
- **Endpoint**: `GET /metrics` (configurable via `metrics.prometheus_path`)
- **Configuration**: `metrics.enabled`, `metrics.prometheus_path` в config.yaml

### Observability

- **Enterprise-Ready Monitoring**: Полная интеграция с Prometheus/Grafana stack
- **Production Metrics**: Low-cardinality labels для efficient storage
- **Real-Time Visibility**: Automatic metrics collection без manual instrumentation
- **Alerting Support**: Ready-to-use alert rules в documentation

### Use Cases

1. **Production Monitoring**: Track HTTP performance, API usage, model latency
2. **Capacity Planning**: Monitor resource usage (memory, goroutines, connections)
3. **Performance Optimization**: Identify slow endpoints и models
4. **Cost Tracking**: Monitor token usage per API key/model
5. **SLA Compliance**: Track request success rate и latency percentiles
6. **Incident Response**: Real-time dashboards для quick troubleshooting

## [1.11.5] - 2025-10-25

### Added

- **RBAC-01: Custom Roles & Permissions** 🛡️
  - **Fine-Grained Permission System** с wildcard support (*:*, api_keys:*, *:create)
  - **Role Management** (system + custom roles)
    - **System Roles**: super_admin, admin, user, api_manager, read_only
    - **Custom Roles**: создание tenant-specific или global ролей
  - **Permission Types** (23+ permissions):
    - API Keys: create, read, update, delete, revoke
    - Users: create, read, update, delete, manage
    - Tenants: create, read, update, delete, manage
    - Files: upload, read, delete, manage
    - Conversations: read, delete
    - Usage Stats: view
    - Backups: create, restore
    - Models: view, manage
    - System: admin, read
  - **RBAC Service** с permission checking и user roles resolution
  - **RBAC Middleware** для Gin:
    - `RequirePermission(permission)` - проверка одного разрешения
    - `RequireAnyPermission(...)` - проверка любого из разрешений
    - `RequireAllPermissions(...)` - проверка всех разрешений
    - `RequireRole(roleName)` - проверка роли по имени
  - **Database CRUD** для permissions, roles, role_permissions, user_roles
  - **Admin UI** для RBAC Management:
    - Roles management (create, edit, delete, assign permissions)
    - User roles assignment/removal
    - Permissions reference viewer
    - Quick stats dashboard
  - **API Endpoints** (`/api/admin/rbac/*`):
    - `GET /permissions` - список всех системных permissions
    - `GET /roles`, `POST /roles`, `PUT /roles/:id`, `DELETE /roles/:id`
    - `GET /roles/:id/permissions`, `POST /roles/:id/permissions`
    - `GET /users/:id/roles`, `POST /users/:id/roles`, `DELETE /users/:id/roles/:role_id`
    - `GET /users/:id/permissions` - computed permissions от всех ролей

### Changed

- **Database Schema** (migration v42):
  - Таблицы: `permissions`, `roles`, `role_permissions`, `user_roles`
  - Индексы для оптимизации permission checks
- **Admin Panel** добавлена вкладка RBAC с quick stats и link на full RBAC manager

### Technical

- **models/rbac.go**: RBACPermission, Role, UserRole data models
- **services/rbac/**: Service, Permissions definitions
- **api/middleware/rbac.go**: RBAC middleware для authorization
- **api/handlers/rbac.go**: CRUD handlers для roles/permissions
- **web/admin-rbac.html**, **web/js/admin-rbac.js**: Full-featured RBAC UI
- **Wildcard Support** в permission matching (resource:*, *:action, *:*)

### Security

- **Granular Access Control**: Замена грубой admin/non-admin логики на fine-grained permissions
- **Tenant Isolation**: Роли могут быть tenant-specific или global
- **System Roles Protection**: System roles (super_admin, admin) не могут быть удалены или изменены
- **Super Admin Bypass**: Super admins автоматически проходят все permission checks

### Use Cases

1. **Content Manager Role**: upload/read/delete files, но не может управлять users
2. **API Manager Role**: create/read/update/delete API keys, но не может создавать users
3. **Read-Only Admin**: view usage stats, models, system info без возможности изменений
4. **Tenant Admin**: управление users/members внутри своего tenant, но не global admin
5. **Custom Business Roles**: например "Support Agent", "Billing Manager", "DevOps Engineer"

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

