# Переключение версий WebUI (Legacy / Svelte)

## Требование

Возможность переключения между старым (Legacy HTML+JS) и новым (Svelte) интерфейсом через флаг при запуске сервера.

---

## Архитектура решения

### Структура каталогов

```
project/
├── web/                    # Legacy UI (HTML + JS)
│   ├── index.html
│   ├── login.html
│   ├── css/
│   └── js/
│
├── web-svelte/             # Svelte source
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
│
└── web-svelte-build/       # Svelte production build
    ├── index.html
    ├── _app/
    └── assets/
```

### Конфигурация

```yaml
# configs/dev.yaml или production.yaml
server:
  host: "0.0.0.0"
  port: 8080

webui:
  enabled: true
  version: "legacy"    # "legacy" | "svelte"
```

### CLI флаг

```bash
# Использование legacy UI (по умолчанию)
./server --webui-version=legacy

# Использование нового Svelte UI
./server --webui-version=svelte
```

---

## Go реализация

### internal/config/config.go

```go
type WebUIConfig struct {
    Enabled bool   `yaml:"enabled" default:"true"`
    Version string `yaml:"version" default:"legacy"` // "legacy" | "svelte"
}

type Config struct {
    // ... existing fields
    WebUI WebUIConfig `yaml:"webui"`
}

func (c *Config) Validate() error {
    if c.WebUI.Version != "legacy" && c.WebUI.Version != "svelte" {
        return fmt.Errorf("invalid webui.version: %s (expected 'legacy' or 'svelte')", c.WebUI.Version)
    }
    return nil
}
```

### internal/web/embed.go

```go
package web

import (
    "embed"
    "io/fs"
)

//go:embed all:static
var legacyFiles embed.FS

//go:embed all:svelte-build
var svelteFiles embed.FS

// GetStaticFS returns the appropriate static files based on version
func GetStaticFS(version string) (fs.FS, error) {
    switch version {
    case "svelte":
        return fs.Sub(svelteFiles, "svelte-build")
    case "legacy":
        fallthrough
    default:
        return fs.Sub(legacyFiles, "static")
    }
}
```

### internal/web/handler.go

```go
package web

import (
    "io/fs"
    "net/http"
    "strings"
)

type Handler struct {
    fileServer http.Handler
    indexHTML  []byte
    version    string
}

func NewHandler(version string) (*Handler, error) {
    staticFS, err := GetStaticFS(version)
    if err != nil {
        return nil, fmt.Errorf("failed to get static FS: %w", err)
    }
    
    // Read index.html for SPA fallback
    indexHTML, err := fs.ReadFile(staticFS, "index.html")
    if err != nil {
        return nil, fmt.Errorf("failed to read index.html: %w", err)
    }
    
    return &Handler{
        fileServer: http.FileServer(http.FS(staticFS)),
        indexHTML:  indexHTML,
        version:    version,
    }, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    path := r.URL.Path
    
    // Skip API routes
    if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/v1/") {
        http.NotFound(w, r)
        return
    }
    
    // Try to serve static file
    // For SPA (Svelte), fall back to index.html for client-side routing
    if h.version == "svelte" && !h.hasExtension(path) {
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Write(h.indexHTML)
        return
    }
    
    h.fileServer.ServeHTTP(w, r)
}

func (h *Handler) hasExtension(path string) bool {
    // Check if path has a file extension
    for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
        if path[i] == '.' {
            return true
        }
    }
    return false
}
```

### cmd/server/main.go

```go
func main() {
    // Parse flags
    webuiVersion := flag.String("webui-version", "", "WebUI version: legacy or svelte")
    flag.Parse()
    
    // Load config
    cfg, err := config.Load(configPath)
    if err != nil {
        log.Fatal(err)
    }
    
    // Override from CLI flag if provided
    if *webuiVersion != "" {
        cfg.WebUI.Version = *webuiVersion
    }
    
    // Validate
    if err := cfg.Validate(); err != nil {
        log.Fatal(err)
    }
    
    // Create web handler
    webHandler, err := web.NewHandler(cfg.WebUI.Version)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("WebUI version: %s", cfg.WebUI.Version)
    
    // Setup routes...
}
```

---

## Build Pipeline

### Makefile

```makefile
# Build Svelte UI
.PHONY: build-svelte
build-svelte:
	cd web-svelte && npm ci && npm run build
	rm -rf internal/web/svelte-build
	cp -r web-svelte/build internal/web/svelte-build

# Build with legacy UI only
.PHONY: build-legacy
build-legacy:
	go build -o bin/server cmd/server/main.go

# Build with Svelte UI
.PHONY: build-with-svelte
build-with-svelte: build-svelte
	go build -o bin/server cmd/server/main.go

# Full build (both UIs embedded)
.PHONY: build
build: build-svelte
	go build -o bin/server cmd/server/main.go
```

### GitHub Actions

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: 'web-svelte/package-lock.json'
      
      - name: Build Svelte UI
        run: |
          cd web-svelte
          npm ci
          npm run build
          cp -r build ../internal/web/svelte-build
      
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      
      - name: Build Server
        run: go build -o bin/server cmd/server/main.go
      
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: server
          path: bin/server
```

---

## Структура internal/web/

```
internal/web/
├── embed.go           # embed.FS declarations
├── handler.go         # HTTP handler for static files
├── static/            # Legacy UI files (copied from web/)
│   ├── index.html
│   ├── login.html
│   ├── css/
│   └── js/
└── svelte-build/      # Svelte build output (copied from web-svelte/build/)
    ├── index.html
    ├── _app/
    └── assets/
```

---

## Процесс миграции

### Этап 1: Подготовка (текущий)
```yaml
webui:
  version: "legacy"  # Default
```
- Svelte UI в разработке
- Пользователи используют Legacy

### Этап 2: Beta Testing
```yaml
webui:
  version: "legacy"  # Default, но svelte доступен
```
- Svelte UI доступен через `--webui-version=svelte`
- Сбор feedback от beta-testers

### Этап 3: Transition
```yaml
webui:
  version: "svelte"  # Default меняется
```
- Svelte UI по умолчанию
- Legacy доступен через `--webui-version=legacy`
- Уведомление о deprecation Legacy

### Этап 4: Cleanup
```yaml
webui:
  version: "svelte"  # Only option
```
- Удаление Legacy UI
- Удаление флага переключения

---

## Альтернативный подход: Build Tags

Если нужно минимизировать размер бинарника:

### internal/web/embed_legacy.go

```go
//go:build !svelte

package web

import "embed"

//go:embed all:static
var staticFiles embed.FS

const Version = "legacy"
```

### internal/web/embed_svelte.go

```go
//go:build svelte

package web

import "embed"

//go:embed all:svelte-build
var staticFiles embed.FS

const Version = "svelte"
```

### Build команды

```bash
# Legacy build (default)
go build -o bin/server-legacy cmd/server/main.go

# Svelte build
go build -tags svelte -o bin/server-svelte cmd/server/main.go
```

**Минус:** Нужно собирать два бинарника. Рекомендуется первый подход (runtime switch).

---

## Тестирование

### Проверка версии через API

```go
// GET /api/system/info
type SystemInfo struct {
    Version      string `json:"version"`
    WebUIVersion string `json:"webui_version"` // "legacy" или "svelte"
    // ...
}
```

### Автоматические тесты

```go
func TestWebHandler_Legacy(t *testing.T) {
    h, err := web.NewHandler("legacy")
    require.NoError(t, err)
    
    req := httptest.NewRequest("GET", "/login.html", nil)
    rec := httptest.NewRecorder()
    
    h.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
}

func TestWebHandler_Svelte(t *testing.T) {
    h, err := web.NewHandler("svelte")
    require.NoError(t, err)
    
    // SPA fallback test
    req := httptest.NewRequest("GET", "/dashboard", nil)
    rec := httptest.NewRecorder()
    
    h.ServeHTTP(rec, req)
    
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Contains(t, rec.Body.String(), "<!DOCTYPE html>")
}
```

---

## Чек-лист

- [ ] Создать internal/web/embed.go
- [ ] Создать internal/web/handler.go
- [ ] Добавить WebUIConfig в config
- [ ] Добавить CLI флаг --webui-version
- [ ] Скопировать web/ в internal/web/static/
- [ ] Настроить build pipeline для Svelte
- [ ] Добавить webui_version в /api/system/info
- [ ] Написать тесты
- [ ] Обновить документацию (README)

