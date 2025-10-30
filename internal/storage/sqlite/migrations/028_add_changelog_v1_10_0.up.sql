
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.0', '2025-10-16', '## [1.10.0] - 2025-10-16

### Added
- **FILE-STORAGE-01: Universal File Storage & Processing System** ✅
  - Storage Backends: Local filesystem и S3-compatible (MinIO)
  - Document Extractors: PDF, DOCX, TXT, CSV с автоматическим определением кодировки
  - Database Integration: Таблицы files, file_access_logs, message_files
  - API Endpoints: /api/files/* для upload, download, delete, list
  - WebUI: Страница Files для управления файлами
  - Admin Panel: Новая вкладка Files для управления всеми файлами
  - Chat Integration: Прикрепление файлов к сообщениям
  - LLM Context Enrichment: Автоматическое включение содержимого файлов
  - Path Traversal Prevention: Robust защита от path traversal
  - Unicode Filenames: Полная поддержка Unicode (Cyrillic, Chinese, Emoji)

- **Cross-Platform PDF Text Extraction** 🚀
  - Pure Go Library: github.com/ledongthuc/pdf
  - Automatic Fallback: pdftotext → go-pdf
  - Three Methods: auto, pdftotext, go-pdf
  - Docker-Ready: Работает без внешних зависимостей

- **Advanced Text Encoding Detection** 🔍
  - UTF-8 with BOM, UTF-16 LE/BE
  - Windows-1251 Fallback для русского текста
  - Cyrillic Detection
  - Reasonable Text Validation

- **Comprehensive Unit Tests** ✅
  - Validator Tests: 13 тестов (100% pass)
  - Local Storage Tests: 15 тестов (100% pass)
  - Coverage: filestorage 46.7%, storage 29.6%

### Changed
- Chat Messages: Добавлено поле file_ids
- Configuration: Секции file_storage и extractors

### Fixed
- File Upload Integrity: SkipContentValidation flag
- Windows Path Separators: filepath.FromSlash()
- File Deletion: Robust path traversal checks
- Text Encoding: Windows-1251 heuristic detection

### Technical
- Dependencies: github.com/ledongthuc/pdf
- Migrations: v26 files table, v27 message_files junction
- Packages: internal/filestorage, internal/extractors
- Tests: validator_test.go, local_test.go');
	