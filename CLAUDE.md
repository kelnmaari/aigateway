# AIGateway - Claude Code Guidelines

**AIGateway** — Go 1.25 приложение, предоставляющее OpenAI-совместимый API Gateway с multi-provider backend.

**Текущая версия:** 4.8.8
**Последняя миграция:** 138

---

## Project Structure

### Entry Points
- `cmd/server/main.go` - HTTP/WebSocket server (основной)
- `cmd/tui/main.go` - Terminal UI для мониторинга
- `cmd/webui/main.go` - Web UI

### Core Directories

```
internal/
├── api/                    # HTTP handlers & middleware
│   ├── handlers/           # API endpoints
│   └── middleware/         # Auth, rate limit, logging
├── auth/                   # Authentication (JWT, LDAP, OIDC, API Keys)
├── cache/                  # API key cache layer
├── config/                 # Configuration system
├── inference/              # Docker-based inference (vLLM, SGLang, TGI, llama.cpp)
├── providers/              # External model providers (Ollama, vLLM endpoints)
├── storage/postgresql/     # PostgreSQL database layer
├── gitlab/                 # GitLab integration (RAG, Test Gen, Code Review)
├── rag/                    # RAG pipeline (embeddings, Qdrant)
├── services/               # Business logic services
├── models/                 # Domain models
├── metrics/                # Prometheus metrics
└── web/                    # Embedded WebUI
```

---

## Critical Rules

### НИКОГДА НЕ ЗАПУСКАЙ СЕРВЕР

```bash
# ✅ РАЗРЕШЕНО
go build -o bin/server.exe cmd/server/main.go
go test ./...
go test -race ./...

# ❌ ЗАПРЕЩЕНО
go run cmd/server/main.go
./bin/server.exe
```

Если нужно проверить работу - сообщи пользователю и дай инструкции.

### PostgreSQL Only

- Все миграции в `internal/storage/postgresql/migrations/`
- Используй `$1, $2` placeholders (не `?`)
- `ON CONFLICT ... DO UPDATE` для upsert
- UUID для primary keys

### Миграции НЕИЗМЕНЯЕМЫ

**После создания миграции её содержимое НИКОГДА не редактируется.**

Исключение: миграция не выполняется из-за синтаксической ошибки.

---

## Changelog & Migration Workflow

### При каждом релизе выполни:

#### 1. Обнови VERSION файл

```bash
echo "4.X.Y" > VERSION
```

#### 2. Обнови CHANGELOG.md

```markdown
## [4.X.Y] - YYYY-MM-DD

### Added
- **Feature**: Description

### Changed
- **Changes**: What changed

### Fixed
- **Fixes**: What was fixed

### Technical
- Technical details
```

#### 3. Определи номер следующей миграции

```bash
ls internal/storage/postgresql/migrations/*.up.sql | tail -1
# 138_add_changelog_v4_8_8.up.sql → следующий: 139
```

#### 4. Создай UP файл

`internal/storage/postgresql/migrations/139_add_changelog_v4_8_9.up.sql`:

```sql
INSERT INTO changelogs (version, release_date, content) VALUES
('4.8.9', 'YYYY-MM-DD', '## [4.8.9] - YYYY-MM-DD

### Added
- **Feature**: Description

### Technical
- Details')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;
```

#### 5. Создай DOWN файл

`internal/storage/postgresql/migrations/139_add_changelog_v4_8_9.down.sql`:

```sql
DELETE FROM changelogs WHERE version = '4.8.9';
```

#### 6. Проверь компиляцию

```bash
go build -o bin/server.exe cmd/server/main.go
```

#### 7. Commit все файлы вместе

```
VERSION + CHANGELOG.md + миграции
```

---

## OpenAI API Compatibility

### Core Endpoints
- `/v1/chat/completions` - Chat с streaming
- `/v1/models` - Список моделей
- `/v1/embeddings` - Text embeddings

### Request/Response Format

```go
type ChatCompletionRequest struct {
    Model       string        `json:"model" binding:"required"`
    Messages    []ChatMessage `json:"messages" binding:"required"`
    MaxTokens   *int          `json:"max_tokens,omitempty"`
    Temperature *float64      `json:"temperature,omitempty"`
    Stream      *bool         `json:"stream,omitempty"`
    Tools       []Tool        `json:"tools,omitempty"`
}

type ChatCompletionResponse struct {
    ID      string                 `json:"id"`
    Object  string                 `json:"object"`
    Created int64                  `json:"created"`
    Model   string                 `json:"model"`
    Choices []ChatCompletionChoice `json:"choices"`
    Usage   ChatCompletionUsage    `json:"usage"`
}
```

### Streaming (SSE)

```go
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")

// Chunks
fmt.Fprintf(w, "data: %s\n\n", toJSON(chunk))
flusher.Flush()

// End
fmt.Fprintf(w, "data: [DONE]\n\n")
```

---

## Inference System

### Docker-based Providers

| Provider | Docker Image | Use Case |
|----------|--------------|----------|
| vLLM | `vllm/vllm-openai` | Production inference |
| SGLang | `lmsysorg/sglang` | Fast inference |
| TGI | `ghcr.io/huggingface/text-generation-inference` | HuggingFace models |
| TEI | `ghcr.io/huggingface/text-embeddings-inference` | Embeddings |
| llama.cpp | `ghcr.io/ggerganov/llama.cpp` | GGUF models |

### Configuration

```yaml
inference:
  docker:
    enabled: true
    hf_cache_dir: "./data/models/hf"
    gguf_dir: "./data/models/gguf"
    max_running_models: 2
    default_provider: "vllm"
```

---

## Go Development Standards

### Error Handling

```go
// Wrapped errors with context
if err := s.validateRequest(req); err != nil {
    return fmt.Errorf("request validation failed: %w", err)
}

// Custom error types
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}
```

### Context Patterns

```go
func (p *Provider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, p.timeout)
    defer cancel()

    return p.makeRequest(ctx, req)
}
```

### WaitGroup Pattern

```go
func processRequests(ctx context.Context, requests []Request) error {
    var wg sync.WaitGroup
    errChan := make(chan error, len(requests))

    for _, req := range requests {
        wg.Add(1) // Add before goroutine
        go func(r Request) {
            defer wg.Done()
            if err := processRequest(ctx, r); err != nil {
                errChan <- err
            }
        }(req)
    }

    wg.Wait()
    close(errChan)

    return errors.Join(collectErrors(errChan)...)
}
```

### CPU Cache-Friendly Structures

При высоком contention используй padding:

```go
type Stats struct {
    TotalRequests int64
    _pad1         [56]byte  // 64 - 8 = 56 bytes

    ActiveRequests int64
    _pad2          [56]byte
}
```

---

## Configuration

### Environment Prefix: `AIGATEWAY`

```go
v.SetEnvPrefix("AIGATEWAY")
v.AutomaticEnv()
v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
```

### Config Files
- `configs/config.yaml` - основной
- `configs/dev.yaml` - development

---

## Testing

### Commands

```bash
go test ./...                    # Все тесты
go test -race ./...              # Race detection
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total
```

### Naming Convention

```go
// TestFunctionName_Scenario_ExpectedResult
func TestChatHandler_ValidRequest_ReturnsSuccess(t *testing.T) {}
func TestChatHandler_InvalidModel_ReturnsError(t *testing.T) {}
```

### Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name      string
        input     Config
        wantError bool
    }{
        {"valid config", Config{Port: 8080}, false},
        {"invalid port", Config{Port: 0}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Validate()
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## Security

### Never Log Sensitive Data

```go
// ✅ Correct
l.WithField("key_id", key.ID).Info("API key created")

// ❌ Wrong - never log actual keys
l.WithField("key", plainKey).Info("API key created")
```

### API Key Extraction

```go
func extractAPIKey(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return r.Header.Get("X-API-Key")
}
```

### Input Validation

```go
func validateChatRequest(req ChatCompletionRequest) error {
    if req.Model == "" {
        return ValidationError{Field: "model", Message: "required"}
    }
    if len(req.Messages) == 0 {
        return ValidationError{Field: "messages", Message: "at least one required"}
    }
    return nil
}
```

---

## Prohibited Actions

- ❌ НЕ запускать сервер автоматически
- ❌ НЕ удалять базу данных
- ❌ НЕ изменять существующие миграции
- ❌ НЕ пропускать создание миграции при релизе
- ❌ НЕ использовать psql CLI для ручных INSERT
- ❌ НЕ логировать API ключи и пароли
