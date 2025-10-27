# OBSERV-01: OpenTelemetry Integration

**Версия:** v1.6.1  
**Приоритет:** LOW  
**Оценка времени:** 10-12 часов  
**Статус:** 📋 Не начато  

---

## 📋 Описание

Интеграция OpenTelemetry для distributed tracing в production. Позволит отслеживать запросы через все компоненты системы (Proxy → Ollama), анализировать bottlenecks и оптимизировать производительность.

## 🎯 Цели

1. **Distributed Tracing** - сквозная трассировка запросов
2. **Trace Context Propagation** - передача контекста между компонентами
3. **Span Annotations** - детальная информация о каждой операции
4. **Integration** - поддержка Jaeger, Zipkin, OTLP exporters

## 🔧 Технические детали

### 1. OpenTelemetry SDK Setup

**Пакеты:**
```go
go.opentelemetry.io/otel
go.opentelemetry.io/otel/trace
go.opentelemetry.io/otel/sdk/trace
go.opentelemetry.io/otel/exporters/jaeger
go.opentelemetry.io/otel/exporters/zipkin
go.opentelemetry.io/otel/exporters/otlp/otlptrace
go.opentelemetry.io/otel/propagation
```

**Конфигурация:**
```yaml
# configs/dev.yaml
observability:
  tracing:
    enabled: true
    provider: "jaeger"  # jaeger | zipkin | otlp
    
    # Jaeger specific
    jaeger:
      endpoint: "http://localhost:14268/api/traces"
      service_name: "ollama-proxy"
      
    # Zipkin specific  
    zipkin:
      endpoint: "http://localhost:9411/api/v2/spans"
      
    # OTLP specific
    otlp:
      endpoint: "localhost:4317"
      insecure: true
      
    # Sampling
    sampling_rate: 1.0  # 1.0 = 100%, 0.1 = 10%
```

### 2. Tracer Provider Initialization

**Файл:** `internal/observability/tracer.go`

```go
package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

type TracingConfig struct {
	Enabled      bool
	Provider     string  // jaeger, zipkin, otlp
	ServiceName  string
	Endpoint     string
	SamplingRate float64
}

type TracerProvider struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracerProvider creates and configures OpenTelemetry tracer provider
func NewTracerProvider(cfg TracingConfig) (*TracerProvider, error) {
	if !cfg.Enabled {
		// Return no-op tracer if tracing is disabled
		return &TracerProvider{
			tracer: trace.NewNoopTracerProvider().Tracer("noop"),
		}, nil
	}

	// Create resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String("1.6.1"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on provider type
	var exporter sdktrace.SpanExporter
	switch cfg.Provider {
	case "jaeger":
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(cfg.Endpoint),
		))
	case "zipkin":
		exporter, err = zipkin.New(cfg.Endpoint)
	default:
		return nil, fmt.Errorf("unsupported tracing provider: %s", cfg.Provider)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create tracer provider with sampling
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SamplingRate)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: tp,
		tracer:   tp.Tracer("ollama-proxy"),
	}, nil
}

// Tracer returns the tracer instance
func (tp *TracerProvider) Tracer() trace.Tracer {
	return tp.tracer
}

// Shutdown flushes and shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}
```

### 3. HTTP Middleware для трассировки

**Файл:** `internal/api/middleware/tracing.go`

```go
package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TracingMiddleware adds OpenTelemetry tracing to HTTP requests
func TracingMiddleware(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from incoming request
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Start new span
			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.url", r.URL.String()),
					attribute.String("http.scheme", r.URL.Scheme),
					attribute.String("http.host", r.Host),
					attribute.String("http.target", r.URL.Path),
					attribute.String("http.user_agent", r.UserAgent()),
				),
			)
			defer span.End()

			// Wrap response writer to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler with traced context
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Record response status
			span.SetAttributes(
				attribute.Int("http.status_code", ww.statusCode),
			)

			// Mark span as error if status >= 400
			if ww.statusCode >= 400 {
				span.SetStatus(codes.Error, http.StatusText(ww.statusCode))
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
```

### 4. Ollama Client трассировка

**Файл:** `internal/client/ollama/client_tracing.go`

```go
package ollama

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ChatCompletionWithTracing wraps ChatCompletion with tracing
func (c *Client) ChatCompletionWithTracing(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Get tracer from context or use global
	tracer := trace.SpanFromContext(ctx).TracerProvider().Tracer("ollama-client")

	// Start span for Ollama request
	ctx, span := tracer.Start(ctx, "ollama.chat_completion",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("ollama.model", req.Model),
			attribute.Int("ollama.messages_count", len(req.Messages)),
			attribute.Bool("ollama.stream", req.Stream),
		),
	)
	defer span.End()

	// Make actual request
	resp, err := c.ChatCompletion(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	// Record success metrics
	span.SetAttributes(
		attribute.Int("ollama.response_tokens", resp.TotalTokens),
		attribute.Int64("ollama.duration_ms", resp.DurationMS),
	)
	span.SetStatus(codes.Ok, "")

	return resp, nil
}
```

### 5. Span Annotations для ключевых операций

**Операции для трассировки:**

1. **API Request** (`internal/api/handlers/chat.go`)
   - Span: `POST /v1/chat/completions`
   - Attributes: model, user_id, api_key_id, streaming

2. **Request Conversion** (`internal/converter/chat.go`)
   - Span: `convert.openai_to_ollama`
   - Attributes: input_format, output_format, message_count

3. **Ollama Request** (`internal/client/ollama/client.go`)
   - Span: `ollama.chat_completion`
   - Attributes: model, tokens, duration

4. **Response Conversion** (`internal/converter/response.go`)
   - Span: `convert.ollama_to_openai`
   - Attributes: tokens_generated, finish_reason

5. **Database Operations** (`internal/storage/*.go`)
   - Span: `db.query` / `db.insert` / `db.update`
   - Attributes: table, operation, duration

### 6. Context Propagation

**Передача контекста:**
- HTTP headers (W3C Trace Context)
- Внутри goroutines
- Между микросервисами (если будет split)

```go
// Пример передачи контекста в goroutine
func (h *ChatHandler) processAsync(ctx context.Context, req Request) {
	// Extract span context
	spanCtx := trace.SpanContextFromContext(ctx)
	
	go func() {
		// Create new context with span context
		newCtx := trace.ContextWithSpanContext(context.Background(), spanCtx)
		
		// Start child span
		_, span := h.tracer.Start(newCtx, "async_processing")
		defer span.End()
		
		// Do work...
	}()
}
```

## 📁 Структура файлов

```
internal/
├── observability/
│   ├── tracer.go           # TracerProvider setup
│   ├── tracer_test.go      # Tests
│   └── noop.go             # No-op implementation
├── api/
│   └── middleware/
│       ├── tracing.go      # HTTP tracing middleware
│       └── tracing_test.go
├── client/
│   └── ollama/
│       ├── client_tracing.go  # Ollama client tracing
│       └── client_tracing_test.go
└── storage/
    └── tracing.go          # Database tracing helpers
```

## 🧪 Тестирование

### Unit Tests

```go
func TestTracerProvider_Init(t *testing.T) {
	cfg := TracingConfig{
		Enabled:      true,
		Provider:     "jaeger",
		ServiceName:  "test-service",
		Endpoint:     "http://localhost:14268/api/traces",
		SamplingRate: 1.0,
	}
	
	provider, err := NewTracerProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer provider.Shutdown(context.Background())
	
	// Test tracer creation
	tracer := provider.Tracer()
	require.NotNil(t, tracer)
}

func TestTracingMiddleware(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("test")
	middleware := TracingMiddleware(tracer)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check span is in context
		span := trace.SpanFromContext(r.Context())
		assert.NotNil(t, span)
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	
	middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}
```

### Integration Tests

**Jaeger All-in-One (для тестирования):**
```bash
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest
```

**Проверка трассировки:**
1. Запустить Jaeger
2. Сделать несколько запросов к API
3. Открыть Jaeger UI: http://localhost:16686
4. Найти traces для "ollama-proxy"
5. Проверить наличие spans для всех операций

## 📊 Результаты

**После реализации:**

1. ✅ **Distributed Tracing** - полная трассировка запросов
2. ✅ **Span Hierarchy** - дерево операций с таймингами
3. ✅ **Error Tracking** - автоматическое отслеживание ошибок
4. ✅ **Performance Analysis** - выявление bottlenecks
5. ✅ **Context Propagation** - передача контекста между компонентами

**Пример trace:**
```
POST /v1/chat/completions [200ms]
├── convert.openai_to_ollama [2ms]
├── ollama.chat_completion [190ms]
│   ├── http.request [185ms]
│   └── response.parse [5ms]
└── convert.ollama_to_openai [8ms]
```

## 🔗 Зависимости

**Go packages:**
```bash
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/exporters/jaeger@latest
go get go.opentelemetry.io/otel/exporters/zipkin@latest
```

## 📝 Конфигурация

**Добавить в `internal/config/config.go`:**
```go
type ObservabilityConfig struct {
	Tracing TracingConfig `yaml:"tracing"`
}

type TracingConfig struct {
	Enabled      bool              `yaml:"enabled"`
	Provider     string            `yaml:"provider"`
	ServiceName  string            `yaml:"service_name"`
	Jaeger       JaegerConfig      `yaml:"jaeger"`
	Zipkin       ZipkinConfig      `yaml:"zipkin"`
	OTLP         OTLPConfig        `yaml:"otlp"`
	SamplingRate float64           `yaml:"sampling_rate"`
}
```

## 🎯 Критерии успеха

- [ ] TracerProvider успешно инициализируется
- [ ] HTTP middleware добавляет spans
- [ ] Ollama client requests трассируются
- [ ] Database operations имеют spans
- [ ] Context propagation работает
- [ ] Traces видны в Jaeger UI
- [ ] Unit tests покрывают >95% кода
- [ ] Integration tests с Jaeger проходят
- [ ] Документация обновлена

## 📖 Документация

**Обновить:**
- `docs/CONFIGURATION.md` - новая секция Observability
- `docs/PERFORMANCE.md` - как использовать tracing
- `README.md` - упомянуть OpenTelemetry support
- `configs/production.yaml.example` - примеры конфигурации

## 🚀 Внедрение

**cmd/server/main.go:**
```go
// Initialize tracer provider
tracerProvider, err := observability.NewTracerProvider(observability.TracingConfig{
	Enabled:      cfg.Observability.Tracing.Enabled,
	Provider:     cfg.Observability.Tracing.Provider,
	ServiceName:  "ollama-proxy",
	Endpoint:     cfg.Observability.Tracing.Jaeger.Endpoint,
	SamplingRate: cfg.Observability.Tracing.SamplingRate,
})
if err != nil {
	log.Fatalf("Failed to initialize tracer: %v", err)
}
defer tracerProvider.Shutdown(context.Background())

// Add tracing middleware to router
router.Use(middleware.TracingMiddleware(tracerProvider.Tracer()))
```

---

**Оценка времени:** 10-12 часов  
**Сложность:** HIGH (требует понимания distributed tracing)  
**Зависимости:** Нет  
**Блокирует:** OBSERV-02 (будет использовать spans)


