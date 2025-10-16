# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
