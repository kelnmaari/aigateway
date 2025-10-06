# 🔨 Build Instructions

**Ollama-OpenAI Proxy v1.3.0**

Инструкции по сборке бинарных файлов для Linux и Windows.

---

## 📋 Требования

- **Go 1.25+** ([скачать](https://go.dev/dl/))
- **Git** (опционально, для metadata)
- **Make** (опционально, для Makefile)

---

## 🚀 Quick Start

### Linux / macOS

```bash
# Сборка для всех платформ
./build.sh all

# Или с помощью Make
make build-all

# Результат в dist/
ls dist/
```

### Windows (PowerShell)

```powershell
# Сборка для всех платформ
.\build.ps1 all

# Результат в dist\
dir dist\
```

---

## 📦 Доступные команды

### Bash скрипт (build.sh)

```bash
./build.sh all       # Собрать для всех платформ
./build.sh linux     # Только Linux (amd64 + arm64)
./build.sh windows   # Только Windows (amd64)
./build.sh package   # Создать distribution пакеты
./build.sh clean     # Очистить build артефакты
./build.sh help      # Показать справку
```

### PowerShell скрипт (build.ps1)

```powershell
.\build.ps1 all      # Собрать для всех платформ
.\build.ps1 linux    # Только Linux (amd64 + arm64)
.\build.ps1 windows  # Только Windows (amd64)
.\build.ps1 package  # Создать distribution пакеты
.\build.ps1 clean    # Очистить build артефакты
.\build.ps1 help     # Показать справку
```

### Makefile

```bash
make build-all       # Собрать для всех платформ
make build-linux     # Только Linux
make build-windows   # Только Windows
make package         # Создать пакеты
make clean           # Очистить
make test            # Запустить тесты
make lint            # Запустить линтеры
make help            # Показать все команды
```

---

## 🎯 Поддерживаемые платформы

| OS | Architecture | Binary Name | Status |
|----|--------------|-------------|--------|
| **Linux** | amd64 (x86_64) | `ollama-proxy-linux-amd64` | ✅ Supported |
| **Linux** | arm64 (aarch64) | `ollama-proxy-linux-arm64` | ✅ Supported |
| **Windows** | amd64 (x86_64) | `ollama-proxy-windows-amd64.exe` | ✅ Supported |
| **macOS** | amd64 (Intel) | `ollama-proxy-darwin-amd64` | 🔄 Not built by default |
| **macOS** | arm64 (Apple Silicon) | `ollama-proxy-darwin-arm64` | 🔄 Not built by default |

---

## 🔧 Custom Build

### Изменить версию

```bash
# Linux/macOS
VERSION=1.3.1 ./build.sh all

# Windows
.\build.ps1 all -Version 1.3.1

# Make
make build-all VERSION=1.3.1
```

### Manual Build

```bash
# Set target platform
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# Build
go build \
    -ldflags="-w -s -X main.Version=1.3.0" \
    -tags "sqlite_fts5 sqlite_json1" \
    -trimpath \
    -o dist/ollama-proxy \
    ./cmd/server
```

### Build with CGO (SQLite optimization)

```bash
# Enable CGO for better SQLite performance
export CGO_ENABLED=1

# Linux
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -tags "sqlite_fts5 sqlite_json1" \
    -o dist/ollama-proxy-linux-amd64 \
    ./cmd/server

# Note: Cross-compilation with CGO requires C cross-compiler
```

---

## 📦 Distribution Packages

### Создать пакеты

```bash
# Linux/macOS
./build.sh all
./build.sh package

# Windows
.\build.ps1 all
.\build.ps1 package

# Make
make release
```

### Структура пакета

```
ollama-proxy-1.3.0-linux-amd64.tar.gz
└── ollama-proxy-1.3.0-linux-amd64/
    ├── ollama-proxy-linux-amd64    # Binary
    ├── configs/                    # Config files
    │   ├── dev.yaml
    │   └── production.yaml.example
    ├── web/                        # WebUI files
    │   ├── index.html
    │   ├── css/
    │   └── js/
    ├── README.md                   # Documentation
    ├── LICENSE                     # License file
    └── start.sh                    # Start script
```

---

## 🔐 Build Flags Explained

### LDFLAGS

```
-w              # Omit DWARF debug information
-s              # Omit symbol table
-X main.Version # Set version variable
```

### Build Tags

```
sqlite_fts5     # Enable SQLite Full-Text Search
sqlite_json1    # Enable SQLite JSON support
```

### Other Flags

```
-trimpath       # Remove file paths from binary
CGO_ENABLED=0   # Disable CGO for static binary
```

---

## 📊 Build Sizes

| Platform | Binary Size | Notes |
|----------|-------------|-------|
| Linux amd64 | ~15-20 MB | Without CGO |
| Linux arm64 | ~15-20 MB | Without CGO |
| Windows amd64 | ~16-22 MB | .exe format |

*Sizes are approximate and may vary depending on Go version*

---

## 🐛 Troubleshooting

### Build fails with "undefined: sqlite3"

**Solution:** Enable CGO or use build tags:

```bash
go build -tags "sqlite_fts5 sqlite_json1" ./cmd/server
```

### Cross-compilation CGO error

**Problem:** Cannot cross-compile with CGO enabled

**Solution:** Either disable CGO or use Docker:

```bash
# Disable CGO
export CGO_ENABLED=0

# Or use Docker
docker run --rm -v "$PWD":/src -w /src golang:1.25 \
    go build -o dist/ollama-proxy ./cmd/server
```

### Windows antivirus blocks binary

**Problem:** Windows Defender flags the binary

**Solution:** This is a false positive. Sign the binary or add exception:

```powershell
# Add exception in Windows Defender
Add-MpPreference -ExclusionPath "C:\path\to\ollama-proxy.exe"
```

### Binary size too large

**Solution:** Use compression or UPX:

```bash
# Install UPX
# Linux: apt-get install upx
# macOS: brew install upx
# Windows: download from upx.github.io

# Compress binary
upx --best dist/ollama-proxy-*
```

---

## 🔄 CI/CD Integration

### GitHub Actions

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      
      - name: Build all platforms
        run: make build-all
      
      - name: Create packages
        run: make package
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: binaries
          path: dist/*
```

### GitLab CI

```yaml
build:
  image: golang:1.25
  script:
    - make build-all
    - make package
  artifacts:
    paths:
      - dist/
```

---

## 📚 Related Documentation

- [README.md](README.md) - Project overview
- [WEBUI.md](WEBUI.md) - WebUI documentation
- [Roadmap.MD](Roadmap.MD) - Development roadmap
- [configs/production.yaml.example](configs/production.yaml.example) - Configuration template

---

## 💡 Tips

1. **Use Make for convenience**: `make help` shows all commands
2. **Check binary with `file`**: Verify platform/architecture
3. **Test binary**: Always test before distribution
4. **Keep binaries small**: Use `-ldflags="-w -s"` flags
5. **Version your builds**: Always include version in filename

---

**Last Updated:** 2025-10-06  
**Version:** 1.3.0
