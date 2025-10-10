# TODO-02: Build Version Information

**Версия:** v1.4.3  
**Приоритет:** MEDIUM  
**Оценка:** 1-2 часа  
**Зависимости:** Нет  
**Статус:** 📋 Planned  
**Тип:** Technical Debt / DevOps

---

## 📋 Описание

Заменить hardcoded версию "dev" на реальную версию из build процесса. Использовать Go ldflags для инъекции версии, commit hash и build time при компиляции.

## 🎯 Цели

1. Показывать реальную версию приложения
2. Упростить debugging (версия, commit, build time)
3. Tracking версий в production
4. Подготовка к автоматизированным релизам

## 📊 Current Issues

### Найденные TODO комментарии

**Файл:** `internal/logger/logger.go` (строка 66)

```go
"version": "dev", // TODO: Получать из build информации
```

**Файл:** `internal/api/handlers/health.go` (строка 36, 38)

```go
"version":   "dev", // TODO: Получать из build информации
"uptime":    time.Since(time.Now().Add(-5 * time.Minute)).String(), // TODO: Реальный uptime
```

### Проблемы

1. Невозможно определить версию в production
2. Нет информации о commit для debugging
3. Fake uptime вместо реального
4. Отсутствие build metadata

## 🔧 Solution Design

### 1. Version Package

**Создать:** `internal/version/version.go`

```go
package version

import (
    "fmt"
    "runtime"
    "time"
)

var (
    // Эти переменные будут установлены через ldflags при сборке
    Version   = "dev"
    GitCommit = "unknown"
    BuildDate = "unknown"
    GoVersion = runtime.Version()
)

// Info содержит информацию о версии
type Info struct {
    Version   string `json:"version"`
    GitCommit string `json:"git_commit"`
    BuildDate string `json:"build_date"`
    GoVersion string `json:"go_version"`
}

// GetInfo возвращает информацию о версии
func GetInfo() Info {
    return Info{
        Version:   Version,
        GitCommit: GitCommit,
        BuildDate: BuildDate,
        GoVersion: GoVersion,
    }
}

// String возвращает версию в читаемом формате
func (i Info) String() string {
    return fmt.Sprintf(
        "Version: %s, Commit: %s, Built: %s, Go: %s",
        i.Version,
        i.GitCommit,
        i.BuildDate,
        i.GoVersion,
    )
}

// Short возвращает короткую версию
func Short() string {
    if GitCommit != "unknown" && len(GitCommit) > 7 {
        return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
    }
    return Version
}
```

### 2. Build Scripts Update

#### Makefile Update

**Файл:** `Makefile`

```makefile
# Version information
VERSION := $(shell cat VERSION)
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')

# Ldflags для версионирования
LDFLAGS := -X 'github.com/yourusername/ollama-openai-proxy/internal/version.Version=$(VERSION)' \
           -X 'github.com/yourusername/ollama-openai-proxy/internal/version.GitCommit=$(GIT_COMMIT)' \
           -X 'github.com/yourusername/ollama-openai-proxy/internal/version.BuildDate=$(BUILD_DATE)'

# Build команды
.PHONY: build
build:
 @echo "Building with version $(VERSION), commit $(GIT_COMMIT)"
 go build -ldflags="$(LDFLAGS)" -o bin/server cmd/server/main.go
 go build -ldflags="$(LDFLAGS)" -o bin/tui cmd/tui/main.go
 go build -ldflags="$(LDFLAGS)" -o bin/webui cmd/webui/main.go

.PHONY: build-all
build-all:
 @echo "Building all binaries..."
 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/ollama-proxy-linux-amd64 cmd/server/main.go
 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/ollama-proxy-linux-arm64 cmd/server/main.go
 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/ollama-proxy-windows-amd64.exe cmd/server/main.go

.PHONY: version
version:
 @echo "Version: $(VERSION)"
 @echo "Git Commit: $(GIT_COMMIT)"
 @echo "Build Date: $(BUILD_DATE)"
 @echo "Go Version: $(GO_VERSION)"
```

#### build.sh Update

**Файл:** `build.sh`

```bash
#!/bin/bash

# Version information
VERSION=$(cat VERSION)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Ldflags
LDFLAGS="-X 'github.com/yourusername/ollama-openai-proxy/internal/version.Version=${VERSION}' \
         -X 'github.com/yourusername/ollama-openai-proxy/internal/version.GitCommit=${GIT_COMMIT}' \
         -X 'github.com/yourusername/ollama-openai-proxy/internal/version.BuildDate=${BUILD_DATE}'"

echo "Building Ollama-OpenAI Proxy..."
echo "Version: ${VERSION}"
echo "Commit: ${GIT_COMMIT}"
echo "Build Date: ${BUILD_DATE}"

# Build server
echo "Building server..."
go build -ldflags="${LDFLAGS}" -o bin/server cmd/server/main.go

# Build TUI
echo "Building TUI..."
go build -ldflags="${LDFLAGS}" -o bin/tui cmd/tui/main.go

# Build WebUI
echo "Building WebUI..."
go build -ldflags="${LDFLAGS}" -o bin/webui cmd/webui/main.go

echo "Build complete!"
```

#### build.ps1 Update (Windows)

**Файл:** `build.ps1`

```powershell
# Version information
$VERSION = Get-Content VERSION
$GIT_COMMIT = (git rev-parse --short HEAD 2>$null) ?? "unknown"
$BUILD_DATE = (Get-Date).ToUniversalTime().ToString("yyyy-MM-dd_HH:mm:ss")

# Ldflags
$LDFLAGS = "-X 'github.com/yourusername/ollama-openai-proxy/internal/version.Version=$VERSION' " +
           "-X 'github.com/yourusername/ollama-openai-proxy/internal/version.GitCommit=$GIT_COMMIT' " +
           "-X 'github.com/yourusername/ollama-openai-proxy/internal/version.BuildDate=$BUILD_DATE'"

Write-Host "Building Ollama-OpenAI Proxy..." -ForegroundColor Green
Write-Host "Version: $VERSION"
Write-Host "Commit: $GIT_COMMIT"
Write-Host "Build Date: $BUILD_DATE"

# Build
Write-Host "`nBuilding server..." -ForegroundColor Cyan
go build -ldflags=$LDFLAGS -o bin/server.exe cmd/server/main.go

Write-Host "Building TUI..." -ForegroundColor Cyan
go build -ldflags=$LDFLAGS -o bin/tui.exe cmd/tui/main.go

Write-Host "Building WebUI..." -ForegroundColor Cyan
go build -ldflags=$LDFLAGS -o bin/webui.exe cmd/webui/main.go

Write-Host "`nBuild complete!" -ForegroundColor Green
```

### 3. Code Updates

#### Logger Update

**Файл:** `internal/logger/logger.go`

```go
import (
    "github.com/sirupsen/logrus"
    "your-project/internal/version"
)

func NewLogger(config LogConfig) *Logger {
    log := logrus.New()
    
    // ... existing setup
    
    // Добавить version в structured fields
    log.WithFields(logrus.Fields{
        "version":    version.Version,
        "git_commit": version.GitCommit,
        "build_date": version.BuildDate,
        "go_version": version.GoVersion,
    }).Info("Logger initialized")
    
    return &Logger{Logger: log}
}
```

#### Health Handler Update

**Файл:** `internal/api/handlers/health.go`

```go
import (
    "time"
    "github.com/gin-gonic/gin"
    "your-project/internal/version"
)

var (
    startTime = time.Now() // Глобальная переменная для uptime
)

func init() {
    startTime = time.Now()
}

func HealthCheck(c *gin.Context) {
    uptime := time.Since(startTime)
    versionInfo := version.GetInfo()
    
    c.JSON(http.StatusOK, gin.H{
        "status":      "healthy",
        "version":     versionInfo.Version,
        "git_commit":  versionInfo.GitCommit,
        "build_date":  versionInfo.BuildDate,
        "go_version":  versionInfo.GoVersion,
        "uptime":      uptime.String(),
        "uptime_seconds": int(uptime.Seconds()),
    })
}
```

#### Version Command (Optional)

**Файл:** `cmd/server/main.go`

```go
import (
    "flag"
    "fmt"
    "os"
    "your-project/internal/version"
)

var (
    showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
    flag.Parse()
    
    if *showVersion {
        info := version.GetInfo()
        fmt.Println(info.String())
        os.Exit(0)
    }
    
    // ... rest of main
}
```

### 4. VERSION File

**Файл:** `VERSION`

```
1.4.3
```

Этот файл будет источником версии для build скриптов.

## 📝 Implementation Plan

### Phase 1: Version Package (20 мин)

1. Создать `internal/version/version.go`
2. Определить variables для ldflags
3. Создать helper functions

### Phase 2: Build Scripts (30 мин)

1. Обновить `Makefile`
2. Обновить `build.sh`
3. Обновить `build.ps1`
4. Создать/обновить `VERSION` файл

### Phase 3: Code Integration (20 мин)

1. Обновить `internal/logger/logger.go`
2. Обновить `internal/api/handlers/health.go`
3. Добавить `-version` flag в cmd/server

### Phase 4: Testing (20 мин)

1. Test local builds
2. Test cross-compilation
3. Verify version in API responses
4. Verify version in logs

## ✅ Acceptance Criteria

- [ ] Version package создан
- [ ] Build scripts используют ldflags
- [ ] Реальная версия в `/health` endpoint
- [ ] Реальная версия в логах
- [ ] Реальный uptime в health check
- [ ] `-version` flag работает
- [ ] Cross-compilation с версией работает
- [ ] VERSION файл существует
- [ ] TODO комментарии удалены

## 🧪 Testing

### Build Testing

```bash
# Local build
make build
./bin/server -version

# Expected output:
# Version: 1.4.3, Commit: abc1234, Built: 2025-10-10_12:34:56, Go: go1.21.0

# Cross-compilation
make build-all
./dist/ollama-proxy-linux-amd64 -version
```

### API Testing

```bash
# Start server
./bin/server -config configs/dev.yaml

# Check health endpoint
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "version": "1.4.3",
  "git_commit": "abc1234",
  "build_date": "2025-10-10_12:34:56",
  "go_version": "go1.21.0",
  "uptime": "5m30s",
  "uptime_seconds": 330
}
```

### Log Testing

```bash
# Check logs for version
./bin/server -config configs/dev.yaml 2>&1 | grep version

# Expected:
# INFO[0000] Logger initialized  version=1.4.3 git_commit=abc1234 ...
```

## 📚 References

- [Go ldflags documentation](https://pkg.go.dev/cmd/link)
- [Semantic Versioning](https://semver.org/)
- [12-Factor App: Build, release, run](https://12factor.net/build-release-run)

## 🔄 Follow-up Tasks

- [ ] Автоматическое обновление VERSION при релизе
- [ ] Integration с CI/CD для автоматической версии
- [ ] Version compatibility checking
- [ ] Changelog generation на основе версии
- [ ] API версионирование (/v1, /v2, etc.)

## 🎯 Release Process

После реализации этой задачи:

1. Обновить `VERSION` файл на `1.4.3`
2. Commit изменений
3. Create git tag: `git tag v1.4.3`
4. Build: `make build-all`
5. Verify versions: `./dist/ollama-proxy-* -version`
6. Push tag: `git push origin v1.4.3`

## 📦 Distribution

**Binary naming convention:**

```
ollama-proxy-{os}-{arch}-v{version}

Примеры:
- ollama-proxy-linux-amd64-v1.4.3
- ollama-proxy-windows-amd64-v1.4.3.exe
- ollama-proxy-linux-arm64-v1.4.3
```
