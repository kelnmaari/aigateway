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

- [ ] **CLIENT-006** File Picker
  - Wails native file dialog (OpenFileDialog)
  - Drag & drop files в chat window
  - File info display (name, size, type)
  
- [ ] **CLIENT-007** File Upload to RAG
  - Чтение файла локально
  - POST /api/rag/index с file content
  - Progress indicator для больших файлов
  - Batch upload (multiple files)
  
- [ ] **CLIENT-008** Project Management
  - Open folder dialog
  - Recent projects list (stored locally)
  - Auto-indexing prompt для новых проектов
  
- [ ] **CLIENT-009** Basic File Viewer
  - Display text files в read-only mode
  - Syntax highlighting для code files
  - File tree navigation (left sidebar)

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

- [ ] **CLIENT-010** Monaco Editor Setup
  - Интеграция Monaco Editor в Svelte
  - Multi-tab support (открытие нескольких файлов)
  - Theme support (dark/light)
  - Language detection по расширению
  
- [ ] **CLIENT-011** Editor Features
  - Syntax highlighting для 50+ языков
  - Code completion (IntelliSense из Monaco)
  - Find & Replace
  - Code folding, minimap
  
- [ ] **CLIENT-012** File Editing
  - Edit local files
  - Save/Save As functionality
  - Unsaved changes indicator
  - Auto-save (опционально)
  
- [ ] **CLIENT-013** Chat Integration
  - Send code snippet to chat (выделение → context menu)
  - "Explain this code" action
  - Apply AI suggestions к файлу (diff view)

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

- [ ] **CLIENT-014** WebSocket Client
  - Подключение к /ws/chat?token={api_key}
  - Auto-reconnect logic
  - Ping/Pong для keep-alive
  
- [ ] **CLIENT-015** Streaming UI
  - Отображение partial responses в real-time
  - Typing indicator
  - Stop generation button
  
- [ ] **CLIENT-016** Fallback to HTTP
  - Detect если WebSocket unavailable
  - Fallback to POST /api/chat/completions
  - Graceful degradation

**Acceptance Criteria:**

- ✅ WebSocket streaming работает
- ✅ Auto-reconnect на disconnect
- ✅ Fallback to HTTP если WS fail

**Время:** ~14 дней

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
  
- [ ] **CLIENT-019** Settings UI
  - Server configuration
  - Theme selection
  - Keyboard shortcuts
  - Auto-start preferences
  
- [ ] **CLIENT-020** Device Management
  - Show all user devices (GET /api/auth/devices)
  - Self-revoke button
  - Device info display

**Acceptance Criteria:**

- ✅ Terminal работает
- ✅ Git status видно
- ✅ Settings UI функционален

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

### Version 1.0.0 - Production Release 🎯 MILESTONE

**Total Development Time:** ~5-6 месяцев

**Requirements:**

- ✅ All v0.1.0-v0.7.0 features implemented
- ✅ Test coverage > 70%
- ✅ Performance benchmarks pass
- ✅ Security audit complete
- ✅ Documentation complete
- ✅ Cross-platform testing done

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

### Technical Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Server unavailable | High | Local cache + offline mode |
| API Key revoked | Medium | Graceful re-login flow |
| WebSocket failures | Low | Fallback to HTTP |
| Cross-platform bugs | Medium | Extensive testing on all OS |

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

- MVP (v0.1.0): 2 weeks
- Beta (v0.4.0): 2-3 months  
- Production (v1.0.0): 5-6 months

**Next Steps:**

1. Implement server v2.4.1 (Auto API Keys) - **PREREQUISITE**
2. Initialize Wails project: `wails init -n aigateway-desktop`
3. Start with CLIENT-001 (Project Setup)
4. Weekly releases для rapid iteration

Let's build the best AI desktop client! 🚀
