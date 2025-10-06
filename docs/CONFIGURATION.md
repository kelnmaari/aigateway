# ⚙️ Configuration Guide

Complete configuration reference for Ollama-OpenAI Proxy.

## 📑 Table of Contents

- [Overview](#overview)
- [Configuration Files](#configuration-files)
- [Server Configuration](#server-configuration)
- [Ollama Configuration](#ollama-configuration)
- [Authentication](#authentication)
- [Logging](#logging)
- [Models](#models)
- [Prompts](#prompts)
- [Tools (Function Calling)](#tools-function-calling)
- [Metrics](#metrics)
- [Terminal UI](#terminal-ui)
- [Development](#development)
- [Environment Variables](#environment-variables)
- [Production Configuration](#production-configuration)
- [Best Practices](#best-practices)

---

## Overview

Ollama-OpenAI Proxy uses **YAML configuration files** powered by Viper. Configuration can be overridden with **environment variables**.

### Configuration Hierarchy

1. **Config file** (`configs/dev.yaml` or `configs/production.yaml`)
2. **Environment variables** (`PROXY_*`)
3. **Command-line flags** (if implemented)

Higher priority wins (env vars override config file).

---

## Configuration Files

### File Locations

```
configs/
├── dev.yaml                  # Development configuration
└── production.yaml.example   # Production template (copy & customize)
```

### Loading Configuration

The proxy automatically loads:
- `configs/dev.yaml` in development
- `configs/production.yaml` in production (based on `GO_ENV`)

```bash
# Development (default)
./bin/server.exe

# Production
GO_ENV=production ./bin/server
```

---

## Server Configuration

HTTP server settings.

### Configuration

```yaml
server:
  port: 8080                    # Server port
  host: "0.0.0.0"              # Bind address (0.0.0.0 = all interfaces)
  read_timeout: "5m"           # Read timeout (5 minutes for large models)
  write_timeout: "5m"          # Write timeout (5 minutes for streaming)
  idle_timeout: "120s"         # Idle connection timeout
  max_header_bytes: 1048576    # Max HTTP header size (1MB)
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `port` | int | `8080` | HTTP server port |
| `host` | string | `"0.0.0.0"` | Bind address (`0.0.0.0` = all, `localhost` = local only) |
| `read_timeout` | duration | `"30s"` | Max time to read request |
| `write_timeout` | duration | `"30s"` | Max time to write response |
| `idle_timeout` | duration | `"120s"` | Keep-alive timeout |
| `max_header_bytes` | int | `1048576` | Max HTTP header size in bytes |

### Recommendations

**Development:**
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "5m"
  write_timeout: "5m"
```

**Production (LAN):**
```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "2m"
  write_timeout: "2m"
```

**Production (Internet-facing):**
```yaml
server:
  host: "127.0.0.1"  # Use reverse proxy (nginx/caddy)
  port: 8080
  read_timeout: "1m"
  write_timeout: "1m"
```

### Timeout Considerations

- **Small models (< 10B)**: 30s-1m
- **Large models (> 30B)**: 2m-5m
- **First load**: +60s (model loading time)
- **Streaming**: Use same as write_timeout

---

## Ollama Configuration

Connection settings for Ollama server.

### Configuration

```yaml
ollama:
  url: "http://localhost:11434"  # Ollama server URL
  timeout: "5m"                   # Request timeout (5 min for first load)
  retry_attempts: 3               # Number of retry attempts
  retry_delay: "2s"               # Delay between retries
  connection_pool_size: 10        # HTTP connection pool size
  keep_alive: true                # Enable connection reuse
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `url` | string | `"http://localhost:11434"` | Ollama server URL |
| `timeout` | duration | `"30s"` | Request timeout |
| `retry_attempts` | int | `3` | Number of retry attempts on failure |
| `retry_delay` | duration | `"1s"` | Delay between retry attempts |
| `connection_pool_size` | int | `10` | Max concurrent connections |
| `keep_alive` | bool | `true` | Reuse connections |

### Remote Ollama Server

```yaml
ollama:
  url: "http://192.168.1.100:11434"  # Remote server
  timeout: "10m"                      # Higher timeout over network
  retry_attempts: 5                   # More retries over network
  connection_pool_size: 50            # Higher concurrency
```

### SSH Tunnel

```bash
# Create SSH tunnel
ssh -L 11434:localhost:11434 user@remote-server

# Use local forwarded port
ollama:
  url: "http://localhost:11434"
  timeout: "10m"  # Higher timeout through SSH
```

---

## Authentication

API key authentication and rate limiting.

### Configuration

```yaml
auth:
  enabled: true                        # Enable authentication
  storage_type: "json"                 # "json" or "sqlite"
  storage_path: "data/api_keys.json"  # Storage file path
  admin_key: "sk-admin-dev-key-12345" # Bootstrap admin key
  
  # Rate limiting
  rate_limiting:
    enabled: true                      # Enable rate limiting
    default_requests_per_minute: 60    # Default limit per minute
    default_requests_per_hour: 1000    # Default limit per hour
    
    # Redis for distributed rate limiting (optional)
    redis:
      enabled: false
      url: "redis://localhost:6379"
      key_prefix: "ollama_proxy_rl:"
  
  # JWT for internal tokens
  jwt:
    secret: "dev-jwt-secret-for-testing"
    expiry: "24h"
```

### Parameters

#### Authentication

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable API key authentication |
| `storage_type` | string | `"json"` | Storage type: `json` or `sqlite` |
| `storage_path` | string | `"data/api_keys.json"` | Storage file path |
| `admin_key` | string | `""` | Bootstrap admin API key |

#### Rate Limiting

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `rate_limiting.enabled` | bool | `true` | Enable rate limiting |
| `rate_limiting.default_requests_per_minute` | int | `30` | Default requests per minute |
| `rate_limiting.default_requests_per_hour` | int | `500` | Default requests per hour |

#### Redis (Optional)

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `rate_limiting.redis.enabled` | bool | `false` | Use Redis for distributed rate limiting |
| `rate_limiting.redis.url` | string | `""` | Redis connection URL |
| `rate_limiting.redis.key_prefix` | string | `"ollama_proxy_rl:"` | Redis key prefix |

### Security Best Practices

**⚠️ CRITICAL: Change admin key in production!**

```yaml
# ❌ DON'T use default key in production!
admin_key: "sk-admin-dev-key-12345"

# ✅ Generate secure random key
admin_key: "sk-$(openssl rand -hex 32)"
```

**Storage Options:**

- **JSON** (small deployments): Simple, file-based
- **SQLite** (medium deployments): Better performance, concurrent access
- **Future: PostgreSQL/MySQL** (large deployments): Planned

**Rate Limiting:**

```yaml
# Conservative (public API)
rate_limiting:
  default_requests_per_minute: 10
  default_requests_per_hour: 100

# Moderate (internal API)
rate_limiting:
  default_requests_per_minute: 30
  default_requests_per_hour: 500

# High (trusted clients)
rate_limiting:
  default_requests_per_minute: 100
  default_requests_per_hour: 10000
```

---

## Logging

Structured logging configuration.

### Configuration

```yaml
logging:
  level: "debug"                 # Log level: debug, info, warn, error
  format: "text"                 # Format: text or json
  output: "both"                 # Output: stdout, file, or both
  file_path: "logs/proxy-dev.log" # Log file path
  max_size: 100                  # Max file size in MB before rotation
  max_backups: 5                 # Number of old log files to keep
  max_age: 30                    # Days to keep old log files
  compress: true                 # Compress rotated logs (gzip)
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `level` | string | `"info"` | Log level: `debug`, `info`, `warn`, `error` |
| `format` | string | `"text"` | Output format: `text` or `json` |
| `output` | string | `"stdout"` | Output destination: `stdout`, `file`, `both` |
| `file_path` | string | `"logs/proxy.log"` | Log file path (when output includes `file`) |
| `max_size` | int | `100` | Max file size in MB before rotation |
| `max_backups` | int | `5` | Number of old log files to keep |
| `max_age` | int | `30` | Days to keep old log files |
| `compress` | bool | `true` | Compress rotated logs with gzip |

### Log Levels

**DEBUG** - Detailed diagnostic information
```
time="2025-10-04 12:30:20" level=debug msg="API key validated" key_id="ak_123..."
time="2025-10-04 12:30:21" level=debug msg="Request body" body="{\"model\":\"llama3.1\"...}"
```

**INFO** - General informational messages
```
time="2025-10-04 12:30:15" level=info msg="Server started" port=8080
time="2025-10-04 12:30:16" level=info msg="Connected to Ollama" url="http://localhost:11434"
```

**WARN** - Warning messages
```
time="2025-10-04 12:30:25" level=warning msg="Rate limit approaching" key_id="ak_456..." remaining=5
time="2025-10-04 12:30:30" level=warning msg="Slow request" duration="5.2s" model="qwen2.5-coder:30b"
```

**ERROR** - Error messages
```
time="2025-10-04 12:30:35" level=error msg="Ollama timeout" model="qwen2.5-coder:30b" error="context deadline exceeded"
time="2025-10-04 12:30:40" level=error msg="Authentication failed" reason="invalid API key"
```

### Recommendations

**Development:**
```yaml
logging:
  level: "debug"
  format: "text"
  output: "both"
  max_size: 100
  max_backups: 3
```

**Production:**
```yaml
logging:
  level: "info"
  format: "json"    # Easier for log aggregation
  output: "file"    # Don't pollute stdout
  max_size: 500     # Larger files
  max_backups: 10   # Keep more history
  compress: true
```

**Production (with ELK/Loki):**
```yaml
logging:
  level: "info"
  format: "json"
  output: "stdout"  # Pipe to log collector
```

### Log Rotation

Automatic rotation when file reaches `max_size`:

```
logs/
├── proxy-dev.log           # Current log
├── proxy-dev-2025-10-03.log.gz
├── proxy-dev-2025-10-02.log.gz
└── proxy-dev-2025-10-01.log.gz
```

Old logs are automatically deleted after `max_age` days or when count exceeds `max_backups`.

---

## Models

Model listing and caching configuration.

### Configuration

```yaml
models:
  # Model listing cache settings
  cache:
    enabled: true             # Enable model list caching
    ttl: "5m"                 # Cache time-to-live
    refresh_interval: "1m"    # Background refresh interval
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `cache.enabled` | bool | `true` | Enable caching of model list |
| `cache.ttl` | duration | `"5m"` | Cache time-to-live |
| `cache.refresh_interval` | duration | `"1m"` | Background refresh interval |

### Caching Strategy

1. **First request**: Fetches models from Ollama, caches result
2. **Subsequent requests** (within TTL): Returns cached result
3. **Background refresh** (every `refresh_interval`): Updates cache asynchronously
4. **Cache expiry** (after TTL): Forces fresh fetch on next request

### Recommendations

**Frequent model changes:**
```yaml
models:
  cache:
    enabled: true
    ttl: "1m"
    refresh_interval: "30s"
```

**Stable model list:**
```yaml
models:
  cache:
    enabled: true
    ttl: "30m"
    refresh_interval: "10m"
```

**No caching:**
```yaml
models:
  cache:
    enabled: false
```

---

## Prompts

System prompts and message customization.

### Configuration

```yaml
prompts:
  # Additional system message automatically added to all requests
  additional_system_message: |
    ВАЖНО: Всегда отвечай на русском языке, независимо от языка запроса пользователя.
    Используй простой и понятный стиль общения.
  
  # Add at start of system message (true) or end (false)
  prepend_to_system: false
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `additional_system_message` | string | `""` | Additional system message for all requests |
| `prepend_to_system` | bool | `false` | Add message at start (true) or end (false) |

### Use Cases

**Force language:**
```yaml
prompts:
  additional_system_message: "Always respond in Russian, regardless of input language."
  prepend_to_system: false
```

**Set personality:**
```yaml
prompts:
  additional_system_message: "You are a friendly coding assistant. Use emojis."
  prepend_to_system: true  # High priority
```

**Disable:**
```yaml
prompts:
  additional_system_message: ""  # Empty = disabled
```

---

## Tools (Function Calling)

Function calling configuration and optimization.

### Configuration

```yaml
tools:
  # Automatically apply tool_choice: "required" when tools are present
  force_usage: false
  
  # Default tool_choice if client doesn't specify
  default_choice: "auto"  # auto, required, none
  
  # Tool-aware model routing - automatically switch to tool-capable model
  fallback_model: "llama3.1"  # Model for requests with tools
  
  # Prompt optimizer for IDE integration
  optimizer:
    enabled: true                    # Enable optimization
    simplify_system_message: true    # Simplify verbose system messages
    smart_tool_filtering: false      # Filter irrelevant tools
    max_tools_per_request: 14        # Max tools per request (0 = unlimited)
    
    # Instructions to preserve in simplified system message
    preserve_instructions:
      - "When user asks to find/list/show files, use find_path or list_directory tool."
      - "When user asks to read a file, use read_file tool."
      - "When user asks to edit or create a file, use edit_file tool."
      - "Use grep tool to search code content, not paths."
```

### Parameters

#### Basic Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `force_usage` | bool | `false` | Auto-apply `tool_choice: "required"` |
| `default_choice` | string | `"auto"` | Default tool_choice: `auto`, `required`, `none` |
| `fallback_model` | string | `""` | Model for tool requests (empty = use requested model) |

#### Optimizer Settings

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `optimizer.enabled` | bool | `true` | Enable prompt optimization |
| `optimizer.simplify_system_message` | bool | `true` | Simplify verbose system messages |
| `optimizer.smart_tool_filtering` | bool | `false` | Filter irrelevant tools by semantic matching |
| `optimizer.max_tools_per_request` | int | `0` | Max tools per request (0 = unlimited) |
| `optimizer.preserve_instructions` | array | `[]` | Instructions to keep in simplified messages |

### Tool-Aware Model Routing

Automatically routes requests with `tools` to a tool-capable model:

```yaml
tools:
  fallback_model: "llama3.1"
```

**Behavior:**
- Request **without tools** → Uses requested model (e.g., `qwen2.5-coder:7b`)
- Request **with tools** → Automatically switches to `llama3.1`

**Recommended models for tools:**
- `llama3.1` - Best tool calling support (Meta)
- `llama3.2` - Improved tool calling
- `mistral` - Good tool support
- `qwen2.5` - Experimental tool support

### Prompt Optimizer

Optimizes complex system messages from IDEs (Zed, Cursor, VSCode):

**Problem:** IDEs send huge system messages (6000+ chars) with:
- Verbose communication guidelines
- Unnecessary formatting examples
- Redundant instructions

**Solution:** Optimizer reduces to essentials (~1200 chars):
- Tool usage instructions
- Critical communication rules
- Preserved custom instructions

**Results:**
- 80% reduction in system message length
- Faster model processing
- Better tool invocation accuracy

**Enable/Disable:**
```yaml
# For IDEs (Zed, Cursor, Continue.dev)
tools:
  optimizer:
    enabled: true
    simplify_system_message: true
    max_tools_per_request: 14

# For direct API usage (already optimized prompts)
tools:
  optimizer:
    enabled: false
```

### Best Practices

**IDE Usage (Zed, Cursor):**
```yaml
tools:
  force_usage: false              # Let model decide
  default_choice: "auto"
  fallback_model: "llama3.1"     # Tool-capable model
  optimizer:
    enabled: true
    simplify_system_message: true
    smart_tool_filtering: false  # Can filter wrong tools
    max_tools_per_request: 14
```

**Direct API Usage:**
```yaml
tools:
  force_usage: false
  default_choice: "auto"
  fallback_model: ""             # Use requested model
  optimizer:
    enabled: false               # Prompts already optimized
```

**Force Tool Usage:**
```yaml
tools:
  force_usage: true              # Always require tool calls
  default_choice: "required"
  fallback_model: "llama3.1"
```

---

## Metrics

Prometheus metrics collection.

### Configuration

```yaml
metrics:
  enabled: true                # Enable metrics collection
  prometheus_path: "/metrics"  # Metrics endpoint path
  
  # Internal metrics collection
  collection:
    enabled: true              # Enable internal collection
    buffer_size: 1000          # Metrics buffer size
    flush_interval: "10s"      # Flush interval
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable Prometheus metrics |
| `prometheus_path` | string | `"/metrics"` | Metrics endpoint path |
| `collection.enabled` | bool | `true` | Enable internal metrics collection |
| `collection.buffer_size` | int | `1000` | Metrics buffer size |
| `collection.flush_interval` | duration | `"10s"` | Metrics flush interval |

### Available Metrics

See [API_DOCUMENTATION.md](API_DOCUMENTATION.md#get-metrics) for full metrics list.

**Key metrics:**
- `ollama_proxy_http_requests_total` - Total HTTP requests
- `ollama_proxy_http_request_duration_seconds` - Request latency
- `ollama_proxy_ollama_requests_total` - Ollama API calls
- `ollama_proxy_api_keys_active_total` - Active API keys
- `ollama_proxy_api_key_rate_limit_exceeded_total` - Rate limit violations

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Disable Metrics

```yaml
metrics:
  enabled: false
```

---

## Terminal UI

TUI (Terminal User Interface) settings.

### Configuration

```yaml
tui:
  enabled: true           # Enable TUI
  refresh_rate: "1s"      # Data refresh interval
  theme: "default"        # Color theme
  
  # Dashboard settings
  dashboard:
    charts_history: 100   # Historical data points
    auto_refresh: true    # Auto-refresh data
  
  # Request monitor settings
  request_monitor:
    max_requests: 1000    # Max requests to track
    auto_scroll: true     # Auto-scroll to latest
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable TUI |
| `refresh_rate` | duration | `"1s"` | Data refresh interval |
| `theme` | string | `"default"` | Color theme |
| `dashboard.charts_history` | int | `100` | Chart history points |
| `dashboard.auto_refresh` | bool | `true` | Auto-refresh charts |
| `request_monitor.max_requests` | int | `1000` | Max requests to track |
| `request_monitor.auto_scroll` | bool | `true` | Auto-scroll to latest |

### Recommendations

**Real-time monitoring:**
```yaml
tui:
  refresh_rate: "500ms"  # Faster updates
  dashboard:
    charts_history: 200  # More history
```

**Resource-constrained:**
```yaml
tui:
  refresh_rate: "5s"     # Slower updates
  dashboard:
    charts_history: 50   # Less history
```

---

## Development

Development and debugging settings.

### Configuration

```yaml
development:
  hot_reload: true        # Enable hot reload
  debug_mode: true        # Enable debug mode
  profile_enabled: true   # Enable profiling
  pprof_enabled: true     # Enable pprof endpoints
  race_detection: true    # Enable race detector
  
  # Mock settings (when Ollama unavailable)
  mock_ollama:
    enabled: true         # Enable mock Ollama
    response_delay: "500ms"  # Mock response delay
    random_errors: false  # Randomly return errors
    error_rate: 0.1       # Error probability (0-1)
```

### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `hot_reload` | bool | `false` | Enable hot reload (requires air/CompileDaemon) |
| `debug_mode` | bool | `false` | Enable debug logging |
| `profile_enabled` | bool | `false` | Enable CPU/memory profiling |
| `pprof_enabled` | bool | `false` | Enable pprof HTTP endpoints |
| `race_detection` | bool | `false` | Enable race detector (slow) |

### Profiling

When `pprof_enabled: true`, available at:
- `http://localhost:8080/debug/pprof/` - Index
- `http://localhost:8080/debug/pprof/heap` - Heap profile
- `http://localhost:8080/debug/pprof/goroutine` - Goroutines
- `http://localhost:8080/debug/pprof/profile` - CPU profile (30s)

**Usage:**
```bash
# CPU profile
go tool pprof http://localhost:8080/debug/pprof/profile

# Heap profile
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine profile
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

### Mock Ollama

For testing without Ollama:

```yaml
development:
  mock_ollama:
    enabled: true
    response_delay: "1s"    # Simulate slow model
    random_errors: true     # Test error handling
    error_rate: 0.2         # 20% error rate
```

---

## Environment Variables

Override configuration with environment variables.

### Format

```bash
PROXY_<SECTION>_<KEY>=value
```

**Examples:**
```bash
export PROXY_SERVER_PORT=9000
export PROXY_OLLAMA_URL="http://192.168.1.100:11434"
export PROXY_AUTH_ADMIN_KEY="sk-production-key"
export PROXY_LOGGING_LEVEL="info"
```

### Common Environment Variables

```bash
# Server
PROXY_SERVER_PORT=8080
PROXY_SERVER_HOST="0.0.0.0"

# Ollama
PROXY_OLLAMA_URL="http://localhost:11434"
PROXY_OLLAMA_TIMEOUT="5m"

# Authentication
PROXY_AUTH_ENABLED=true
PROXY_AUTH_ADMIN_KEY="sk-your-admin-key"
PROXY_AUTH_RATE_LIMITING_ENABLED=true

# Logging
PROXY_LOGGING_LEVEL="info"
PROXY_LOGGING_OUTPUT="file"
PROXY_LOGGING_FILE_PATH="logs/proxy.log"

# Tools
PROXY_TOOLS_FALLBACK_MODEL="llama3.1"
PROXY_TOOLS_OPTIMIZER_ENABLED=true

# Metrics
PROXY_METRICS_ENABLED=true
```

### Docker Example

```bash
docker run -e PROXY_SERVER_PORT=8080 \
           -e PROXY_OLLAMA_URL="http://host.docker.internal:11434" \
           -e PROXY_AUTH_ADMIN_KEY="sk-secure-key" \
           ollama-openai-proxy
```

---

## Production Configuration

Recommended production settings.

### Production Template

```yaml
# configs/production.yaml

server:
  port: 8080
  host: "127.0.0.1"  # Use reverse proxy
  read_timeout: "2m"
  write_timeout: "2m"
  idle_timeout: "120s"

ollama:
  url: "http://localhost:11434"
  timeout: "5m"
  retry_attempts: 5
  retry_delay: "2s"
  connection_pool_size: 50
  keep_alive: true

auth:
  enabled: true
  storage_type: "sqlite"  # Better performance
  storage_path: "/data/api_keys.db"
  admin_key: "${ADMIN_KEY}"  # From environment!
  
  rate_limiting:
    enabled: true
    default_requests_per_minute: 30
    default_requests_per_hour: 500

logging:
  level: "info"
  format: "json"
  output: "file"
  file_path: "/var/log/ollama-proxy/proxy.log"
  max_size: 500
  max_backups: 10
  max_age: 30
  compress: true

tools:
  fallback_model: "llama3.1"
  optimizer:
    enabled: true
    simplify_system_message: true

metrics:
  enabled: true
  prometheus_path: "/metrics"

tui:
  enabled: true
  refresh_rate: "2s"

development:
  hot_reload: false
  debug_mode: false
  profile_enabled: false
  pprof_enabled: false
```

### Security Checklist

- [ ] Change `admin_key` from default
- [ ] Use environment variables for secrets
- [ ] Enable rate limiting
- [ ] Use `json` or `sqlite` storage (not in-memory)
- [ ] Set appropriate timeouts
- [ ] Enable structured logging (`json` format)
- [ ] Use reverse proxy (nginx/caddy) for TLS
- [ ] Restrict `server.host` to `127.0.0.1` if behind proxy
- [ ] Enable metrics for monitoring
- [ ] Regularly rotate log files

---

## Best Practices

### 1. Secrets Management

**❌ Don't:**
```yaml
auth:
  admin_key: "sk-admin-dev-key-12345"  # Hardcoded!
```

**✅ Do:**
```yaml
auth:
  admin_key: "${ADMIN_KEY}"  # From environment
```

```bash
export PROXY_AUTH_ADMIN_KEY="$(openssl rand -hex 32)"
```

### 2. Timeout Configuration

**Match timeouts to model size:**

```yaml
# Small models (< 10B)
ollama:
  timeout: "1m"
server:
  write_timeout: "1m"

# Large models (> 30B)
ollama:
  timeout: "5m"
server:
  write_timeout: "5m"
```

### 3. Rate Limiting

**Set conservative defaults, adjust per key:**

```yaml
auth:
  rate_limiting:
    default_requests_per_minute: 10  # Conservative
    default_requests_per_hour: 100
```

Then create keys with higher limits for trusted clients via TUI.

### 4. Logging

**Development:**
```yaml
logging:
  level: "debug"
  format: "text"
  output: "both"
```

**Production:**
```yaml
logging:
  level: "info"
  format: "json"
  output: "file"
```

**Staging:**
```yaml
logging:
  level: "debug"  # More verbose
  format: "json"
  output: "both"
```

### 5. Resource Limits

**Connection Pool:**
```yaml
# Low traffic
ollama:
  connection_pool_size: 10

# High traffic
ollama:
  connection_pool_size: 100
```

**Request Tracking:**
```yaml
# Limited memory
tui:
  request_monitor:
    max_requests: 100

# Ample memory
tui:
  request_monitor:
    max_requests: 10000
```

### 6. Configuration Validation

Always validate configuration after changes:

```bash
# Test configuration
./bin/server --validate-config

# Or start server and check logs
./bin/server
# Look for "Configuration loaded successfully"
```

### 7. Version Control

**Do commit:**
- `configs/dev.yaml`
- `configs/production.yaml.example`

**Don't commit:**
- `configs/production.yaml` (has secrets)
- `data/api_keys.json`
- `logs/`

**`.gitignore`:**
```gitignore
configs/production.yaml
data/api_keys.json
data/api_keys.db
logs/
```

---

## Additional Resources

- **[README.md](../README.md)** - Project overview
- **[TUI_GUIDE.md](TUI_GUIDE.md)** - Terminal UI guide
- **[API_DOCUMENTATION.md](API_DOCUMENTATION.md)** - API reference
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Problem solutions
- **[PERFORMANCE.md](PERFORMANCE.md)** - Performance tuning

---

**Questions?** [Open an issue on GitHub](https://github.com/yourusername/ollama-openai-proxy/issues)

