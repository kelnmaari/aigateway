# AIGateway Desktop - Development Roadmap

> **Platform:** Wails v2 (Go + WebView)  
> **Current Status:** Planning Phase 🎯  
> **Target:** Cross-platform desktop app (Windows, macOS, Linux)  
> **Vision:** Lightweight AI-powered IDE with integrated chat, RAG, and MCP support  
> **Last Updated:** 2025-10-28

---

## 🎯 Project Vision

Создать нативное desktop-приложение на базе **Wails v2**, которое объединит:

- 💬 **AI Chat** - интерфейс для общения с LLM моделями
- 📝 **Code Editor** - Monaco Editor для редактирования кода с AI-подсказками
- 📚 **RAG Integration** - работа с документами и knowledge base
- 🔌 **MCP Servers** - управление Model Context Protocol серверами
- 🗂️ **File Explorer** - навигация по проектам и файлам

**Ключевые преимущества**:

- ⚡ Быстрая работа (нативная Go интеграция)
- 🪶 Маленький размер (~10-20 MB vs Electron ~150 MB)
- 💾 Низкое потребление памяти (~50-100 MB vs Electron ~300 MB)
- 🔐 Прямой доступ к файловой системе
- 🚀 Без HTTP overhead (прямые Go calls из JavaScript)
- 📦 Один .exe файл со всем необходимым

---

## 🛠️ Technology Stack

### Core Framework

- **Wails v2** - Go + WebView2 (Windows) / WKWebView (macOS) / WebKitGTK (Linux)
- **Go 1.25** - Backend language
- **Svelte** - Frontend framework (легковесный, быстрый)

### UI Components

- **Monaco Editor** - Code editor (VS Code движок)
- **Existing Web Assets** - Переиспользование `web/` компонентов
- **Tailwind CSS** - Styling (или существующий style.css)
- **Marked.js** - Markdown rendering для chat

### Go Backend Integration

- **Direct Bindings** - JavaScript → Go без HTTP
- **Embedded Server** - AIGateway server как библиотека (не child process)
- **SQLite** - Embedded database (shared with main server)
- **Ollama Client** - Прямая интеграция с локальным Ollama

---

## 📅 Release Roadmap

### Version 0.1.0 - MVP Desktop Wrapper (2 недели) 🎯 PRIORITY 1

**Фокус:** Базовое Wails приложение с chat интерфейсом

**Цели:**

- ✅ Работающий Wails app с embedded WebView
- ✅ Базовый chat интерфейс (переиспользование `web/chat.html`)
- ✅ Прямые Go bindings для chat API
- ✅ Системный tray icon
- ✅ Горячие клавиши (Ctrl+Shift+A для открытия)

**Задачи:**

- [ ] **DESK-001** Инициализация Wails проекта → `aigateway-desktop/`
  - Создать структуру проекта
  - Настроить `wails.json` конфигурацию
  - Symlink к `internal/` для переиспользования кода
  
- [ ] **DESK-002** Embedded AIGateway Server
  - Импортировать `internal/server` как библиотеку
  - Инициализация DB, handlers, config при старте app
  - Graceful shutdown при закрытии приложения
  
- [ ] **DESK-003** Chat UI (Svelte component)
  - Портировать `web/chat.html` → `ChatPanel.svelte`
  - Прямые Go bindings: `SendMessage(message, model) -> response`
  - Streaming support через Wails events
  - Markdown rendering для AI ответов
  
- [ ] **DESK-004** System Integration
  - System tray icon с context menu (Show/Hide/Quit)
  - Global hotkey registration (Ctrl+Shift+A)
  - Window state persistence (размер, позиция)
  - Auto-start при входе в систему (опционально)
  
- [ ] **DESK-005** Models Management
  - Список доступных моделей (Ollama discovery)
  - Переключение между моделями в UI
  - Индикатор статуса Ollama (running/stopped)

**Acceptance Criteria:**

- ✅ Приложение запускается и показывает chat UI
- ✅ Можно отправить сообщение и получить ответ от LLM
- ✅ Работает system tray и hotkeys
- ✅ Размер .exe < 30 MB
- ✅ Потребление памяти < 150 MB

**Время:** ~10-14 дней

---

### Version 0.2.0 - File System Integration (2 недели) 🎯 PRIORITY 2

**Фокус:** Работа с локальными файлами и проектами

**Задачи:**

- [ ] **DESK-006** File Explorer Component
  - Tree view для навигации по папкам
  - Open folder dialog (Wails native)
  - Recent projects list
  - Context menu (Open, Delete, Rename, etc.)
  
- [ ] **DESK-007** File Viewer/Editor (basic)
  - Просмотр текстовых файлов
  - Syntax highlighting для популярных языков
  - Basic text editing (без Monaco пока)
  - Save/Save As functionality
  
- [ ] **DESK-008** Project Settings
  - Сохранение настроек проекта (.aigateway/config.json)
  - Default model для проекта
  - Exclude patterns для indexing
  - Git integration (detect .git folder)

**Acceptance Criteria:**

- ✅ Можно открыть папку и увидеть файлы
- ✅ Можно открыть файл на просмотр/редактирование
- ✅ Сохранение изменений в файлы
- ✅ Recent projects list

**Время:** ~10-14 дней

---

### Version 0.3.0 - RAG Integration (2-3 недели) 🎯 PRIORITY 3

**Фокус:** Индексация файлов проекта и контекстный поиск

**Задачи:**

- [ ] **DESK-009** RAG Sources Management
  - UI для управления RAG sources
  - Drag & drop файлов/папок для индексации
  - Progress bar при индексации
  - Список проиндексированных файлов с метками
  
- [ ] **DESK-010** Auto-indexing
  - Автоматическая индексация при открытии проекта
  - Watch mode для отслеживания изменений в файлах
  - Incremental indexing (только измененные файлы)
  - Background processing (не блокирует UI)
  
- [ ] **DESK-011** RAG Search Integration
  - Контекстный поиск в chat
  - Автоматическое добавление relevant files в context
  - UI индикатор использованных RAG файлов
  - Preview найденных фрагментов кода
  
- [ ] **DESK-012** Advanced RAG Features
  - Semantic search по содержимому файлов
  - Фильтрация по типам файлов
  - Exclude patterns (node_modules, .git, etc.)
  - Manual context selection (выбор файлов для chat)

**Acceptance Criteria:**

- ✅ Можно проиндексировать папку с проектом
- ✅ Chat использует RAG для ответов
- ✅ Видно какие файлы были использованы
- ✅ Автоматическая переиндексация при изменениях

**Время:** ~14-21 день

---

### Version 0.4.0 - Monaco Editor Integration (3 недели) 🎯 PRIORITY 4

**Фокус:** Полноценный code editor как в VS Code

**Задачи:**

- [ ] **DESK-013** Monaco Editor Setup
  - Интеграция Monaco Editor в Svelte
  - Multi-tab support (открытие нескольких файлов)
  - Split view (вертикальный/горизонтальный)
  - Theme support (dark/light/custom)
  
- [ ] **DESK-014** Language Support
  - Syntax highlighting для 50+ языков
  - IntelliSense (автодополнение) из Monaco
  - Code folding
  - Minimap
  - Line numbers, breadcrumbs
  
- [ ] **DESK-015** Editor Features
  - Find & Replace (в файле и во всех файлах)
  - Go to definition (если есть LSP)
  - Multi-cursor editing
  - Code formatting (prettier/gofmt)
  - Diff viewer для git changes
  
- [ ] **DESK-016** Chat + Editor Integration
  - Выделение кода → отправка в chat (context menu)
  - AI suggestions inline (hover tooltips)
  - Apply AI changes to code (diff view)
  - Code explanations при наведении

**Acceptance Criteria:**

- ✅ Полноценный code editor с Monaco
- ✅ Можно редактировать несколько файлов одновременно
- ✅ Интеграция с chat (send code snippet)
- ✅ Базовые editor shortcuts работают

**Время:** ~21 день

---

### Version 0.5.0 - AI Code Features (3-4 недели) 🎯 PRIORITY 5

**Фокус:** Cursor-like функционал для кода

**Задачи:**

- [ ] **DESK-017** Inline AI Suggestions
  - Streaming suggestions при наборе кода
  - Ghost text для предложений (Tab to accept)
  - Context из открытых файлов
  - Debouncing для оптимизации
  
- [ ] **DESK-018** Code Actions
  - "Explain this code" action
  - "Fix this code" action  
  - "Add comments" action
  - "Write tests" action
  - "Refactor" suggestions
  
- [ ] **DESK-019** Multi-line Edits
  - AI генерирует изменения в нескольких местах
  - Diff view для preview changes
  - Accept/Reject отдельных изменений
  - Undo/Redo stack
  
- [ ] **DESK-020** Chat-driven Development
  - "Apply this to MyFile.go" команда в chat
  - AI генерирует полные функции/классы
  - Automatic imports добавление
  - Test generation из chat

**Acceptance Criteria:**

- ✅ Inline suggestions работают
- ✅ Code actions доступны через context menu
- ✅ Можно применить AI изменения к коду
- ✅ Chat может редактировать файлы

**Время:** ~21-28 дней

---

### Version 0.6.0 - MCP Integration (2-3 недели) 🎯 PRIORITY 6

**Фокус:** Model Context Protocol servers management

**Задачи:**

- [ ] **DESK-021** MCP Servers Discovery
  - Автоматическое обнаружение локальных MCP servers
  - Configuration UI для добавления servers
  - Connection status indicator
  - Logs viewer для debugging
  
- [ ] **DESK-022** MCP Operations
  - Listing available tools/resources
  - Calling MCP tools из chat
  - Resource preview (files, URLs, etc.)
  - Error handling и retry logic
  
- [ ] **DESK-023** Built-in MCP Servers
  - File System MCP (browse local files)
  - Git MCP (git operations)
  - Web Search MCP (опционально)
  - Memory MCP (persistent context)
  
- [ ] **DESK-024** MCP Configuration
  - Add/Remove/Edit MCP servers
  - Enable/Disable servers
  - Custom env variables
  - Stdio/HTTP transport support

**Acceptance Criteria:**

- ✅ Можно добавить и подключить MCP server
- ✅ Chat может использовать MCP tools
- ✅ Видны логи и ошибки MCP
- ✅ Built-in File System MCP работает

**Время:** ~14-21 день

---

### Version 0.7.0 - Terminal Integration (1-2 недели) 🎯 PRIORITY 7

**Фокус:** Встроенный терминал как в VS Code

**Задачи:**

- [ ] **DESK-025** Terminal Component (xterm.js)
  - Embedded terminal в нижней панели
  - Multiple terminal tabs
  - Split terminal view
  - Shell selection (bash/zsh/powershell/cmd)
  
- [ ] **DESK-026** Terminal Features
  - Copy/Paste support
  - Find in terminal
  - Clear terminal
  - Persistent history
  - Working directory sync с opened folder

**Acceptance Criteria:**

- ✅ Работающий terminal внизу UI
- ✅ Можно выполнять команды
- ✅ Multiple terminals support

**Время:** ~7-14 дней

---

### Version 0.8.0 - Git Integration (2 недели) 🎯 PRIORITY 8

**Фокус:** Source control интеграция

**Задачи:**

- [ ] **DESK-027** Git Status
  - Source Control panel (левый sidebar)
  - Changed files list
  - Diff viewer для changes
  - Stage/Unstage files
  
- [ ] **DESK-028** Git Operations
  - Commit changes
  - Push/Pull
  - Branch management (create/switch/delete)
  - Merge conflicts resolver
  
- [ ] **DESK-029** Git History
  - Commit history viewer
  - Blame annotations в editor
  - File history

**Acceptance Criteria:**

- ✅ Видны git changes
- ✅ Можно commit и push
- ✅ Branch switching работает

**Время:** ~14 дней

---

### Version 0.9.0 - Polish & Performance (2 недели) 🎯 PRIORITY 9

**Фокус:** Оптимизация и улучшение UX

**Задачи:**

- [ ] **DESK-030** Performance Optimization
  - Lazy loading для Monaco Editor
  - Virtual scrolling для file tree
  - Debouncing для search/indexing
  - Memory profiling и оптимизация
  
- [ ] **DESK-031** UI/UX Improvements
  - Keyboard shortcuts panel
  - Command palette (Ctrl+Shift+P)
  - Settings UI (preferences)
  - Themes customization
  
- [ ] **DESK-032** Error Handling
  - Graceful error messages
  - Crash reporting (опционально)
  - Logs export
  - Debug mode

**Acceptance Criteria:**

- ✅ UI отзывчивый и быстрый
- ✅ Понятные error messages
- ✅ Settings UI работает

**Время:** ~14 дней

---

### Version 1.0.0 - Production Release (2 недели) 🎯 MILESTONE

**Фокус:** Готовность к production использованию

**Задачи:**

- [ ] **DESK-033** Packaging & Distribution
  - Windows .exe с installer (NSIS/WiX)
  - macOS .app с DMG
  - Linux AppImage/deb/rpm
  - Code signing для Windows/macOS
  
- [ ] **DESK-034** Auto-updates
  - Update checker при старте
  - Download & install updates
  - Release notes display
  - Rollback mechanism
  
- [ ] **DESK-035** Documentation
  - User guide (README.md)
  - Keyboard shortcuts reference
  - Settings documentation
  - FAQ и troubleshooting
  
- [ ] **DESK-036** Testing
  - E2E tests (playwright/tauri-driver)
  - Unit tests для Go bindings
  - Performance benchmarks
  - Cross-platform testing

**Acceptance Criteria:**

- ✅ Installable packages для всех платформ
- ✅ Auto-update работает
- ✅ Полная документация
- ✅ Test coverage > 70%

**Время:** ~14 дней

---

## 🚀 Post 1.0 Features (Future Roadmap)

### Version 1.1.0+ - Advanced Features

**Потенциальные фичи:**

- [ ] **Multi-workspace** support (несколько проектов одновременно)
- [ ] **Remote SSH** editing (как VS Code Remote)
- [ ] **Collaboration** (real-time code sharing)
- [ ] **Local AI Models** (встроенный Ollama без внешнего сервера)
- [ ] **Plugin System** (пользовательские расширения)
- [ ] **Jupyter Notebooks** support
- [ ] **Database Explorer** (SQL queries, schema viewer)
- [ ] **Docker Integration** (container management)
- [ ] **Cloud Sync** (настройки между устройствами)
- [ ] **Mobile Companion App** (просмотр проектов на телефоне)

---

## 📐 Architecture Overview

### Project Structure

```
aigateway-desktop/
├── main.go                    # Wails entry point
├── app.go                     # Go bindings для frontend
├── wails.json                 # Wails configuration
├── build/                     # Build artifacts
│   ├── bin/
│   │   └── aigateway-desktop.exe
│   └── windows/              # Platform-specific resources
│       └── icon.ico
├── frontend/                  # Svelte app
│   ├── src/
│   │   ├── App.svelte
│   │   ├── lib/
│   │   │   ├── components/
│   │   │   │   ├── Chat.svelte
│   │   │   │   ├── Editor.svelte
│   │   │   │   ├── FileExplorer.svelte
│   │   │   │   ├── RAGPanel.svelte
│   │   │   │   ├── MCPPanel.svelte
│   │   │   │   └── Terminal.svelte
│   │   │   └── stores/
│   │   │       ├── chat.js
│   │   │       ├── files.js
│   │   │       └── settings.js
│   │   └── assets/
│   │       ├── styles/          # Копия web/css/
│   │       └── images/
│   ├── wailsjs/                # Auto-generated Go bindings
│   └── package.json
├── internal/                   # Symlink → ../internal (shared)
│   ├── models/
│   ├── storage/
│   ├── api/
│   └── services/
└── configs/                    # App configurations
    └── default.yaml
```

### Go ↔ Frontend Communication

```go
// app.go - Go bindings
type App struct {
    ctx      context.Context
    server   *server.Server
    db       storage.Database
    ollama   *ollama.Client
}

// Exposed to JavaScript
func (a *App) SendChatMessage(message, model string) (*models.ChatResponse, error)
func (a *App) GetModels() ([]models.Model, error)
func (a *App) IndexFile(path string) error
func (a *App) SearchRAG(query string) ([]models.RAGResult, error)
func (a *App) GetMCPServers() ([]models.MCPServer, error)
```

```javascript
// Frontend (Svelte)
import { SendChatMessage, GetModels } from '../wailsjs/go/main/App';

async function sendMessage() {
    const response = await SendChatMessage(message, selectedModel);
    messages = [...messages, response];
}
```

### Event System (для streaming)

```go
// Go → Frontend events
runtime.EventsEmit(a.ctx, "chat:stream", chunk)
runtime.EventsEmit(a.ctx, "rag:progress", progress)
```

```javascript
// Frontend listens
import { EventsOn } from '../wailsjs/runtime/runtime';

EventsOn('chat:stream', (chunk) => {
    currentResponse += chunk;
});
```

---

## 🎯 Success Metrics

### Version 0.1.0 (MVP)

- ✅ App размер < 30 MB
- ✅ Memory usage < 150 MB (idle)
- ✅ Startup time < 3 seconds
- ✅ First message response < 1 second (after LLM processing)

### Version 1.0.0 (Production)

- ✅ App размер < 50 MB
- ✅ Memory usage < 200 MB (with Monaco + file indexing)
- ✅ Startup time < 5 seconds
- ✅ 100+ concurrent files open without lag
- ✅ RAG indexing 1000 files in < 30 seconds
- ✅ Zero crashes per week average

---

## ⚠️ Risks & Mitigation

### Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| WebView различия между платформами | High | Medium | Тестирование на всех OS, fallback UI |
| Monaco Editor performance в WebView | Medium | Low | Lazy loading, virtual scrolling |
| Wails breaking changes | Medium | Low | Pin версии, follow updates |
| Go bindings overhead | Low | Low | Benchmarking, optimization |
| Large file indexing блокирует UI | High | Medium | Background workers, progress indicators |

### Product Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Users prefer web version | High | Medium | Unique desktop features (hotkeys, file access) |
| Ollama не установлен | High | High | Bundled Ollama или clear setup guide |
| Сложная установка | Medium | Low | Single .exe installer |
| Competition (Cursor, Windsurf) | High | High | Focus on open-source, local-first, privacy |

---

## 📚 Resources & Links

### Wails Documentation

- [Wails v2 Docs](https://wails.io/docs/introduction)
- [Go Bindings Guide](https://wails.io/docs/reference/runtime/intro)
- [Frontend Integration](https://wails.io/docs/guides/application-development)

### Similar Projects

- [Cursor](https://cursor.sh/) - AI-first code editor (closed-source)
- [Windsurf](https://codeium.com/windsurf) - Codeium IDE
- [VS Code](https://github.com/microsoft/vscode) - Reference implementation

### Libraries to Use

- [Monaco Editor](https://microsoft.github.io/monaco-editor/)
- [xterm.js](https://xtermjs.org/) - Terminal emulator
- [simple-git](https://github.com/steveukx/git-js) - Git integration
- [chokidar](https://github.com/paulmillr/chokidar) - File watching

---

## 🎉 Conclusion

**Estimated Timeline:**

- MVP (v0.1.0): 2 weeks
- Beta (v0.5.0): 3-4 months
- Production (v1.0.0): 5-6 months

**Team Requirements:**

- 1x Go developer (backend integration)
- 1x Frontend developer (Svelte + Monaco)
- 0.5x UI/UX designer (part-time)

**Next Steps:**

1. Initialize Wails project: `wails init -n aigateway-desktop -t svelte`
2. Setup project structure and symlinks
3. Start with **DESK-001** (MVP Chat UI)
4. Weekly releases for rapid iteration

**Let's build the future of AI-powered development tools! 🚀**
