# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
