
UPDATE changelogs SET content = '## [2.0.0] - 2025-10-27

### 🚀 Major Features

- **RAG System (Retrieval-Augmented Generation)**: Полная интеграция системы RAG для работы с внешними источниками данных
  - Поддержка REST API, PostgreSQL, File Upload, Web Scraping
  - Semantic chunking с intelligent text splitting
  - Vector embeddings через Ollama (mxbai-embed-large, nomic-embed-text)
  - PgVector для similarity search с HNSW indexing
  - RAG Orchestrator с reranking и context assembly
  - Query logging для analytics

- **RAG Management WebUI**: Полнофункциональный интерфейс управления RAG
  - User Dashboard (/rag-sources.html) - CRUD операции для личных RAG источников
  - Admin Panel (/admin-rag.html) - управление всеми источниками
  - Chat Integration - RAG toggle, source selector, параметры в /chat.html

- **Chat Export/Import UI**: Полнофункциональный интерфейс экспорта и импорта conversations
  - Export Dropdown в Chat Header: JSON, Markdown, Text форматы
  - Автоматическое скачивание файла с sanitized filename
  - Import Modal: Upload JSON с опциями preserve timestamps/IDs
  - Validation JSON структуры перед импортом
  - Success notification с количеством imported messages

- **Browser Testing Integration**: MCP browser extension для E2E тестирования
  - Chrome automation, accessibility snapshots, screenshots

### 🎨 UI/UX Improvements

- **Modal Windows Centering**: Модалки теперь по центру экрана (horizontal + vertical)
- **Consistent Dashboard Styling**: Единообразный dark theme на всех страницах
- **Login Page Redesign**: Двухколоночный layout с gradient background и info panel
- **Error Messages Styling**: Улучшенный контраст и visibility
- **RBAC/Audit Page Refactoring**: Удален Bootstrap, full custom CSS
- **Export/Import Dropdown**: Stylish dropdown menu с animations

### 🔐 Security & Authentication

- **JWT Token Rotation**: Refresh token rotation для enhanced security
- **Authentication Flow Fixes**: Исправлен logout loop
- **Public API Endpoints**: /api/models доступен без аутентификации
- **RAG Credentials Encryption**: AES-256 шифрование credentials

### 📊 Logging & Monitoring

- **Separate Error Logging**: Dedicated error log file с rotation
- **Audit Events Metadata Fix**: JSON serialization для metadata

### 🛠️ Technical Improvements

- **API Client Enhancements**: Generic HTTP methods + RAG/RBAC/Audit methods
- **Context Parsing Fix**: User/Tenant ID prefix stripping
- **CSS Conflicts Resolution**: Исправлено позиционирование модалок
- **Navigation Component**: Поддержка новых RAG страниц
- **Export API Integration**: Frontend integration для conversation export/import

### 📚 Documentation

- RAG Deployment Guide, Config Guide, Testing Guide
- Error Logging Guide

### 🔧 Configuration

- **RAG Configuration**: Полная секция rag в config
- **Logging Enhancements**: error_log_* настройки

### 🗃️ Database

- **RAG Schema**: rag_data_sources, rag_documents, rag_chunks, rag_jobs, rag_query_logs

### 🧪 Testing

- **Go Unit Tests**: 70+ tests для RAG components
- **Playwright E2E Tests**: Browser-based UI testing

### 🐛 Bug Fixes

- Fixed modal windows appearing off-center
- Fixed logout loop when refresh token is blacklisted
- Fixed user_id UUID parsing with prefix
- Fixed audit events metadata serialization error
- Fixed white-on-white text readability
- Fixed RBAC page non-clickable buttons
- Fixed MCP Catalog styling issues

### ⚡ Performance

- Worker Pool для document processing
- Batch embeddings для Ollama
- Connection pooling для HTTP и PostgreSQL

### 🔄 Breaking Changes

- **Version Jump**: 1.12.3 → 2.0.0 (major release)
- **New Dependencies**: PostgreSQL with pgvector, Ollama с embedding models
- **Configuration Changes**: Новый раздел rag в config
- **Database Schema**: Новые таблицы для RAG

### 📦 Dependencies

- Added github.com/pgvector/pgvector-go
- Added Playwright для E2E testing
- Added lumberjack для log rotation' 
WHERE version = '2.0.0';
    