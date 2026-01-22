# Failsafe-Go Analysis

> Анализ библиотеки [failsafe-go](https://github.com/failsafe-go/failsafe-go) для улучшения отказоустойчивости.

**Репозиторий:** https://github.com/failsafe-go/failsafe-go  
**Документация:** https://failsafe-go.dev  
**Версия:** v0.9.4  
**Лицензия:** MIT  
**Stars:** 2.1k

## Обзор

Failsafe-go — библиотека для построения отказоустойчивых приложений на Go. Предоставляет composable policies:

| Категория | Политики |
|-----------|----------|
| **Failure Handling** | Retry, Fallback |
| **Load Limiting** | Circuit Breaker, Adaptive Limiter, Adaptive Throttler, Bulkhead, Rate Limiter, Cache |
| **Time Limiting** | Timeout, Hedge |

## Установка

```bash
go get github.com/failsafe-go/failsafe-go
```

## Ключевые паттерны

### 1. Retry Policy

```go
import (
    "github.com/failsafe-go/failsafe-go"
    "github.com/failsafe-go/failsafe-go/retrypolicy"
)

// Retry с exponential backoff
retryPolicy := retrypolicy.Builder[*http.Response]().
    HandleErrors(ErrConnectionFailed).
    WithDelay(time.Second).
    WithMaxRetries(3).
    WithBackoff(time.Second, 30*time.Second).
    Build()

response, err := failsafe.Get(func() (*http.Response, error) {
    return client.Do(req)
}, retryPolicy)
```

### 2. Circuit Breaker

```go
import "github.com/failsafe-go/failsafe-go/circuitbreaker"

// Circuit breaker: открывается после 5 failures за 10 секунд
cb := circuitbreaker.Builder[any]().
    WithFailureThresholdRatio(5, 10).
    WithDelay(30 * time.Second).
    Build()

result, err := failsafe.Get(func() (any, error) {
    return callExternalService()
}, cb)
```

### 3. Timeout

```go
import "github.com/failsafe-go/failsafe-go/timeout"

timeoutPolicy := timeout.With[*Response](10 * time.Second)

response, err := failsafe.Get(func() (*Response, error) {
    return longRunningOperation()
}, timeoutPolicy)
```

### 4. Fallback

```go
import "github.com/failsafe-go/failsafe-go/fallback"

fb := fallback.WithResult(defaultResponse)
// или
fb := fallback.WithFunc(func(exec failsafe.Execution[*Response]) (*Response, error) {
    return getCachedResponse()
})
```

### 5. Hedge Policy (параллельные запросы)

```go
import "github.com/failsafe-go/failsafe-go/hedgepolicy"

// Запустить hedge request через 1 секунду если первый не ответил
hedge := hedgepolicy.Builder[*Response]().
    WithDelay(time.Second).
    Build()

// Возвращает первый успешный результат
response, err := failsafe.Get(func() (*Response, error) {
    return callLLM()
}, hedge)
```

### 6. Rate Limiter

```go
import "github.com/failsafe-go/failsafe-go/ratelimiter"

// 100 requests per second с bursty
rl := ratelimiter.BurstyBuilder[any](100, time.Second).Build()

// Smooth rate limiting
rl := ratelimiter.SmoothBuilder[any](100, time.Second).Build()
```

### 7. Bulkhead (ограничение concurrency)

```go
import "github.com/failsafe-go/failsafe-go/bulkhead"

// Максимум 10 concurrent executions
bh := bulkhead.With[any](10)
```

### 8. Композиция политик

```go
// Порядок: внешняя → внутренняя
// Fallback → Retry → CircuitBreaker → Timeout → function
result, err := failsafe.Get(func() (*Response, error) {
    return callService()
}, fallbackPolicy, retryPolicy, circuitBreaker, timeoutPolicy)
```

## Применение в AIGateway

### 1. Ollama Client (высокий приоритет)

**Текущие проблемы:**
- Ollama может быть недоступен
- Модель может долго грузиться в память
- Timeout при inference

**Решение с failsafe-go:**

```go
// internal/ollama/client.go
type ResilientClient struct {
    client   *http.Client
    baseURL  string
    
    retryPolicy     retrypolicy.RetryPolicy[*OllamaResponse]
    circuitBreaker  circuitbreaker.CircuitBreaker[*OllamaResponse]
    timeoutPolicy   timeout.Timeout[*OllamaResponse]
}

func NewResilientClient(baseURL string) *ResilientClient {
    return &ResilientClient{
        baseURL: baseURL,
        client:  &http.Client{},
        
        // Retry: 3 попытки с exponential backoff
        retryPolicy: retrypolicy.Builder[*OllamaResponse]().
            HandleErrors(ErrOllamaUnavailable, ErrModelLoading).
            WithBackoff(time.Second, 30*time.Second).
            WithMaxRetries(3).
            Build(),
        
        // Circuit Breaker: открывается после 5 failures за минуту
        circuitBreaker: circuitbreaker.Builder[*OllamaResponse]().
            WithFailureThresholdRatio(5, 60).
            WithDelay(30 * time.Second).
            OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
                log.WithField("state", event.NewState).Warn("Ollama circuit breaker state changed")
            }).
            Build(),
        
        // Timeout: 2 минуты для inference
        timeoutPolicy: timeout.With[*OllamaResponse](2 * time.Minute),
    }
}

func (c *ResilientClient) ChatCompletion(ctx context.Context, req *ChatRequest) (*OllamaResponse, error) {
    return failsafe.NewExecutor[*OllamaResponse](
        c.retryPolicy,
        c.circuitBreaker,
        c.timeoutPolicy,
    ).WithContext(ctx).Get(func() (*OllamaResponse, error) {
        return c.doRequest(req)
    })
}
```

### 2. GitLab API Client

**Текущие проблемы:**
- Rate limiting от GitLab
- Network timeouts
- 502/503 errors

**Решение:**

```go
// internal/gitlab/client.go
gitlabRetry := retrypolicy.Builder[*gitlab.Response]().
    HandleIf(func(resp *gitlab.Response, err error) bool {
        if err != nil {
            return true
        }
        // Retry on 429, 502, 503, 504
        return resp.StatusCode == 429 || resp.StatusCode >= 502
    }).
    WithBackoff(time.Second, time.Minute).
    WithMaxRetries(5).
    Build()
```

### 3. Qdrant Vector Store

**Текущие проблемы:**
- Connection timeouts
- Bulk upsert failures

**Решение:**

```go
qdrantRetry := retrypolicy.Builder[*QdrantResponse]().
    HandleErrors(ErrQdrantConnection, ErrQdrantTimeout).
    WithDelay(500 * time.Millisecond).
    WithMaxRetries(3).
    Build()

qdrantBulkhead := bulkhead.With[*QdrantResponse](20) // Max 20 concurrent upserts
```

### 4. LLM Inference с Hedge Policy

**Сценарий:** Есть несколько LLM провайдеров, хотим минимальную latency.

```go
// Если первый LLM не ответил за 5 сек — запускаем параллельный запрос к другому
hedgePolicy := hedgepolicy.Builder[*LLMResponse]().
    WithDelay(5 * time.Second).
    Build()

// Первый успешный ответ возвращается
response, err := failsafe.Get(func() (*LLMResponse, error) {
    return callPrimaryLLM(prompt)
}, hedgePolicy)
```

### 5. Замена текущего Rate Limiter

**Текущая реализация:** `internal/services/ratelimit/sliding_window.go`

**Можно заменить на failsafe-go:**

```go
// Smooth rate limiting — равномерное распределение
rl := ratelimiter.SmoothBuilder[any](
    cfg.RequestsPerMinute,
    time.Minute,
).Build()

// В middleware
_, err := failsafe.Run(func() error {
    next.ServeHTTP(w, r)
    return nil
}, rl)

if err != nil {
    // Rate limit exceeded
    http.Error(w, "Too Many Requests", 429)
}
```

## Сравнение с текущей реализацией

| Функция | Текущая реализация | failsafe-go |
|---------|-------------------|-------------|
| Rate Limiting | Своя (sliding window) | `ratelimiter.SmoothBuilder` |
| Retry | Простой loop | `retrypolicy` с backoff |
| Circuit Breaker | Нет | `circuitbreaker` |
| Timeout | `context.WithTimeout` | `timeout` (composable) |
| Fallback | `if err != nil { return default }` | `fallback` (composable) |
| Bulkhead | Semaphore | `bulkhead` |
| Hedge | Нет | `hedgepolicy` |

## План интеграции

### Phase 1 (v4.3.x): Ollama Client

```
internal/ollama/
├── client.go           # Existing
├── resilient_client.go # NEW: failsafe-go wrapper
└── policies.go         # NEW: configurable policies
```

- [ ] Обернуть Ollama client в failsafe-go
- [ ] Добавить Circuit Breaker для Ollama
- [ ] Retry с exponential backoff
- [ ] Timeout policy (configurable)
- [ ] Metrics/Events logging

### Phase 2 (v4.4.x): External APIs

- [ ] GitLab API client resilience
- [ ] Qdrant client resilience
- [ ] Redis client resilience

### Phase 3 (v4.5.x): Advanced Features

- [ ] Hedge policy для multi-LLM
- [ ] Adaptive limiter для auto-tuning
- [ ] Cache policy для inference results

## Конфигурация

```yaml
# configs/dev.yaml
resilience:
  ollama:
    retry:
      max_attempts: 3
      initial_delay: 1s
      max_delay: 30s
      backoff_multiplier: 2
    circuit_breaker:
      failure_threshold: 5
      failure_window: 60s
      reset_timeout: 30s
    timeout: 120s
    
  gitlab:
    retry:
      max_attempts: 5
      initial_delay: 1s
      max_delay: 60s
    timeout: 30s
    
  qdrant:
    retry:
      max_attempts: 3
      initial_delay: 500ms
    bulkhead:
      max_concurrent: 20
    timeout: 10s
```

## HTTP/gRPC Middleware

```go
import "github.com/failsafe-go/failsafe-go/failsafehttp"

// HTTP client with retry
client := failsafehttp.NewClient(http.DefaultClient, retryPolicy, circuitBreaker)

// HTTP handler wrapper
handler := failsafehttp.NewHandler(myHandler, bulkhead, ratelimiter)
```

## Мониторинг

```go
circuitBreaker := circuitbreaker.Builder[any]().
    OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
        metrics.CircuitBreakerState.WithLabelValues("ollama").Set(float64(event.NewState))
        logger.WithFields(logrus.Fields{
            "service":   "ollama",
            "old_state": event.OldState,
            "new_state": event.NewState,
        }).Warn("Circuit breaker state changed")
    }).
    Build()

retryPolicy := retrypolicy.Builder[any]().
    OnRetry(func(event failsafe.ExecutionAttemptedEvent[any]) {
        metrics.RetryCount.WithLabelValues("ollama").Inc()
    }).
    Build()
```

## Риски

1. **Дополнительная зависимость** — но MIT лицензия, активная разработка
2. **Learning curve** — новые паттерны для команды
3. **Дублирование** — частично дублирует наш rate limiter

## Рекомендация

**Высокий приоритет** для интеграции.

Основные gains:
1. **Circuit Breaker** — критично для Ollama (сейчас отсутствует)
2. **Composable policies** — чистый код вместо nested if/retry loops
3. **Hedge Policy** — уникальная возможность для multi-LLM
4. **Production-ready** — 2.1k stars, активная разработка

Начать с Ollama client — там максимальный impact.

## Ссылки

- [GitHub](https://github.com/failsafe-go/failsafe-go)
- [Documentation](https://failsafe-go.dev)
- [failsafehttp](https://github.com/failsafe-go/failsafe-go/tree/main/failsafehttp) — HTTP middleware
- [failsafegrpc](https://github.com/failsafe-go/failsafe-go/tree/main/failsafegrpc) — gRPC middleware

