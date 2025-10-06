# 🏗️ Architecture Overview

Complete architecture documentation for Ollama-OpenAI Proxy.

## 📑 Table of Contents

- [System Overview](#system-overview)
- [High-Level Architecture](#high-level-architecture)
- [Component Details](#component-details)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Design Patterns](#design-patterns)
- [Security Architecture](#security-architecture)
- [Monitoring & Observability](#monitoring--observability)
- [Performance Considerations](#performance-considerations)
- [Extensibility](#extensibility)

---

## System Overview

Ollama-OpenAI Proxy is a **production-ready Go application** that provides an **OpenAI-compatible API** for local Ollama models. It acts as a bridge between OpenAI API clients and local Ollama server, enabling seamless integration with existing tools and workflows.

### Key Principles

1. **OpenAI Compatibility** - Exact API format matching
2. **Security First** - Enterprise-grade authentication and authorization
3. **High Performance** - Connection pooling, circuit breakers, caching
4. **100% Test Coverage** - Comprehensive testing across all components
5. **Observability** - Prometheus metrics, structured logging, TUI monitoring

---

## High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT APPLICATIONS                              │
│           (Zed IDE, Continue.dev, VSCode Copilot, Custom Apps)               │
└────────────────────────────────┬──────────────────────────────────────────────┘
                                 │ OpenAI API Format
                                 │ Authorization: Bearer sk-xxx
                                 ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          OLLAMA-OPENAI PROXY                                  │
│                                                                               │
│  ┌────────────────────────────────────────────────────────────────────────┐  │
│  │  HTTP Server (Gin Framework)                                            │  │
│  │  • /v1/chat/completions  • /v1/models  • /v1/embeddings                │  │
│  │  • /v1/completions  • /health  • /metrics  • /api/stats                │  │
│  └────────────────────────┬───────────────────────────────────────────────┘  │
│                           │                                                   │
│  ┌────────────────────────┴───────────────────────────────────────────────┐  │
│  │  Middleware Pipeline                                                    │  │
│  │  ┌───────────┐  ┌────────────┐  ┌───────┐  ┌──────────┐  ┌──────────┐│  │
│  │  │   Auth    │→ │    Rate    │→ │ Stats │→ │Prometheus│→ │ Logging  ││  │
│  │  │   Keys    │  │  Limiting  │  │       │  │ Metrics  │  │          ││  │
│  │  └───────────┘  └────────────┘  └───────┘  └──────────┘  └──────────┘│  │
│  └────────────────────────┬───────────────────────────────────────────────┘  │
│                           │                                                   │
│  ┌────────────────────────┴───────────────────────────────────────────────┐  │
│  │  Converters & Handlers                                                  │  │
│  │  • OpenAI ↔ Ollama Request/Response Conversion                         │  │
│  │  • Streaming SSE Handler (Server-Sent Events)                          │  │
│  │  • Tool Calls Conversion (Function Calling Support)                    │  │
│  │  • Prompt Optimizer (System Message Simplification)                    │  │
│  │  • Model Routing (Tool-Aware Switching)                                │  │
│  └────────────────────────┬───────────────────────────────────────────────┘  │
│                           │                                                   │
│  ┌────────────────────────┴───────────────────────────────────────────────┐  │
│  │  Ollama Client (HTTP)                                                   │  │
│  │  • Connection Pool (reusable HTTP connections)                          │  │
│  │  • Retry Logic (exponential backoff)                                    │  │
│  │  • Circuit Breaker (fault tolerance)                                    │  │
│  │  • Timeout Management                                                   │  │
│  └────────────────────────┬───────────────────────────────────────────────┘  │
│                           │                                                   │
└───────────────────────────┼───────────────────────────────────────────────────┘
                            │ Ollama API Format
                            │ /api/chat, /api/tags, /api/embed
                            ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                            OLLAMA SERVER                                      │
│              (llama3.1, qwen2.5-coder, mistral, etc.)                        │
└──────────────────────────────────────────────────────────────────────────────┘

                  ┌──────────────────────────────┐
                  │   TERMINAL UI (TUI)          │
                  │   ┌────────────────────────┐ │
                  │   │  Dashboard  │  Logs    │ │
                  │   │  API Keys   │  Control │ │
                  │   │  Models     │  Config  │ │
                  │   └────────────────────────┘ │
                  │   Polls: /api/stats          │
                  │          /api/config         │
                  │          /metrics            │
                  └──────────────────────────────┘
```

---

## Component Details

### 1. HTTP Server Layer

**Package:** `internal/api/router`

**Purpose:** HTTP request routing and server lifecycle management.

**Components:**
- **Router** (`router.go`) - Gin-based HTTP router
- **Middleware Setup** - Configures middleware pipeline
- **Route Groups** - Organizes endpoints by functionality

**Key Features:**
- TLS support (optional)
- Graceful shutdown
- Health check endpoints
- CORS configuration
- Timeout management

**Technologies:**
- [Gin](https://github.com/gin-gonic/gin) - HTTP framework
- net/http - Standard library HTTP server

---

### 2. Middleware Pipeline

**Package:** `internal/api/middleware`

**Purpose:** Request/response processing pipeline.

**Middlewares (in order):**

1. **Error Handling** (`error_handling.go`)
   - Catches panics from downstream handlers
   - Formats errors into OpenAI-compatible responses
   - Logs error details

2. **Panic Recovery** (`recovery.go`)
   - Recovers from panics
   - Logs stack traces
   - Returns 500 Internal Server Error

3. **Prometheus Metrics** (`metrics.go`)
   - Records HTTP request metrics
   - Tracks request duration, status codes, response sizes
   - Updates Prometheus counters/histograms

4. **Statistics** (`stats.go`)
   - Tracks global request counts
   - Records success/error rates
   - Calculates average durations

5. **Request Logging** (`logging.go`)
   - Structured logging with logrus
   - Logs method, path, status, duration, client IP
   - Configurable skip paths (health checks)

6. **CORS** (`cors.go`)
   - Handles Cross-Origin Resource Sharing
   - Configurable allowed origins, methods, headers

7. **Authentication** (`auth_keys.go`)
   - Validates API keys from Authorization header
   - Checks key permissions (model access)
   - Injects key info into request context

8. **Rate Limiting** (`../auth/ratelimit/middleware.go`)
   - Per-key rate limiting
   - Token bucket algorithm
   - Returns 429 when exceeded
   - Bypasses admin keys

---

### 3. API Handlers

**Package:** `internal/api/handlers`

**Components:**

#### Chat Completions (`chat.go`)
- Handles `/v1/chat/completions`
- Supports streaming (SSE)
- Function calling (tools)
- Converts OpenAI → Ollama format
- Handles tool-aware model routing

#### Models (`models.go`)
- Handles `/v1/models`
- Lists available Ollama models
- Returns OpenAI-compatible format
- Implements caching

#### Embeddings (`embeddings.go`)
- Handles `/v1/embeddings`
- Supports batch embeddings
- Converts OpenAI → Ollama format
- Returns proper usage counts

#### Legacy Completions (`completions.go`)
- Handles `/v1/completions`
- Text completions (non-chat)
- Converts to chat format internally
- Supports streaming

#### Statistics (`stats.go`)
- Handles `/api/stats`
- Returns server statistics
- Used by TUI for monitoring

#### Configuration (`config.go`)
- Handles `/api/config`
- Returns server configuration
- Excludes sensitive data

---

### 4. Request/Response Converters

**Package:** `internal/converter`

**Purpose:** Convert between OpenAI and Ollama formats.

**Components:**

#### Request Converter (`request.go`)
- `ConvertChatRequest()` - OpenAI → Ollama chat
- `ConvertCompletionRequest()` - OpenAI → Ollama completion
- `ConvertEmbeddingsRequest()` - OpenAI → Ollama embeddings
- Maps parameters (temperature, top_p, stop, etc.)
- Handles tool definitions conversion

#### Response Converter (`response.go`)
- `ConvertChatResponse()` - Ollama → OpenAI chat
- `ConvertEmbeddingsResponse()` - Ollama → OpenAI embeddings
- `ConvertCompletionResponse()` - Ollama → OpenAI completion
- Calculates token counts (approximation)
- Formats timestamps

#### Streaming Converter (`streaming.go`)
- `StreamConverter` - Stateful streaming converter
- Converts Ollama streaming chunks to OpenAI SSE format
- **Critical:** Maintains state for tool calls across chunks
- Handles `finish_reason` correctly for tools
- Must be reset between requests

**Key Implementation Details:**

```go
// Tool calls conversion is critical for IDE integration
type StreamConverter struct {
    hadToolCalls bool  // Remember if we've seen tool calls
}

// Ollama returns tool_calls in TWO places:
// 1. message.tool_calls (main location)
// 2. response.tool_calls (fallback)

// OpenAI streaming requires:
// - "index" field in each tool_call (no omitempty!)
// - finish_reason: "tool_calls" if tools were invoked
```

---

### 5. Ollama Client

**Package:** `internal/client/ollama`

**Purpose:** HTTP client for Ollama API.

**Components:**

#### Client (`client.go`)
- HTTP client with connection pooling
- Custom transport configuration
- Timeout management
- Retry logic with exponential backoff

#### Methods (`methods.go`)
- `GetModels()` - Fetch available models
- `ChatCompletion()` - Non-streaming chat
- `ChatCompletionStream()` - Streaming chat
- `Embed()` - Batch embeddings
- `Embeddings()` - Legacy single embedding
- Records Prometheus metrics

#### Streaming (`streaming.go`)
- `StreamChatCompletion()` - SSE streaming handler
- Buffers partial JSON
- Handles connection errors
- Implements backpressure

**Connection Pool Configuration:**
```go
&http.Transport{
    MaxIdleConns:       100,  // Total idle connections
    MaxIdleConnsPerHost: 10,  // Per-host idle connections
    IdleConnTimeout:    90 * time.Second,
}
```

---

### 6. Authentication & Authorization

**Package:** `internal/auth`

#### API Key Manager (`apikey/manager.go`)
- CRUD operations for API keys
- Bcrypt hashing for storage
- Model permissions checking
- Usage tracking

#### Storage (`apikey/storage.go`)
- JSON file storage (simple)
- SQLite storage (production)
- Concurrent access safe

#### Rate Limiting (`ratelimit/middleware.go`)
- Token bucket algorithm
- Per-key rate limits
- Sliding window for hours
- Redis support (optional)

**API Key Structure:**
```go
type APIKey struct {
    ID          string
    Name        string
    KeyHash     string  // Bcrypt hash
    Enabled     bool
    CreatedAt   time.Time
    LastUsedAt  *time.Time
    Permissions []string  // Model names or "*"
    RateLimit   RateLimitConfig
}
```

---

### 7. Prompt Optimizer

**Package:** `internal/optimizer`

**Purpose:** Optimize complex system messages for local models.

**Features:**
- Simplifies verbose IDE system messages (80% reduction)
- Preserves critical tool usage instructions
- Removes redundant formatting examples
- Configurable preservation rules

**Use Case:**
- IDEs (Zed, Cursor, Continue.dev) send 6000+ char system messages
- Local models perform better with concise prompts
- Optimizer reduces to ~1200 chars while keeping essentials

**Configuration:**
```yaml
tools:
  optimizer:
    enabled: true
    simplify_system_message: true
    preserve_instructions:
      - "When user asks to find files, use find_path tool."
      - "When user asks to read a file, use read_file tool."
```

---

### 8. Metrics Collection

**Package:** `internal/metrics`

**Purpose:** Prometheus metrics collection.

**Metrics Categories:**

1. **HTTP Metrics**
   - `ollama_proxy_http_requests_total` - Total requests
   - `ollama_proxy_http_request_duration_seconds` - Latency histogram
   - `ollama_proxy_http_requests_in_flight` - Active requests gauge
   - `ollama_proxy_http_response_size_bytes` - Response sizes

2. **Ollama Metrics**
   - `ollama_proxy_ollama_requests_total` - Ollama API calls
   - `ollama_proxy_ollama_request_duration_seconds` - Ollama latency
   - `ollama_proxy_ollama_errors_total` - Ollama errors

3. **API Key Metrics**
   - `ollama_proxy_api_key_requests_total` - Per-key requests
   - `ollama_proxy_api_key_rate_limit_exceeded_total` - Rate limit violations
   - `ollama_proxy_api_keys_active_total` - Active keys count
   - `ollama_proxy_api_key_tokens_used_total` - Token usage

**Implementation:**
```go
// Singleton pattern
var DefaultMetrics *MetricsManager

// Initialize on server start
metrics.Init("ollama_proxy")

// Record in handlers/middleware
metrics.DefaultMetrics.RecordOllamaRequest(model, operation, status, duration)
```

---

### 9. Terminal UI (TUI)

**Package:** `cmd/tui`

**Purpose:** Terminal-based monitoring and management.

**Architecture:**
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework
- MVC pattern (Model-View-Controller)
- Event-driven updates
- HTTP polling for data

**Components:**

1. **Main Model** (`main.go`)
   - Application state
   - Screen management
   - Input handling
   - Update loop

2. **Metrics Parser** (`metrics_parser.go`)
   - Parses Prometheus text format
   - Extracts key metrics
   - Calculates aggregates

3. **Logs Reader** (`logs_reader.go`)
   - Reads log file
   - Detects log levels
   - Color coding

4. **Scroll Helper** (`scroll_helper.go`)
   - Handles content scrolling
   - Viewport management
   - Scroll indicators

**Data Flow:**
```
TUI Init
  ↓
Fetch /api/stats → Display Dashboard
  ↓
Fetch /api/config → Display Config
  ↓
Fetch /metrics → Parse & Display
  ↓
Every 1s: Repeat
```

**Screens:**
1. Dashboard - Overview & metrics
2. Requests - Request monitoring (placeholder)
3. Models - Available models
4. API Keys - Key management
5. Config - Configuration viewer
6. Logs - Log viewer with color coding
7. Control - Server status & commands

---

## Data Flow

### Chat Completion Request Flow

```
1. Client sends POST /v1/chat/completions
   ↓
2. Middleware Pipeline:
   - Error Handling wraps the pipeline
   - Panic Recovery catches crashes
   - Prometheus Metrics starts timer
   - Stats Middleware increments counters
   - Logging Middleware logs request
   - Auth Middleware validates API key
   - Rate Limit Middleware checks limits
   ↓
3. Chat Handler (chat.go):
   - Validates request body
   - Checks model permissions
   - Tool-aware routing (if tools present)
   - Prompt optimization (if enabled)
   ↓
4. Request Converter:
   - Converts OpenAI format → Ollama format
   - Maps parameters
   - Converts tool definitions
   ↓
5. Ollama Client:
   - Sends request to Ollama
   - Handles retry logic
   - Manages timeouts
   ↓
6. Ollama Server:
   - Loads model (if needed)
   - Generates response
   - Returns result
   ↓
7. Response Converter:
   - Converts Ollama format → OpenAI format
   - Extracts tool calls
   - Calculates tokens
   ↓
8. Middleware Pipeline (reverse):
   - Prometheus Metrics records duration
   - Stats Middleware updates counters
   - Logging Middleware logs response
   ↓
9. Client receives response
```

### Streaming Request Flow

**Differences from non-streaming:**

- Step 5: `ChatCompletionStream()` instead of `ChatCompletion()`
- Step 7: `StreamConverter` processes chunks incrementally
- Response: SSE format (`data: {...}\n\ndata: [DONE]\n\n`)

**Streaming Converter State:**
```go
converter := &StreamConverter{}  // Create per-request

for chunk := range stream {
    sseChunk := converter.ConvertChunk(chunk)
    // Write: "data: {...}\n\n"
}

// Must call converter.Reset() before next request!
```

---

## Technology Stack

### Core Technologies

| Technology | Purpose | Version |
|------------|---------|---------|
| **Go** | Programming language | 1.25+ |
| **Gin** | HTTP framework | Latest |
| **Viper** | Configuration management | Latest |
| **Logrus** | Structured logging | Latest |
| **Bubble Tea** | Terminal UI framework | Latest |
| **Prometheus** | Metrics collection | Client Go |

### Libraries

#### HTTP & Networking
- `github.com/gin-gonic/gin` - HTTP routing
- `net/http` - HTTP client/server
- `nhooyr.io/websocket` - WebSocket support (future)

#### Authentication & Security
- `golang.org/x/crypto/bcrypt` - Password hashing
- `github.com/golang-jwt/jwt/v5` - JWT tokens

#### Storage
- `encoding/json` - JSON storage
- `database/sql` + `modernc.org/sqlite` - SQLite storage

#### Configuration
- `github.com/spf13/viper` - Config loading
- `gopkg.in/yaml.v3` - YAML parsing

#### Logging
- `github.com/sirupsen/logrus` - Structured logging
- `gopkg.in/natefinch/lumberjack.v2` - Log rotation

#### Metrics
- `github.com/prometheus/client_golang` - Prometheus client

#### Terminal UI
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/atotto/clipboard` - Clipboard operations

#### Testing
- `github.com/stretchr/testify` - Test assertions & mocks
- `github.com/stretchr/testify/mock` - Mocking
- `net/http/httptest` - HTTP testing

---

## Design Patterns

### 1. Middleware Pattern

**Purpose:** Chain of responsibility for request processing.

**Implementation:**
```go
func RequestLogging(config LoggingConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()  // Call next middleware
        
        duration := time.Since(start)
        logger.Info("request processed", 
            "method", c.Request.Method,
            "path", c.Request.URL.Path,
            "duration", duration)
    }
}
```

### 2. Singleton Pattern

**Purpose:** Single instance of metrics manager.

**Implementation:**
```go
var DefaultMetrics *MetricsManager

func Init(namespace string) {
    DefaultMetrics = NewMetricsManager(namespace)
}
```

### 3. Factory Pattern

**Purpose:** Create configured clients.

**Implementation:**
```go
func NewOllamaClient(config OllamaConfig) *Client {
    return &Client{
        baseURL: config.URL,
        client: &http.Client{
            Timeout: config.Timeout,
            Transport: createTransport(config),
        },
    }
}
```

### 4. Strategy Pattern

**Purpose:** Different storage backends.

**Implementation:**
```go
type Storage interface {
    Save(key *APIKey) error
    Get(id string) (*APIKey, error)
    List() ([]*APIKey, error)
    Delete(id string) error
}

type JSONStorage struct { /* ... */ }
type SQLiteStorage struct { /* ... */ }
```

### 5. Observer Pattern

**Purpose:** Metrics collection.

**Implementation:**
```go
// Middleware observes requests
func PrometheusMetrics() gin.HandlerFunc {
    return func(c *gin.Context) {
        metrics.HTTPRequestsInFlight.Inc()
        defer metrics.HTTPRequestsInFlight.Dec()
        
        c.Next()
        
        metrics.HTTPRequestsTotal.Inc()
    }
}
```

### 6. Adapter Pattern

**Purpose:** Convert between API formats.

**Implementation:**
```go
// Adapter from OpenAI to Ollama
func ConvertChatRequest(openai OpenAIRequest) OllamaRequest {
    return OllamaRequest{
        Model:  openai.Model,
        Messages: convertMessages(openai.Messages),
        Options: convertOptions(openai),
    }
}
```

---

## Security Architecture

### Authentication Flow

```
1. Client includes API key:
   Authorization: Bearer sk-1234567890-abcdef-xyz
   ↓
2. Auth Middleware extracts key
   ↓
3. Key Manager validates:
   - Key exists?
   - Key enabled?
   - Key hash matches?
   ↓
4. Permission check:
   - Model in allowed list?
   - Or "*" permission?
   ↓
5. Rate limit check:
   - Requests per minute OK?
   - Requests per hour OK?
   ↓
6. Inject key info into context
   ↓
7. Continue to handler
```

### Security Features

1. **API Key Hashing**
   - Bcrypt with cost 10
   - Plaintext keys never stored
   - Only shown once at creation

2. **Rate Limiting**
   - Per-key limits
   - Token bucket algorithm
   - Sliding window for hourly limits

3. **Permission Model**
   - Model-based access control
   - Wildcard (`*`) for all models
   - Per-key permissions

4. **Admin Key**
   - Bootstrap admin key from config
   - Bypasses rate limits
   - Full model access

5. **Secure Storage**
   - JSON with file permissions
   - SQLite with encryption (optional)
   - No plain-text secrets in config

---

## Monitoring & Observability

### Logging Strategy

**Structured Logging:**
```go
logger.WithFields(logrus.Fields{
    "method": "POST",
    "path": "/v1/chat/completions",
    "model": "llama3.1",
    "duration": "2.5s",
    "status": 200,
}).Info("request completed")
```

**Log Levels:**
- **DEBUG** - Detailed diagnostics
- **INFO** - General informational
- **WARN** - Warning conditions
- **ERROR** - Error conditions

**Log Rotation:**
- Max size: 100MB (configurable)
- Max backups: 5 (configurable)
- Max age: 30 days (configurable)
- Compression: gzip

### Metrics Collection

**Metrics Types:**
- **Counter** - Monotonically increasing (total requests)
- **Gauge** - Can go up/down (active connections)
- **Histogram** - Distribution (request duration)

**Collection Points:**
- HTTP middleware (all requests)
- Ollama client (all Ollama calls)
- Rate limiter (limit violations)
- Auth middleware (token usage)

### TUI Monitoring

**Real-time displays:**
- Server status
- Request statistics
- Error rates
- API key usage
- Model availability
- Log stream

**Update frequency:** 1 second (configurable)

---

## Performance Considerations

### Connection Pooling

```go
// Reuse HTTP connections to Ollama
&http.Transport{
    MaxIdleConns:       100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:    90 * time.Second,
}
```

### Caching

**Model List Cache:**
- TTL: 5 minutes
- Background refresh: 1 minute
- Reduces load on Ollama

### Timeout Management

**Hierarchical timeouts:**
```
Request Context Timeout (5m)
  ↓
Ollama Client Timeout (5m)
  ↓
HTTP Transport Timeout (5m)
```

### Concurrency

**Safe concurrent access:**
- Mutex for API key storage
- Channel-based metrics buffering
- Context cancellation propagation

---

## Extensibility

### Adding New Endpoints

1. Create handler in `internal/api/handlers/`
2. Add route in `internal/api/router/router.go`
3. Add tests in `*_test.go`

### Adding New Storage Backend

1. Implement `Storage` interface
2. Add factory method in `storage.go`
3. Add config option in `config.yaml`

### Adding New Middleware

1. Create middleware in `internal/api/middleware/`
2. Add to pipeline in `router/router.go`
3. Configure order appropriately

### Adding New Metrics

1. Register metric in `internal/metrics/metrics.go`
2. Record in appropriate component
3. Document in metrics list

---

## Additional Resources

- **[README.md](../README.md)** - Project overview
- **[API_DOCUMENTATION.md](API_DOCUMENTATION.md)** - API reference
- **[CONFIGURATION.md](CONFIGURATION.md)** - Configuration guide
- **[PERFORMANCE.md](PERFORMANCE.md)** - Performance tuning
- **[Plan.md](../Plan.md)** - Development roadmap

---

**Questions about architecture?** [Open a discussion on GitHub](https://github.com/yourusername/ollama-openai-proxy/discussions)

