# AIGateway Desktop - Development Roadmap

> **Platform:** Wails v2 (Go + WebView)  
> **Architecture:** 🎯 **Thin Client** → Remote AIGateway Server  
> **Current Status:** Planning Phase 📋  
> **Target:** Cross-platform desktop app (Windows, macOS, Linux)  
> **Vision:** Native desktop UI для AIGateway с local file system integration  
> **Last Updated:** 2025-10-28  
> **Project Path:** `G:\golang\aigateway-desktop` (отдельный репозиторий)  
> **Related:** [Roadmap.MD](Roadmap.MD) v2.4.0 - Server-side Desktop Support

---

## 🎯 Project Vision

Создать **тонкий нативный desktop-клиент** на базе **Wails v2** для AIGateway Platform.

### Desktop App = Rich UI Client (НЕ standalone server!)

```
┌──────────────────────────────────────┐
│   Desktop App (Wails - Thin Client)  │
│  ┌────────────────────────────────┐  │
│  │  Native UI (WebView)           │  │
│  │  • Chat Interface              │  │
│  │  • Monaco Editor               │  │
│  │  • File Explorer               │  │
│  └────────────────────────────────┘  │
│  ┌────────────────────────────────┐  │
│  │  Local Features (Go)           │  │
│  │  • File System Access          │  │
│  │  • HTTP Client to Server       │  │
│  │  • Local Cache                 │  │
│  │  • API Key Storage             │  │
│  └────────────────────────────────┘  │
└──────────────┬───────────────────────┘
               │ HTTPS + API Key
               │ (auto-generated)
               ▼
┌──────────────────────────────────────┐
│  AIGateway Server (Remote)           │
│  • AI Chat Processing                │
│  • RAG Search & Indexing             │
│  • MCP Servers Management            │
│  • User Management                   │
│  • Database (PostgreSQL/SQLite)      │
│  └────────┬─────────────────────────┘
│           ▼
│  ┌────────────────────────────┐
│  │  Ollama / vLLM            │
│  │  (LLM Inference)          │
│  └────────────────────────────┘
└──────────────────────────────────────┘
```

**Desktop App делает:**

- 💻 **Native UI** - красивый интерфейс без браузера
- 📁 **File System Access** - работа с локальными файлами и проектами
- 📝 **Monaco Editor** - code editing с syntax highlighting
- 🎯 **System Integration** - tray icon, global hotkeys, native notifications
- 💾 **Local Cache** - offline mode с синхронизацией при reconnect
- 🔐 **Secure Auth** - auto-generated API keys для каждого device
- 📤 **File Upload** - загрузка локальных файлов в RAG на server

**Remote Server делает:**

- 🤖 **All AI Logic** - chat processing, embeddings, inference
- 🗄️ **Database** - все данные хранятся централизованно
- 🔐 **Authentication** - user management, permissions, RBAC
- 📊 **Rate Limiting** - контроль использования API
- 🔌 **Ollama/vLLM** - LLM inference engines
- 📚 **RAG Processing** - vector search, indexing
- 🔌 **MCP Servers** - Model Context Protocol management

**Ключевые преимущества**:

- ⚡ **Производительность** - WebView ~50 MB vs Chromium ~150 MB (Electron)
- 💾 **Память** - ~100 MB vs ~300 MB (Electron)
- 🔐 **Security** - один раз login → permanent API key для device
- 📱 **Multi-Device** - один аккаунт на web, desktop, mobile
- 🌐 **Scalability** - server независимо масштабируется
- 🔄 **Updates** - server обновляется без переустановки desktop app
- 📦 **Размер** - ~10-20 MB installer (без embedded server!)

---

## 🔐 Authentication Architecture

### One-Time Password Flow → Long-Lived API Key

```
┌─────────────────────────────────────────────────────────┐
│ 1. First Launch (User Login)                            │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Desktop App                     AIGateway Server       │
│  ───────────                     ─────────────────      │
│                                                          │
│  Login Screen                                            │
│  ┌──────────────────────┐                               │
│  │ Server URL:          │                               │
│  │ [aigateway.com]      │                               │
│  │                      │                               │
│  │ Username: [user]     │                               │
│  │ Password: [****]     │   POST /api/auth/login        │
│  │                      │  ─────────────────────────>   │
│  │      [Login]         │   {username, password}        │
│  └──────────────────────┘                               │
│                                  │                       │
│                                  ▼                       │
│                            Verify credentials            │
│                            Generate JWT token            │
│                                  │                       │
│                      <───────────┘                       │
│                     { token: "eyJhbG..." }               │
│                                                          │
│  Store JWT temporarily                                   │
│  (in memory, not persisted)                              │
│                                                          │
│                        POST /api/auth/devices/register   │
│                       ───────────────────────────>       │
│                       Authorization: Bearer eyJhbG...    │
│                       {                                  │
│                         "device_name": "Desktop-Win",    │
│                         "device_info": {                 │
│                           "os": "windows",               │
│                           "hostname": "PC-123",          │
│                           "app_version": "0.1.0"         │
│                         }                                │
│                       }                                  │
│                                  │                       │
│                                  ▼                       │
│                         Create API Key for device        │
│                         Name: "Desktop App - Windows..." │
│                         Models: ["*"]                    │
│                         Never expires (or 90 days)       │
│                                  │                       │
│                      <───────────┘                       │
│                     {                                    │
│                       "api_key": "sk-desktop_abc123",    │
│                       "key_id": "key_xyz",               │
│                       "device_id": "device_123"          │
│                     }                                    │
│                                                          │
│  Discard JWT token                                       │
│  Save API Key to:                                        │
│  ~/.aigateway/config.json                                │
│  {                                                       │
│    "server_url": "https://aigateway.com",                │
│    "api_key": "sk-desktop_abc123",                       │
│    "key_id": "key_xyz",                                  │
│    "device_id": "device_123"                             │
│  }                                                       │
│                                                          │
│  → Navigate to Chat Screen                               │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ 2. Subsequent Launches (Automatic Auth)                 │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  Load API Key from config.json                           │
│                                                          │
│                       GET /api/health                    │
│                       ───────────────────────────>       │
│                       Authorization: Bearer sk-desktop...│
│                                  │                       │
│                                  ▼                       │
│                         Verify API Key                   │
│                         Update last_seen_at              │
│                                  │                       │
│                      <───────────┘                       │
│                     { "status": "ok" }                   │
│                                                          │
│  → Navigate to Chat Screen                               │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ 3. All API Requests (Using API Key)                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│                       POST /api/chat/completions         │
│                       ───────────────────────────>       │
│                       Authorization: Bearer sk-desktop...│
│                       { "message": "Hello", ... }        │
│                                  │                       │
│                                  ▼                       │
│                         Validate API Key                 │
│                         Check rate limits                │
│                         Process request                  │
│                                  │                       │
│                      <───────────┘                       │
│                     { "response": "..." }                │
│                                                          │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│ 4. Device Management (Web UI)                           │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  User opens Web UI → Devices Page                        │
│                                                          │
│  Devices List:                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │ Desktop App - Windows - 2025-10-28  [Revoke]    │   │
│  │ Last seen: 5 min ago  •  Active                  │   │
│  │                                                  │   │
│  │ Desktop App - MacOS - 2025-10-15    [Revoke]    │   │
│  │ Last seen: 2 days ago  •  Inactive               │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  User clicks [Revoke] →                                  │
│                       DELETE /api/auth/devices/key_xyz   │
│                       ───────────────────────────>       │
│                                  │                       │
│                                  ▼                       │
│                         Revoke API Key                   │
│                                  │                       │
│                      <───────────┘                       │
│                     { "success": true }                  │
│                                                          │
│  Desktop App получает 401 Unauthorized на next request   │
│  → Shows Login Screen                                    │
└─────────────────────────────────────────────────────────┘
```

---

## 📅 Release Roadmap

### Version 0.1.0 - MVP Thin Client (2 недели) 🎯 PRIORITY 1

**Фокус:** Базовое Wails приложение с authentication и chat

**Цели:**

- ✅ HTTP client к remote AIGateway server
- ✅ Auto-generated API keys authentication
- ✅ Базовый chat интерфейс
- ✅ System tray integration
- ✅ Config persistence (~/.aigateway/config.json)

**Задачи:**

- [x] **CLIENT-001** Wails Project Setup ✅
  - Инициализация: `wails init -n aigateway-desktop -t svelte`
  - Project structure
  - Build configuration для Windows/macOS/Linux
  - Icon и app metadata
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-002** Authentication Flow ✅
  - Login screen (Server URL + Username/Password)
  - POST /api/auth/login → получение JWT
  - POST /api/auth/devices/register → создание API Key
  - API Key storage в ~/.aigateway/config.json
  - Auto-login на subsequent launches
  - Device fingerprinting для unique identification
  - CheckAuthStatus() для explicit auth state management
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-003** HTTP Client Layer ✅
  - Go HTTP client с timeout/retry logic (exponential backoff)
  - Authorization header с API Key
  - Error handling (401 → re-login, 429 → rate limit, 5xx → retry)
  - Request/Response logging (опционально)
  - Connection pooling (MaxIdleConns=100)
  - Custom error types (HTTPError, IsUnauthorized, IsRateLimited, IsServerError)
  - Callbacks для 401 и 429 events
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-004** Chat UI (Svelte) ✅
  - Портирование web/chat.html → ChatPanel.svelte
  - Message list с markdown rendering (marked.js)
  - Model selection dropdown (GET /api/models)
  - Send message → POST /api/chat/completions
  - Typing indicator animation
  - Auto-scroll to bottom (smart scroll - не прилипает)
  - Clear chat function
  - Input focus retention после отправки
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-005** System Integration ✅
  - Window state persistence (размер, позиция, maximized) - CLIENT-005.1
  - System tray icon с context menu (Show/Hide/Quit) - CLIENT-005.2
  - Global hotkey (Ctrl+Shift+A / Cmd+Shift+A) для show/hide - CLIENT-005.3
  - **Minimize to tray** - X button скрывает окно вместо закрытия
  - Auto-cleanup on shutdown
  - **Cross-platform:** Windows ✅ | macOS ✅ | Linux ✅
  - **Libraries:** `getlantern/systray` (tray), `golang.design/x/hotkey` (hotkeys)
  - **Status:** Завершено 2025-10-28

**Acceptance Criteria:**

- ✅ User может войти с username/password
- ✅ API Key автоматически создается и сохраняется
- ✅ Chat работает через HTTP к remote server
- ✅ System tray + hotkey функционируют
- ✅ Размер .exe < 30 MB, память < 150 MB

**Время:** ~10-14 дней

---

### Version 0.2.0 - File System Integration (2 недели) 🎯 PRIORITY 2

**Фокус:** Работа с локальными файлами

**Задачи:**

- [x] **CLIENT-006** File Picker ✅
  - Wails native file dialog (OpenFileDialog, OpenMultipleFilesDialog)
  - File info display (name, size, type)
  - File size limit (10 MB для ReadFileContent)
  - UI integration в ChatPanel
  - Note: Drag & drop ограничен Wails WebView (HTML5 File API недоступен)
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-007** File Upload to RAG ✅
  - Upload файлов через multipart/form-data
  - POST /api/files/upload (uses existing server endpoint)
  - Progress indicator для каждого файла (status badges)
  - Batch upload (sequential processing)
  - File size limit: 50 MB per file
  - Automatic text extraction на сервере (extract=true)
  - Success/Error feedback с результатами
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-008** Project Management ✅
  - Open folder dialog (runtime.OpenDirectoryDialog)
  - Recent projects list (max 10, persist in config.json)
  - Current project tracking
  - ProjectInfo metadata (path, name, last_opened, file_count, is_indexed)
  - List project files (recursive/non-recursive with .gitignore patterns)
  - Tab navigation UI (Chat | Projects)
  - Remove from recent projects
  - Project icons based on name heuristics
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-009** Basic File Viewer ✅
  - Display text files в read-only mode
  - Syntax highlighting (highlight.js с 13 языками)
  - File tree navigation (left sidebar с search)
  - File metadata (size, type, path)
  - File icons по типам (emoji-based)
  - Breadcrumb navigation (Back to Projects)
  - .gitignore patterns (node_modules, .git, vendor, etc.)
  - GitHub Dark theme для syntax highlighting
  - Empty states для no files/no selection
  - Integration с ProjectManager component
  - **Status:** Завершено 2025-10-28

**Acceptance Criteria:**

- ✅ User может выбрать файл через dialog
- ✅ Файл загружается на server для RAG indexing
- ✅ Recent projects saved locally
- ✅ Basic file browsing работает

**Время:** ~10-14 дней

---

### Version 0.3.0 - Monaco Editor Integration (3 недели) 🎯 PRIORITY 3

**Фокус:** Полноценный code editor

**Задачи:**

- [x] **CLIENT-010** Monaco Editor Setup ✅
  - Интеграция Monaco Editor в Svelte (monaco-editor npm package)
  - Multi-tab support (открытие нескольких файлов)
  - Theme support (VS Dark встроенная)
  - Language detection по расширению (20+ languages)
  - Modified indicator (● для несохраненных файлов)
  - Tab management (close, switch tabs)
  - Event-driven integration с FileViewer
  - Breadcrumb navigation (Back to Files)
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-011** Editor Features ✅
  - Syntax highlighting для 50+ языков (built-in Monaco)
  - Code completion (IntelliSense из Monaco) (built-in)
  - Find & Replace (built-in Ctrl+F, Ctrl+H)
  - Code folding, minimap (built-in)
  - Go to definition, Find references (built-in)
  - Command palette (F1)
  - **Note:** Все features уже встроены в Monaco Editor
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-012** File Editing ✅
  - Edit local files (✅ работает)
  - Save functionality (✅ Ctrl+S, кнопка)
  - Save As functionality (✅ Ctrl+Shift+S, native dialog)
  - Unsaved changes indicator (✅ ● в табе)
  - Unsaved changes confirmation при close (✅ работает)
  - Keyboard shortcuts (✅ Ctrl+S, Ctrl+Shift+S)
  - Auto-disable Save button when no changes (✅)
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-013** Chat Integration ✅
  - Send code snippet to chat (✅ context menu)
  - "Explain this code" action (✅ реализовано)
  - Auto-switch to Chat tab (✅)
  - Keyboard shortcut Ctrl+Shift+C (✅)
  - Language-aware code blocks (✅)
  - Apply AI suggestions к файлу (diff view) (TODO: future feature)
  - **Status:** Завершено 2025-10-28

- [x] **CLIENT-013.1** Markdown Preview ✅
  - Toggle button для .md файлов (👁️ Preview / 📝 Source)
  - Markdown rendering с помощью `marked.js` (GitHub-flavored)
  - **Syntax highlighting** в code blocks (`highlight.js` с GitHub Dark темой)
  - Auto-detect языка программирования если не указан
  - Поддержка 50+ языков (JavaScript, Python, Go, Rust, TypeScript, SQL, Bash, etc.)
  - GitHub-like dark theme styling
  - Поддержка всех Markdown elements (headers, code blocks, tables, etc.)
  - Auto-reset preview mode при переключении файлов
  - Disable Save button в preview mode
  - Полностью responsive layout (max-width 900px)
  - **Status:** Завершено 2025-10-28

**Acceptance Criteria:**

- ✅ Monaco Editor полностью функционален
- ✅ Можно редактировать несколько файлов
- ✅ Интеграция с chat работает
- ✅ Performance не деградирует (< 200 MB memory)

**Время:** ~21 день

---

### Version 0.4.0 - WebSocket Streaming (2 недели) 🎯 PRIORITY 4

**Фокус:** Real-time streaming responses

**Зависимости:** Требует server v2.4.3 (WebSocket endpoint)

**Задачи:**

- [x] **CLIENT-014** WebSocket Client ✅
  - ✅ Подключение к /ws/chat?token={api_key}
  - ✅ Auto-reconnect logic (exponential backoff)
  - ✅ Ping/Pong для keep-alive (30s interval)
  - ✅ Event-based API (connect, disconnect, error, message, reconnecting, maxReconnectsReached)
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-015** Streaming UI ✅
  - ✅ Отображение partial responses в real-time
  - ✅ Typing indicator (3 bouncing dots)
  - ✅ Stop generation button (⏹️)
  - ✅ Smooth text append без flickering
  - **Status:** Завершено 2025-10-28
  
- [x] **CLIENT-016** Fallback to HTTP ✅
  - ✅ Detect WebSocket unavailability (max reconnects reached)
  - ✅ Automatic fallback to POST `/v1/chat/completions`
  - ✅ Graceful degradation с user notification
  - ✅ Connection status indicator (🟢 WebSocket / 🟠 HTTP / 🔵 Connecting)
  - ✅ Manual retry mechanism (click на HTTP indicator)
  - ✅ `connectionMode` state tracking
  - **Status:** Завершено 2025-10-28

**Acceptance Criteria:**

- ✅ WebSocket streaming работает (real-time character-by-character)
- ✅ Auto-reconnect на disconnect (10 attempts, exponential backoff)
- ✅ Fallback to HTTP если WS fail (automatic + manual retry)
- ✅ UI indicator показывает connection status
- ✅ Graceful degradation без потери функциональности

**Время:** ~14 дней (Завершено 2025-10-28)

---

### Version 0.4.1 - Hotfixes & Polish (3 дня) 🐛 PRIORITY 4.1

**Фокус:** Критичные bug fixes после v0.4.0

**Задачи:**

- [ ] **CLIENT-014.1** Conversations Management (COMPLETED)
  - ✅ Create new conversation (SaveConversation backend)
  - ✅ Load conversation history (GetConversations)
  - ✅ Switch between conversations (UI state management)
  - ✅ Delete conversations (DeleteConversation)
  - ✅ Conversation metadata (title, model, created_at, updated_at)
  - ✅ Auto-generate titles from first message
  - ✅ Persist conversations in config.json
  - **Status:** Завершено 2025-10-28
  
- [ ] **CLIENT-014.2** UI State Persistence (COMPLETED)
  - ✅ Save/Restore panel sizes (left, right panels)
  - ✅ Save/Restore current chat ID
  - ✅ Save/Restore selected model
  - ✅ Save/Restore conversations list height
  - ✅ Debounced saving (prevent excessive writes)
  - ✅ Integration с Wails runtime
  - **Status:** Завершено 2025-10-28
  
- [ ] **CLIENT-014.3** Open Files Persistence (COMPLETED)
  - ✅ Track open files in EditorPanel (path, pinned, modified)
  - ✅ Save open files to config on change
  - ✅ Restore open files on app startup
  - ✅ Preserve file order and pinned status
  - ✅ `getOpenFiles()` and `restoreOpenFiles()` API
  - **Status:** Завершено 2025-10-28
  
- [ ] **CLIENT-014.4** Resizable Conversations List (COMPLETED)
  - ✅ Vertical resize handle для conversations list
  - ✅ Save/Restore conversations height в UI state
  - ✅ Min/Max constraints (150px - 600px)
  - ✅ Smooth resize с throttling
  - ✅ Cursor feedback (ns-resize)
  - **Status:** Завершено 2025-10-28

- [ ] **CLIENT-030** 🐛 Fix: Models не загружаются при первом запуске
  - **Issue:** После свежей установки нужно перезапустить app для загрузки моделей
  - **Root Cause:** Race condition между auth и models loading
  - **Solution:**
    - Добавить explicit `loadModels()` после successful auth
    - Retry механизм для models loading (3 attempts)
    - Loading state в UI (spinner в model dropdown)
    - Fallback на первую available модель если `selectedModel` empty
  - **Testing:**
    - Удалить `~/.aigateway/config.json`
    - Первый запуск → login → models должны загрузиться сразу
  - **Priority:** 🔴 CRITICAL

**Acceptance Criteria:**

- ✅ Models loading работает с первого запуска (no restart needed)
- ✅ UI показывает loading state при загрузке моделей
- ✅ Graceful fallback если models не загрузились

**Время:** ~3 дня

---

### Version 0.5.0 - Advanced Features (3 недели) 🎯 PRIORITY 5

**Фокус:** Terminal, Git, Settings

**Задачи:**

- [ ] **CLIENT-017** Embedded Terminal (xterm.js)
  - Terminal в bottom panel
  - Multiple terminal tabs
  - Shell selection (bash/powershell)
  
- [ ] **CLIENT-018** Git Integration
  - Detect .git folder
  - Show git status в file tree
  - Diff viewer для changes
  - Скрывать .git каталог при просмотре проекта
  
- [ ] **CLIENT-019** Settings UI
  - Server configuration
  - Theme selection
  - Keyboard shortcuts
  - Auto-start preferences
  
- [ ] **CLIENT-020** Device Management
  - Show all user devices (GET /api/auth/devices)
  - Self-revoke button
  - Device info display
  
- [ ] **CLIENT-029** LSP/Linter Integration
  - **Language Server Protocol (LSP) Client:**
    - Go: `gopls` integration для Go files
    - Python: `pyright` или `pylance` для Python files
    - TypeScript/JavaScript: Built-in Monaco support (уже работает)
    - Rust: `rust-analyzer` (опционально)
  - **Real-time Diagnostics:**
    - Error/Warning markers в editor gutter
    - Problem panel (bottom) с списком issues
    - Severity indicators (error 🔴, warning 🟡, info 🔵)
  - **Quick Fixes:**
    - Lightbulb 💡 для available actions
    - Auto-import missing packages
    - Fix typos, unused variables
  - **Code Actions:**
    - Refactoring suggestions
    - Extract function/variable
    - Organize imports
  - **Performance:**
    - LSP process management (spawn/kill)
    - Debounced diagnostics (300ms delay)
    - Background processing (не блокирует UI)
  - **Configuration:**
    - Enable/Disable LSP per language в Settings
    - LSP binary path configuration
    - Custom LSP settings (formatOnSave, etc.)
  - **Dependencies:**
    - Requires external LSP binaries (gopls, pyright)
    - Auto-detect installed LSPs или prompt user to install
  - **Note:** Это опциональная feature - editor работает без LSP

**Acceptance Criteria:**

- ✅ Terminal работает
- ✅ Git status видно
- ✅ Settings UI функционален
- ✅ LSP diagnostics показывают errors/warnings в real-time
- ✅ Quick fixes доступны через lightbulb menu

**Время:** ~21 день

---

### Version 0.6.0 - Offline Mode & Cache (2 недели) 🎯 PRIORITY 6

**Фокус:** Local caching и offline support

**Задачи:**

- [ ] **CLIENT-021** Local Cache (SQLite)
  - Cache chat history локально
  - Cache user info, models list
  - TTL для cached data
  
- [ ] **CLIENT-022** Offline Queue
  - Queue requests когда offline
  - Auto-sync при reconnect
  - Conflict resolution
  
- [ ] **CLIENT-023** Sync Status
  - Online/offline indicator
  - Sync progress bar
  - Last synced timestamp

**Acceptance Criteria:**

- ✅ App работает offline (readonly)
- ✅ Queue накапливает requests
- ✅ Auto-sync при online

**Время:** ~14 дней

---

### Version 0.7.0 - Polish & Packaging (2 недели) 🎯 PRIORITY 7

**Фокус:** Production-ready release

**Задачи:**

- [ ] **CLIENT-024** Packaging
  - Windows installer (NSIS/WiX)
  - macOS .app + DMG
  - Linux AppImage
  - Code signing
  
- [ ] **CLIENT-025** Auto-Updates
  - Update checker
  - Download & install updates
  - Release notes display
  
- [ ] **CLIENT-026** Error Handling
  - Graceful error messages
  - Crash reporting (optional)
  - Logs export
  
- [ ] **CLIENT-027** Documentation
  - User guide
  - Keyboard shortcuts reference
  - Troubleshooting FAQ

**Acceptance Criteria:**

- ✅ Installers для всех платформ
- ✅ Auto-update работает
- ✅ Error handling polished

**Время:** ~14 дней

---

### Version 0.8.0 - Agentic Mode & MCP Integration (4-5 недель) 🤖 PRIORITY 8

**Фокус:** AI Agent с autonomous capabilities (как Windsurf, Cursor)

**Зависимости:**

- Requires server v2.5.0+ (Agent API endpoints)
- Requires Terminal (CLIENT-017) для command execution
- Requires Monaco Editor (CLIENT-010-012) для code modifications

**Архитектура Agentic Mode:**

```
┌──────────────────────────────────────────────────────────┐
│  User Request → AI Agent (on server)                     │
│                      │                                    │
│                      ▼                                    │
│              Task Planning & Decomposition                │
│              (разбиение на подзадачи)                     │
│                      │                                    │
│                      ▼                                    │
│         ┌────────────┴────────────┐                       │
│         │    Tool Selection       │                       │
│         └────────────┬────────────┘                       │
│                      │                                    │
│      ┌───────────────┼───────────────┐                    │
│      ▼               ▼               ▼                    │
│  File Ops      Terminal Cmds     MCP Tools                │
│  (read/write)  (bash/powershell) (server catalog)        │
│      │               │               │                    │
│      └───────────────┼───────────────┘                    │
│                      │                                    │
│                      ▼                                    │
│              Execution Results                            │
│              (success/failure)                            │
│                      │                                    │
│                      ▼                                    │
│         User Approval / Auto-approve                      │
│         (для destructive operations)                      │
│                      │                                    │
│                      ▼                                    │
│              Apply Changes + Rollback                     │
└──────────────────────────────────────────────────────────┘
```

**Задачи:**

- [ ] **CLIENT-028** Agentic Mode Framework
  
  **28.1. Agent Communication Protocol:**
  - WebSocket или SSE для agent streaming messages
  - Message types:
    - `agent_thinking` - показывает что agent планирует
    - `agent_tool_use` - запрос на использование tool
    - `agent_tool_result` - результат tool execution
    - `agent_approval_needed` - требуется подтверждение user
    - `agent_completed` - task завершена
  - Structured agent responses (JSON schema)
  
  **28.2. Task Planning UI:**
  - Agent thinking indicator (animated brain 🧠)
  - Task decomposition tree view (parent → subtasks)
  - Progress tracking (N/M tasks completed)
  - Current task highlight
  - Estimated time remaining (if available)
  
  **28.3. Tool Execution UI:**
  - Tool cards в chat:

    ```
    ┌────────────────────────────────────┐
    │ 🔧 File Operation                  │
    │ Action: Write file                 │
    │ Path: src/main.go                  │
    │ Changes: +45 lines, -12 lines      │
    │ [ View Diff ] [ Approve ] [ Deny ] │
    └────────────────────────────────────┘
    ```

  - Expandable tool details (показать полный diff)
  - Batch approval (approve all pending)
  - Auto-approve toggle для trusted operations
  
  **28.4. Approval Flow:**
  - **Require approval для:**
    - File deletion/overwrite
    - Terminal commands с `sudo`, `rm -rf`, etc.
    - Network requests (curl, wget)
    - Git operations (commit, push)
  - **Auto-approve для:**
    - File read operations
    - Non-destructive commands (ls, cat, echo)
    - MCP tool queries (если whitelisted)
  - Approval timeout (30 seconds → auto-deny)
  - Approval history log
  
  **28.5. Rollback Mechanism:**
  - Snapshot files before modification (git-like)
  - Undo last N operations (stack-based)
  - "Restore Previous Version" button
  - Diff viewer для rollback preview
  
  **28.6. Agent Settings:**
  - Enable/Disable agentic mode toggle
  - Auto-approve preferences (per tool type)
  - Max tool use per task (limit 50)
  - Timeout settings (per task, per tool)
  
- [ ] **CLIENT-031** MCP Servers Support
  
  **31.1. MCP Catalog Browser:**
  - GET /api/mcp/catalog → список available MCP servers
  - MCP card UI:

    ```
    ┌────────────────────────────────────┐
    │ 🔌 GitHub MCP Server               │
    │ Description: GitHub API integration│
    │ Tools: 12 available                │
    │ Status: ● Enabled                  │
    │ [ Configure ] [ Disable ]          │
    └────────────────────────────────────┘
    ```

  - Filter по categories (filesystem, web, database, etc.)
  - Search MCP servers
  
  **31.2. MCP Tool Management:**
  - Browse tools per MCP server
  - Tool schema display (input/output parameters)
  - Enable/Disable tools individually
  - Tool usage statistics (how many times used)
  
  **31.3. MCP Tool Execution:**
  - Agent can request MCP tool use
  - POST /api/agent/tools/execute

    ```json
    {
      "tool": "github.create_issue",
      "mcp_server": "github-mcp",
      "parameters": {
        "repo": "user/repo",
        "title": "Bug report",
        "body": "Description..."
      }
    }
    ```

  - Tool result display в chat
  - Error handling (tool not found, execution failed)
  
  **31.4. MCP Configuration:**
  - Per-MCP server settings (API keys, base URLs)
  - Sync MCP configs from server (централизованное управление)
  - Local overrides для development
  
  **31.5. Security:**
  - Whitelist MCP servers (only from our catalog)
  - Rate limiting per MCP server (max 100 calls/min)
  - Tool execution logs (audit trail)
  - Dangerous tool warnings (delete operations, etc.)

- [ ] **CLIENT-032** Tool System (File & Terminal)
  
  **32.1. File Operations Tools:**
  - `file.read(path)` - read file content
  - `file.write(path, content)` - write/overwrite file
  - `file.create(path, content)` - create new file (fail if exists)
  - `file.delete(path)` - delete file (requires approval)
  - `file.list(directory)` - list files в directory
  - `file.search(pattern, directory)` - search files by pattern
  - All operations track changes для rollback
  
  **32.2. Terminal Execution Tools:**
  - `terminal.execute(command, cwd)` - run command
  - Command output streaming (real-time)
  - Exit code capture
  - Timeout handling (kill after 60s)
  - Dangerous command detection:
    - `rm -rf`, `sudo`, `format`, `dd`, `mkfs`
    - Require explicit approval
  - Shell selection (bash/powershell/zsh)
  
  **32.3. Tool Registry:**
  - Central tool registry на server
  - Tool versioning (v1, v2 APIs)
  - Tool capability advertisement
  - Dynamic tool loading (plugin-like)

- [ ] **CLIENT-033** Agent Progress & History
  
  **33.1. Progress Visualization:**
  - Agent task tree (expandable/collapsible)
  - Real-time progress updates
  - Time elapsed per task
  - Success/Failure indicators (✅ ❌)
  
  **33.2. Agent History:**
  - Conversation with agent actions included
  - "Agent used 5 tools" summary
  - Expandable tool use details
  - Export agent session (for debugging)
  
  **33.3. Agent Metrics:**
  - Total tools used per session
  - Success rate (tasks completed / tasks attempted)
  - Average approval time (user responsiveness)
  - Most used tools (charts)

**Acceptance Criteria:**

- ✅ Agent mode активируется через toggle в Settings
- ✅ Agent может планировать и выполнять tasks
- ✅ File operations работают с approval flow
- ✅ Terminal commands выполняются безопасно
- ✅ MCP servers доступны из catalog
- ✅ MCP tools выполняются через agent
- ✅ Rollback mechanism работает (undo file changes)
- ✅ UI показывает progress и требует approvals
- ✅ Security: dangerous operations require explicit approval
- ✅ Performance: UI не блокируется во время agent work

**Время:** ~28-35 дней (4-5 недель)

**Notes:**

- Это **самая сложная feature** в roadmap
- Требует тесной интеграции с server Agent API
- Security критичен - все destructive operations require approval
- Тестирование должно быть exhaustive (edge cases, timeouts, errors)

---

### Version 1.0.0 - Production Release 🎯 MILESTONE

**Total Development Time:** ~6-8 месяцев

**Requirements:**

- ✅ All v0.1.0-v0.8.0 features implemented:
  - ✅ Authentication & Chat (v0.1.0)
  - ✅ File System Integration (v0.2.0)
  - ✅ Monaco Editor (v0.3.0)
  - ✅ WebSocket Streaming (v0.4.0)
  - ✅ Hotfixes & Polish (v0.4.1)
  - ✅ Terminal, Git, Settings, LSP (v0.5.0)
  - ✅ Offline Mode & Cache (v0.6.0)
  - ✅ Packaging & Auto-Updates (v0.7.0)
  - ✅ Agentic Mode & MCP (v0.8.0) 🤖
- ✅ Test coverage > 70%
- ✅ Performance benchmarks pass:
  - App size < 50 MB
  - Memory < 250 MB (with Agent mode)
  - Startup < 5 seconds
  - Chat latency < 100ms
- ✅ Security audit complete:
  - Agent approval flow tested
  - Dangerous command detection verified
  - MCP tool execution sandboxed
- ✅ Documentation complete:
  - User guide с Agent mode examples
  - MCP integration guide
  - Troubleshooting FAQ
- ✅ Cross-platform testing done (Windows, macOS, Linux)

---

## 🚀 Post 1.0 Features

### Future Enhancements

- [ ] **Multi-Workspace** support
- [ ] **Collaboration** features (real-time code sharing)
- [ ] **Plugin System** для extensions
- [ ] **Jupyter Notebooks** support
- [ ] **Database Explorer** для SQL queries
- [ ] **Docker Integration** (container management)
- [ ] **Mobile Companion App** (iOS/Android viewer)

---

## 📐 Project Structure

```
aigateway-desktop/
├── main.go                    # Wails entry point
├── app.go                     # Go bindings для frontend
├── wails.json                 # Wails configuration
├── build/                     # Build artifacts
│   ├── bin/
│   │   ├── aigateway-desktop.exe      (Windows)
│   │   ├── aigateway-desktop.app      (macOS)
│   │   └── aigateway-desktop          (Linux)
│   └── appicon.png
├── frontend/                  # Svelte app
│   ├── src/
│   │   ├── App.svelte
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── LoginScreen.svelte
│   │   │   │   ├── ChatPanel.svelte
│   │   │   │   ├── EditorPanel.svelte
│   │   │   │   ├── FileExplorer.svelte
│   │   │   │   ├── Terminal.svelte
│   │   │   │   └── Settings.svelte
│   │   │   ├── stores/
│   │   │   │   ├── auth.js
│   │   │   │   ├── chat.js
│   │   │   │   └── files.js
│   │   │   └── api/
│   │   │       └── client.js      # HTTP client wrapper
│   │   └── assets/
│   │       ├── styles/
│   │       └── images/
│   ├── wailsjs/              # Auto-generated Go bindings
│   └── package.json
├── internal/
│   ├── auth/                 # Authentication logic
│   ├── http/                 # HTTP client
│   ├── cache/                # Local caching
│   └── config/               # Config management
└── configs/
    └── default.yaml
```

---

## 🎯 Success Metrics

### Version 0.1.0 (MVP)

- ✅ App размер < 30 MB
- ✅ Memory usage < 150 MB (idle)
- ✅ Startup time < 3 seconds
- ✅ Login flow < 5 seconds

### Version 1.0.0 (Production)

- ✅ App размер < 50 MB
- ✅ Memory usage < 200 MB (с Monaco)
- ✅ Startup time < 5 seconds
- ✅ File upload 1 MB file < 2 seconds
- ✅ Chat latency < 100ms (network excluded)
- ✅ Zero crashes per week average

---

## ⚠️ Risks & Dependencies

### Dependencies on Server

| Desktop Feature | Requires Server Version | API Endpoint |
|-----------------|-------------------------|--------------|
| Authentication | v2.2.0+ | POST /api/auth/login |
| Auto API Keys | v2.4.1 | POST /api/auth/devices/register |
| Device Management | v2.4.2 | GET/DELETE /api/auth/devices |
| WebSocket Streaming | v2.4.3 | /ws/chat |
| Device UI | v2.4.4 | Web UI updates |
| **Agentic Mode** | **v2.5.0+** | **/api/agent/\*** |
| Agent Task Planning | v2.5.0+ | POST /api/agent/plan |
| Agent Tool Execution | v2.5.0+ | POST /api/agent/tools/execute |
| MCP Catalog | v2.5.0+ | GET /api/mcp/catalog |
| MCP Tool Invoke | v2.5.0+ | POST /api/mcp/invoke |
| File Operations Tool | v2.5.0+ | POST /api/agent/tools/file |
| Terminal Tool | v2.5.0+ | POST /api/agent/tools/terminal |

### Technical Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Server unavailable | High | Local cache + offline mode |
| API Key revoked | Medium | Graceful re-login flow |
| WebSocket failures | Low | Fallback to HTTP |
| Cross-platform bugs | Medium | Extensive testing on all OS |
| **Agent executes dangerous commands** | **Critical** | **Approval flow + command whitelist** |
| **Agent infinite loop (too many tools)** | **High** | **Max tool limit (50), timeout per task** |
| **MCP server compromised** | **High** | **Whitelist only our catalog, rate limiting** |
| **File operations corrupt data** | **Medium** | **Rollback mechanism, git-like snapshots** |
| **LSP binary not installed** | **Low** | **Auto-detect, prompt user to install** |
| **Terminal command hangs** | **Medium** | **60s timeout, kill process** |

---

## 🔗 Links & Resources

### Related Documentation

- [Roadmap.MD](Roadmap.MD) - Server roadmap (v2.4.0 Desktop Support)
- [Architecture.MD](Architecture.MD) - System architecture
- [API.md](docs/API.md) - API documentation

### Wails Resources

- [Wails Documentation](https://wails.io/docs/introduction)
- [Go Bindings Guide](https://wails.io/docs/reference/runtime/intro)
- [Svelte Integration](https://wails.io/docs/guides/frontend)

### Similar Projects

- [Cursor](https://cursor.sh/) - AI code editor (inspiration)
- [VS Code](https://code.visualstudio.com/) - Reference UX
- [Windsurf](https://codeium.com/windsurf) - Competitor

---

## 🎉 Summary

**Desktop App = Thin Client** к существующему AIGateway Server

**Архитектура:**

- 🖥️ Desktop: UI + File System + Local Cache
- 🌐 Server: AI Logic + Database + Auth

**Authentication:**

- 🔐 One-time login → permanent API Key для device
- 📱 Multi-device support через Device Management

**Timeline:**

- MVP (v0.1.0): 2 weeks ✅
- File Integration (v0.2.0): 2 weeks ✅
- Monaco Editor (v0.3.0): 3 weeks ✅
- WebSocket Streaming (v0.4.0): 2 weeks ✅
- Hotfixes (v0.4.1): 3 days ⏳ NEXT
- Terminal & LSP (v0.5.0): 3 weeks
- Offline Mode (v0.6.0): 2 weeks
- Packaging (v0.7.0): 2 weeks
- Agentic Mode (v0.8.0): 4-5 weeks 🤖
- Production (v1.0.0): 6-8 months total

**Next Steps:**

1. ✅ ~~Implement server v2.4.1 (Auto API Keys)~~ - DONE
2. ✅ ~~Initialize Wails project~~ - DONE
3. ✅ ~~CLIENT-001 to CLIENT-016~~ - COMPLETED (v0.1.0 - v0.4.0)
4. ⏳ **CLIENT-030** - Fix models loading bug (v0.4.1) - **IN PROGRESS**
5. 🎯 **CLIENT-017** - Embedded Terminal (v0.5.0) - **NEXT MAJOR**
6. 🤖 **CLIENT-028 to CLIENT-033** - Agentic Mode (v0.8.0) - **FUTURE**

**Current Status (2025-10-29):**

- ✅ v0.1.0 - v0.4.0: COMPLETED
- ⏳ v0.4.1: IN PROGRESS (models loading bug fix)
- 📋 v0.5.0 - v0.8.0: PLANNED

**Major Milestones Ahead:**

1. **v0.5.0** (Terminal & LSP) - ~3 weeks
2. **v0.8.0** (Agentic Mode 🤖) - ~4-5 weeks (BIGGEST FEATURE)
3. **v1.0.0** (Production) - ~6-8 months total

Let's build the best AI desktop client with full Agent capabilities! 🚀🤖
