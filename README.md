# 🦙 Ollama-OpenAI Proxy

> **OpenAI-compatible API for local Ollama models with Enterprise-grade features**

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen.svg)](https://github.com/yourusername/ollama-openai-proxy)

## 📖 What is this?

Ollama-OpenAI Proxy is a **production-ready Go application** that provides an **OpenAI-compatible API** for local Ollama models. It serves as a bridge between clients expecting OpenAI API and your local Ollama server, enabling seamless integration with existing tools and workflows.

### 🎯 Key Features

- ✅ **Full OpenAI API Compatibility** - `/v1/chat/completions`, `/v1/models`, `/v1/embeddings`, `/v1/completions`
- 🔥 **Real-time Streaming** - Server-Sent Events (SSE) for live responses
- 🛠️ **Function Calling (Tools)** - OpenAI-style function calling with automatic model routing
- 🔐 **Enterprise Security** - API key management, rate limiting, permissions, bcrypt hashing
- 📊 **Professional Monitoring** - Prometheus metrics, Terminal UI, Web UI, detailed logs
- ⚡ **High Performance** - Connection pooling, circuit breakers, request timeout handling
- 🧠 **Intelligent Conversion** - Automatic OpenAI ↔ Ollama format conversion
- 🎨 **Dual UI Options** - Terminal UI (TUI) and Web UI (WebUI) for monitoring and management
- 📝 **Comprehensive Logging** - Structured logs with rotation, file output

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+** (for building from source)
- **Ollama** server running locally or remotely ([Download Ollama](https://ollama.ai/))
- At least one Ollama model installed (`ollama pull llama3.1`)

### Installation

#### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/ollama-openai-proxy.git
cd ollama-openai-proxy

# Install dependencies
go mod tidy

# Build the server, TUI and WebUI
go build -o bin/server.exe cmd/server/main.go
go build -o bin/tui.exe cmd/tui/*.go
go build -o bin/webui.exe cmd/webui/main.go

# Run the server
./bin/server.exe

# Or use air for hot-reload during development
air
```

#### Option 2: Pre-built Binaries

```bash
# Download from releases
# Coming soon...
```

### First Run

1. **Start Ollama** (if not already running):

   ```bash
   ollama serve
   ```

2. **Start the Proxy Server**:

   ```bash
   ./bin/server.exe
   # Server will start on http://localhost:8080
   ```

3. **Test the connection**:

```bash
curl http://localhost:8080/v1/models
   ```

4. **Launch TUI** (optional):

   ```bash
   ./bin/tui.exe
   # Press 1-7 to navigate between screens
   ```

5. **Launch WebUI** (optional):

   ```bash
   ./bin/webui.exe
   # WebUI will be available at http://localhost:8081
   # Open in your browser
   ```

---

## 📚 API Documentation

### Supported Endpoints

| Endpoint | Method | Description | OpenAI Compatible |
|----------|--------|-------------|-------------------|
| `/v1/chat/completions` | POST | Chat completions with streaming | ✅ Yes |
| `/v1/models` | GET | List available models | ✅ Yes |
| `/v1/embeddings` | POST | Generate embeddings | ✅ Yes |
| `/v1/completions` | POST | Legacy text completions | ✅ Yes |
| `/health` | GET | Health check | - |
| `/api/stats` | GET | Server statistics | - |
| `/api/config` | GET | Server configuration | - |
| `/metrics` | GET | Prometheus metrics | - |

### Chat Completions

#### Basic Request

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "qwen2.5-coder:7b",
    "messages": [
      {"role": "system", "content": "You are a helpful coding assistant."},
      {"role": "user", "content": "Write a bubble sort function in Go"}
    ],
    "temperature": 0.7,
    "max_tokens": 500
  }'
```

#### Streaming Request

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "qwen2.5-coder:7b", 
    "messages": [{"role": "user", "content": "Explain async/await in JavaScript"}],
    "stream": true
  }' \
  --no-buffer
```

#### Function Calling (Tools)

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "llama3.1",
    "messages": [
      {"role": "user", "content": "What is the weather in London?"}
    ],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_weather",
          "description": "Get current weather for a location",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {
                "type": "string",
                "description": "City name"
              }
            },
            "required": ["location"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

**Response:**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1699999999,
  "model": "llama3.1",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": null,
      "tool_calls": [{
        "id": "call_abc123",
        "type": "function",
        "function": {
          "name": "get_weather",
          "arguments": "{\"location\":\"London\"}"
        }
      }]
    },
    "finish_reason": "tool_calls"
  }]
}
```

### Embeddings

```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "nomic-embed-text",
    "input": ["Hello world", "Goodbye world"]
  }'
```

**Response:**

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.123, -0.456, 0.789, ...],
      "index": 0
    },
    {
      "object": "embedding",
      "embedding": [0.321, -0.654, 0.987, ...],
      "index": 1
    }
  ],
  "model": "nomic-embed-text",
  "usage": {
    "prompt_tokens": 4,
    "total_tokens": 4
  }
}
```

### Legacy Completions

```bash
curl -X POST http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "llama3.1",
    "prompt": "Once upon a time",
    "max_tokens": 100,
    "temperature": 0.7
  }'
```

---

## 🔐 API Key Management

### Creating API Keys (via TUI)

1. Launch TUI: `./bin/tui.exe`
2. Press `4` to go to API Keys screen
3. Press `n` to create a new key
4. Fill in the form:
   - **Name**: Descriptive name (e.g., "Development Key")
   - **Permissions**: Comma-separated models or `*` for all
   - **Rate Limits**: Optional (requests per minute/hour)
5. Press `Enter` to create
6. **Copy the displayed key immediately** - it won't be shown again!
7. Press `c` to copy to clipboard

### Using API Keys

Include the API key in the `Authorization` header:

```bash
curl -H "Authorization: Bearer sk-your-api-key-here" \
  http://localhost:8080/v1/models
```

### Admin API Key

The bootstrap admin key is defined in `configs/dev.yaml`:

```yaml
auth:
  admin_key: "sk-admin-dev-key-12345"
```

**⚠️ Change this in production!**

Admin keys have:

- Unlimited rate limits
- Access to all models
- Access to all API endpoints

---

## 🖥️ Terminal User Interface (TUI)

Launch the TUI for monitoring and management:

```bash
./bin/tui.exe
```

### TUI Features

| Screen | Key | Description |
|--------|-----|-------------|
| **Dashboard** | `1` | Overview, metrics, Prometheus stats |
| **Requests** | `2` | Request monitoring (placeholder) |
| **Models** | `3` | Available Ollama models |
| **API Keys** | `4` | Manage API keys, create new keys |
| **Config** | `5` | View server configuration |
| **Logs** | `6` | Real-time log viewer with color coding |
| **Control** | `7` | Server status, Ollama connection, statistics |

### TUI Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `1-7` | Navigate between screens |
| `n` | Create new API key (on API Keys screen) |
| `r` | Refresh statistics |
| `↑`/`↓` or `j`/`k` | Scroll up/down (line by line) |
| `PgUp`/`PgDn` | Scroll up/down (10 lines) |
| `Home` | Jump to top |
| `c` | Copy API key to clipboard (when shown) |
| `q` or `Ctrl+C` | Quit TUI |
| `Esc` | Clear error message |

### TUI Screenshots

**Dashboard:**

```
🦙 Ollama-OpenAI Proxy TUI
v1.0.0 | Обновлено: 12:34:56

[1: Dashboard] [2: Requests] [3: Models] [4: API Keys] ...

📊 СЕРВЕР
• Статус: ✅ Запущен
• Uptime: 2h15m30s
• Всего запросов: 1,234

📊 PROMETHEUS METRICS
• HTTP Requests: 1234 (0 in-flight)
• Ollama Requests: 1100 (5 errors)
• Avg HTTP Latency: 45.32 ms
• Avg Ollama Latency: 2345.67 ms
```

---

## 🌐 Web User Interface (WebUI)

Launch the WebUI for browser-based monitoring and management:

```bash
./bin/webui.exe
# WebUI will be available at http://localhost:8081
```

### WebUI Features

| Screen | Description |
|--------|-------------|
| **Dashboard** | Real-time metrics, server status, Ollama connection |
| **API Keys** | Create, view, and delete API keys with admin authentication |
| **Models** | Browse available Ollama models |
| **Config** | View server configuration (JSON format) |
| **Logs** | Real-time log viewer (coming soon) |

### WebUI Highlights

- ✨ **Modern Dark Theme** - Beautiful, responsive UI
- 🔄 **Auto-Refresh** - Updates every 5 seconds
- 📱 **Mobile-Friendly** - Works on any device
- 🔐 **Secure** - Admin key required for operations
- 📋 **Copy to Clipboard** - One-click API key copying
- 🚀 **Single Binary** - No Node.js required, everything embedded

### WebUI Configuration

```bash
# Change WebUI port
./bin/webui.exe -port 9000

# Connect to remote server
./bin/webui.exe -server-url http://192.168.1.100:8080

# Bind to specific host
./bin/webui.exe -host 127.0.0.1
```

**📚 Full documentation:**

- **Terminal UI**: [docs/TUI_GUIDE.md](docs/TUI_GUIDE.md)
- **Web UI**: [docs/WEBUI_GUIDE.md](docs/WEBUI_GUIDE.md)

---

## ⚙️ Configuration

Configuration is stored in `configs/dev.yaml` (development) or `configs/production.yaml` (production).

### Full Configuration Example

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "180s"  # High for streaming
  idle_timeout: "120s"
  max_header_bytes: 1048576  # 1MB

# Ollama connection
ollama:
  url: "http://localhost:11434"
  timeout: "180s"  # High timeout for large models
  retry_attempts: 3
  retry_delay: "2s"
  connection_pool_size: 100
  keep_alive: true

# Authentication
auth:
  enabled: true
  storage_type: "json"  # or "sqlite"
  storage_path: "data/api_keys.json"
  admin_key: "sk-admin-dev-key-12345"  # ⚠️ CHANGE IN PRODUCTION!
  
  # Rate limiting
  rate_limiting:
    enabled: true
    default_requests_per_minute: 30
    default_requests_per_hour: 500

# Logging
logging:
  level: "debug"  # debug, info, warn, error
  format: "text"  # text or json
  output: "both"  # stdout, file, or both
  file_path: "logs/proxy-dev.log"
  max_size: 100  # MB
  max_backups: 5
  max_age: 30  # days
  compress: true

# Models
models:
  mapping: {}  # OpenAI name -> Ollama name
  aliases: {}  # Custom aliases
  hidden: []   # Hide specific models
  cache:
    enabled: true
    ttl: "5m"
    refresh_interval: "1m"

# Tools (Function Calling)
tools:
  force_usage: false  # Auto-apply tool_choice: required
  default_choice: "auto"  # auto, required, none
  fallback_model: "llama3.1"  # Model for tool calls
  
  optimizer:
    enabled: true
    simplify_system_message: true
    smart_tool_filtering: false
    max_tools_per_request: 0  # 0 = unlimited

# Prometheus Metrics
metrics:
  enabled: true
  prometheus_path: "/metrics"
  
  collection:
    enabled: true
    buffer_size: 1000
    flush_interval: "10s"

# Terminal UI
tui:
  enabled: true
  refresh_rate: "1s"
  theme: "default"

# Development
development:
  hot_reload: true
  debug_mode: true
  profile_enabled: false
  pprof_enabled: false
  race_detection: true
```

### Environment Variables

You can override configuration with environment variables:

```bash
export PROXY_SERVER_PORT=9000
export PROXY_OLLAMA_URL="http://remote-server:11434"
export PROXY_AUTH_ADMIN_KEY="sk-secure-production-key"
```

See [CONFIGURATION.md](docs/CONFIGURATION.md) for full documentation.

---

## 📊 Monitoring & Metrics

### Prometheus Metrics

Metrics are exposed at `/metrics` endpoint:

```bash
curl http://localhost:8080/metrics
```

**Available metrics:**

| Metric | Type | Description |
|--------|------|-------------|
| `ollama_proxy_http_requests_total` | Counter | Total HTTP requests |
| `ollama_proxy_http_request_duration_seconds` | Histogram | HTTP request duration |
| `ollama_proxy_http_requests_in_flight` | Gauge | Current active requests |
| `ollama_proxy_http_response_size_bytes` | Histogram | Response sizes |
| `ollama_proxy_ollama_requests_total` | Counter | Ollama API calls |
| `ollama_proxy_ollama_request_duration_seconds` | Histogram | Ollama request duration |
| `ollama_proxy_ollama_errors_total` | Counter | Ollama errors |
| `ollama_proxy_api_key_requests_total` | Counter | Requests per API key |
| `ollama_proxy_api_key_rate_limit_exceeded_total` | Counter | Rate limit violations |
| `ollama_proxy_api_keys_active_total` | Gauge | Active API keys count |

### Grafana Dashboard

Import the provided Grafana dashboard (coming soon) for visualization.

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLIENT APPLICATIONS                       │
│         (Zed IDE, Continue.dev, VSCode, Custom Apps)            │
└───────────────────────────┬─────────────────────────────────────┘
                            │ OpenAI API Format
                            │ (Authorization: Bearer sk-xxx)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      OLLAMA-OPENAI PROXY                         │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  HTTP Server (Gin)                                          │ │
│  │  • /v1/chat/completions  • /v1/models                      │ │
│  │  • /v1/embeddings        • /v1/completions                 │ │
│  └────────────────────────────────────────────────────────────┘ │
│                            │                                     │
│  ┌─────────────────────────┴────────────────────────────────┐  │
│  │  Middleware Pipeline                                      │  │
│  │  ┌──────────┐ ┌──────────┐ ┌─────────┐ ┌──────────────┐ │  │
│  │  │   Auth   │→│   Rate   │→│  Stats  │→│  Prometheus  │ │  │
│  │  │   Keys   │ │ Limiting │ │         │ │   Metrics    │ │  │
│  │  └──────────┘ └──────────┘ └─────────┘ └──────────────┘ │  │
│  └────────────────────────────────────────────────────────────┘ │
│                            │                                     │
│  ┌─────────────────────────┴────────────────────────────────┐  │
│  │  Converters & Handlers                                    │  │
│  │  • OpenAI → Ollama format conversion                      │  │
│  │  • Streaming SSE handling                                 │  │
│  │  • Tools/Function calling support                         │  │
│  │  • Prompt optimization                                    │  │
│  └────────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────────┘
                            │ Ollama API Format
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                        OLLAMA SERVER                             │
│              (llama3.1, qwen2.5-coder, etc.)                    │
└─────────────────────────────────────────────────────────────────┘

                  ┌─────────────────────┐
                  │   TERMINAL UI (TUI) │
                  │   • Monitoring      │
                  │   • API Keys Mgmt   │
                  │   • Logs Viewer     │
                  │   • Config Viewer   │
                  └─────────────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

---

## 🧪 Testing

The project has **100% test coverage** across critical components.

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detection
go test -race ./...

# Specific package
go test ./internal/auth/...
```

### Test Categories

- **Unit Tests**: All core packages (`internal/auth`, `internal/client`, etc.)
- **Integration Tests**: API handlers, middleware chains
- **Security Tests**: Timing attacks, brute force, injection
- **Performance Tests**: Rate limiting, concurrent operations
- **Penetration Tests**: Invalid keys, privilege escalation

---

## 🚀 Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/ollama-proxy.service`:

```ini
[Unit]
Description=Ollama-OpenAI Proxy Server
After=network.target ollama.service
Requires=ollama.service

[Service]
Type=simple
User=ollama-proxy
WorkingDirectory=/opt/ollama-proxy
ExecStart=/opt/ollama-proxy/bin/server
Restart=on-failure
RestartSec=5s

# Environment
Environment="PROXY_SERVER_PORT=8080"
Environment="PROXY_OLLAMA_URL=http://localhost:11434"

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/ollama-proxy/logs /opt/ollama-proxy/data

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable ollama-proxy
sudo systemctl start ollama-proxy
sudo systemctl status ollama-proxy
```

### Windows Service

Use [NSSM](https://nssm.cc/) to create a Windows service:

```cmd
nssm install OllamaProxy "C:\ollama-proxy\bin\server.exe"
nssm set OllamaProxy AppDirectory "C:\ollama-proxy"
nssm start OllamaProxy
```

---

## 🔧 Troubleshooting

### Common Issues

#### 1. "Connection refused" to Ollama

**Problem**: Proxy can't connect to Ollama server.

**Solution**:

```bash
# Check Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if needed
ollama serve

# Check firewall (Linux)
sudo ufw allow 11434
```

#### 2. API key not working

**Problem**: `401 Unauthorized` or `API key not found`.

**Solution**:

- Ensure `Authorization: Bearer sk-xxx` header is present
- Check key is not disabled in TUI (screen 4)
- Verify key has permissions for the requested model
- Check rate limits haven't been exceeded

#### 3. Slow responses

**Problem**: Requests take too long.

**Solution**:

```yaml
# Increase timeouts in config
ollama:
  timeout: "300s"

server:
  write_timeout: "300s"
```

#### 4. Tool calling not working

**Problem**: Model returns text instead of tool calls.

**Solution**:

- Use a model that supports tools: `llama3.1`, `llama3.2`, `mistral`, `qwen2.5`
- Set `fallback_model: "llama3.1"` in config
- Check logs for tool call detection issues

See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for more solutions.

---

## 📈 Performance Tuning

### For RTX 4090 (24GB VRAM)

```yaml
# Ollama configuration (in Ollama's config)
# Adjust num_ctx and num_predict based on model size

# For 7B models (e.g., qwen2.5-coder:7b)
num_ctx: 32768      # Context window
num_predict: 4096   # Max tokens to generate

# For 30B models (e.g., qwen2.5-coder:30b)
num_ctx: 16384
num_predict: 2048

# Proxy configuration
ollama:
  timeout: "180s"
  connection_pool_size: 50

server:
  write_timeout: "180s"
```

### Connection Pool Tuning

```yaml
ollama:
  connection_pool_size: 100  # Increase for high traffic
  keep_alive: true           # Reuse connections
```

See [docs/PERFORMANCE.md](docs/PERFORMANCE.md) for comprehensive tuning guide.

---

## 📦 Project Structure

```
ollama-openai-proxy/
├── cmd/
│   ├── server/              # HTTP server entry point
│   │   └── main.go
│   └── tui/                 # Terminal UI entry point
│       ├── main.go
│       ├── logs_reader.go   # Log file viewer
│       ├── metrics_parser.go # Prometheus parser
│       ├── scroll_helper.go  # Scrolling utilities
│       └── time_helper.go    # Time parsing
├── internal/
│   ├── api/
│   │   ├── handlers/        # HTTP request handlers
│   │   │   ├── chat.go      # Chat completions
│   │   │   ├── embeddings.go # Embeddings API
│   │   │   ├── completions.go # Legacy completions
│   │   │   ├── models.go    # Models listing
│   │   │   ├── stats.go     # Statistics endpoint
│   │   │   └── config.go    # Configuration endpoint
│   │   ├── middleware/      # HTTP middleware
│   │   │   ├── auth_keys.go # API key authentication
│   │   │   ├── logging.go   # Request logging
│   │   │   ├── stats.go     # Statistics collection
│   │   │   ├── metrics.go   # Prometheus metrics
│   │   │   ├── cors.go      # CORS handling
│   │   │   └── recovery.go  # Panic recovery
│   │   └── router/          # Route configuration
│   ├── auth/
│   │   ├── apikey/          # API key manager
│   │   │   ├── manager.go
│   │   │   └── storage.go   # JSON/SQLite storage
│   │   └── ratelimit/       # Rate limiting
│   ├── client/
│   │   └── ollama/          # Ollama API client
│   │       ├── client.go
│   │       ├── methods.go   # Chat, Embed, etc.
│   │       └── streaming.go # SSE streaming
│   ├── config/              # Configuration management
│   │   └── config.go
│   ├── converter/           # Request/Response converters
│   │   ├── request.go       # OpenAI → Ollama
│   │   ├── response.go      # Ollama → OpenAI
│   │   ├── streaming.go     # Streaming conversion
│   │   ├── embeddings.go    # Embeddings conversion
│   │   └── completions.go   # Completions conversion
│   ├── logger/              # Logging utilities
│   ├── metrics/             # Prometheus metrics
│   │   └── metrics.go
│   ├── models/              # Data models
│   │   ├── openai.go
│   │   └── ollama.go
│   └── optimizer/           # Prompt optimization
│       └── prompt_optimizer.go
├── configs/
│   ├── dev.yaml             # Development config
│   └── production.yaml.example # Production template
├── data/                    # Runtime data
│   └── api_keys.json        # API keys storage
├── logs/                    # Log files
│   └── proxy-dev.log
├── docs/                    # Documentation
│   ├── TUI_GUIDE.md
│   ├── API_DOCUMENTATION.md
│   ├── CONFIGURATION.md
│   ├── TROUBLESHOOTING.md
│   ├── ARCHITECTURE.md
│   └── PERFORMANCE.md
├── tests/                   # Test files
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 🤝 Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork** the repository
2. Create a **feature branch** (`git checkout -b feature/amazing-feature`)
3. **Write tests** for your changes (100% coverage required)
4. Run **linters** (`go vet`, `golangci-lint`)
5. **Commit** your changes (`git commit -m 'Add amazing feature'`)
6. **Push** to the branch (`git push origin feature/amazing-feature`)
7. Open a **Pull Request**

### Code Standards

- Follow Go best practices and idioms
- Document all exported functions and types
- Write comprehensive tests
- Use structured logging (logrus)
- Handle errors explicitly
- Use contexts for cancellation

See [.github/CONTRIBUTING.md](.github/CONTRIBUTING.md) for details.

---

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- **Ollama** team for the amazing local LLM platform
- **OpenAI** for the API standard
- **Go community** for excellent libraries:
  - [Gin](https://github.com/gin-gonic/gin) - HTTP framework
  - [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
  - [Viper](https://github.com/spf13/viper) - Configuration
  - [Logrus](https://github.com/sirupsen/logrus) - Logging
  - [Prometheus](https://github.com/prometheus/client_golang) - Metrics

---

## 📚 Additional Documentation

- **[TUI Guide](docs/TUI_GUIDE.md)** - Complete Terminal UI documentation
- **[API Documentation](docs/API_DOCUMENTATION.md)** - Full API reference
- **[Configuration](docs/CONFIGURATION.md)** - All configuration options
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Architecture](docs/ARCHITECTURE.md)** - System architecture details
- **[Performance](docs/PERFORMANCE.md)** - Performance tuning guide
- **[Plan.md](Plan.md)** - Development roadmap
- **[Architecture.MD](Architecture.MD)** - Original architecture document
- **[MVP.MD](MVP.MD)** - MVP criteria

---

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/yourusername/ollama-openai-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/yourusername/ollama-openai-proxy/discussions)
- **Email**: <your.email@example.com>

---

## 🗺️ Roadmap

- [x] OpenAI API compatibility
- [x] Streaming support
- [x] Function calling (Tools)
- [x] API key management
- [x] Terminal UI
- [x] Prometheus metrics
- [x] Embeddings API
- [x] Legacy Completions API
- [x] **Web UI (Phase 14)** ✅ **ЗАВЕРШЕНО**
  - [x] Dashboard с real-time метриками
  - [x] API Keys Management
  - [x] Extended Model Info
  - [x] Real-time Logs Viewer
  - [x] Toast Notifications
  - [x] Dark/Light Theme Toggle
  - [x] Export Functions (CSV, JSON)
- [ ] WebSocket для real-time updates (Phase 12.2)
- [ ] Advanced Metrics Collection (Phase 12.1)
- [ ] Multi-user support
- [ ] Usage analytics dashboard с графиками
- [ ] Model fine-tuning integration

---

**Made with ❤️ by the Ollama-OpenAI Proxy team**

Star ⭐ this repo if you find it useful!
