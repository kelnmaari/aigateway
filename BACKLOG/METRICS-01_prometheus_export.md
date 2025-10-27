# METRICS-01: Prometheus Metrics Export

**Версия:** 1.10.0  
**Приоритет:** Medium  
**Сложность:** Low  
**Оценка:** 4-6 часов

## Описание

Экспорт метрик в формате Prometheus для enterprise monitoring и alerting. Предоставление `/metrics` endpoint со всеми ключевыми метриками приложения.

## Проблема

В текущей версии:
- Нет централизованного monitoring
- Невозможно настроить alerting
- Нет visibility в production метрики
- Интеграция с Grafana/Prometheus затруднена

## Решение

Prometheus exporter с comprehensive metrics.

### Архитектура

```
Application → [Metrics Collector] → Prometheus Exporter
                                            ↓
                                      /metrics endpoint
                                            ↓
                              Prometheus Scraper → Grafana
```

## Технические детали

### Metrics Categories

**1. HTTP Metrics**
- Request count (by endpoint, method, status)
- Request duration (histogram)
- Response size (histogram)
- Active connections

**2. API Usage Metrics**
- API key usage (by key, model)
- Token usage (prompt, completion, total)
- Request success/failure rate

**3. Model Metrics**
- Model requests (by model)
- Model latency (by model)
- Model errors

**4. System Metrics**
- Loaded models count
- Database connections
- Goroutines count
- Memory usage

**5. Quota Metrics**
- Quota usage (by user, tenant)
- Quota exceeded count

### Implementation

```go
// internal/metrics/prometheus.go

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // HTTP Metrics
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ollama_proxy_http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ollama_proxy_http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    // API Usage Metrics
    apiTokensUsed = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ollama_proxy_api_tokens_used_total",
            Help: "Total tokens used",
        },
        []string{"api_key_id", "model", "type"}, // type: prompt, completion
    )
    
    apiRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ollama_proxy_api_requests_total",
            Help: "Total API requests",
        },
        []string{"model", "status"}, // status: success, error
    )
    
    // Model Metrics
    modelRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "ollama_proxy_model_request_duration_seconds",
            Help:    "Model request duration in seconds",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
        },
        []string{"model"},
    )
    
    modelsLoadedGauge = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "ollama_proxy_models_loaded",
            Help: "Number of currently loaded models",
        },
    )
    
    // Quota Metrics
    quotaUsageGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "ollama_proxy_quota_usage",
            Help: "Current quota usage",
        },
        []string{"target_id", "type"}, // type: tokens_daily, requests_daily
    )
    
    quotaExceededTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ollama_proxy_quota_exceeded_total",
            Help: "Total quota exceeded events",
        },
        []string{"target_id", "type"},
    )
    
    // System Metrics
    dbConnectionsGauge = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "ollama_proxy_db_connections",
            Help: "Number of active database connections",
        },
    )
)

type MetricsCollector struct {
    db storage.Database
    preloader *model.ModelPreloader
}

func NewMetricsCollector(db storage.Database, preloader *model.ModelPreloader) *MetricsCollector {
    return &MetricsCollector{
        db: db,
        preloader: preloader,
    }
}

// Start periodic metrics collection
func (m *MetricsCollector) Start(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            m.collectSystemMetrics(ctx)
        }
    }
}

func (m *MetricsCollector) collectSystemMetrics(ctx context.Context) {
    // Loaded models count
    if m.preloader != nil {
        loadedModels := m.preloader.GetLoadedModels()
        modelsLoadedGauge.Set(float64(len(loadedModels)))
    }
    
    // DB connections (if available)
    // dbConnectionsGauge.Set(float64(m.db.GetConnectionCount()))
}
```

### Middleware для HTTP Metrics

```go
// internal/api/middleware/metrics.go

func PrometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        method := c.Request.Method
        
        c.Next()
        
        status := strconv.Itoa(c.Writer.Status())
        duration := time.Since(start).Seconds()
        
        httpRequestsTotal.WithLabelValues(method, path, status).Inc()
        httpRequestDuration.WithLabelValues(method, path).Observe(duration)
    }
}
```

### Usage Tracking Integration

```go
// internal/api/middleware/usage_tracking.go (update)

func (m *UsageTrackingMiddleware) recordUsage(...) {
    // ... (existing code)
    
    // Export to Prometheus
    apiTokensUsed.WithLabelValues(
        usage.APIKeyID,
        usage.Model,
        "prompt",
    ).Add(float64(usage.PromptTokens))
    
    apiTokensUsed.WithLabelValues(
        usage.APIKeyID,
        usage.Model,
        "completion",
    ).Add(float64(usage.CompletionTokens))
    
    status := "success"
    if !usage.Success {
        status = "error"
    }
    apiRequestsTotal.WithLabelValues(usage.Model, status).Inc()
}
```

### Metrics Endpoint

```go
// internal/api/router/router.go

import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func (r *Router) setupMetricsRoutes() {
    // Prometheus metrics endpoint
    r.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func (r *Router) setupMiddleware() {
    // ... (existing middleware)
    
    // Prometheus metrics middleware
    r.engine.Use(middleware.PrometheusMiddleware())
}
```

### Configuration

```yaml
# config.yaml
metrics:
  enabled: true
  endpoint: "/metrics"
  
  # Optional: Authentication for metrics endpoint
  auth:
    enabled: false
    username: "prometheus"
    password: "${METRICS_PASSWORD}"
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'ollama-proxy'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    
    # Optional: Basic auth
    # basic_auth:
    #   username: prometheus
    #   password: secret
```

### Grafana Dashboard

**Dashboard Template JSON:**

```json
{
  "dashboard": {
    "title": "AIGateway Platform",
    "panels": [
      {
        "title": "HTTP Requests Rate",
        "targets": [
          {
            "expr": "rate(ollama_proxy_http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Request Duration (p95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(ollama_proxy_http_request_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Tokens Used (by Model)",
        "targets": [
          {
            "expr": "rate(ollama_proxy_api_tokens_used_total[5m])"
          }
        ]
      },
      {
        "title": "Model Request Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(ollama_proxy_model_request_duration_seconds_bucket[5m]))"
          }
        ]
      },
      {
        "title": "Loaded Models Count",
        "targets": [
          {
            "expr": "ollama_proxy_models_loaded"
          }
        ]
      },
      {
        "title": "Quota Exceeded Events",
        "targets": [
          {
            "expr": "rate(ollama_proxy_quota_exceeded_total[5m])"
          }
        ]
      }
    ]
  }
}
```

### Example Queries

**Request rate:**
```promql
rate(ollama_proxy_http_requests_total[5m])
```

**Error rate:**
```promql
rate(ollama_proxy_http_requests_total{status=~"5.."}[5m])
```

**p95 latency:**
```promql
histogram_quantile(0.95, rate(ollama_proxy_http_request_duration_seconds_bucket[5m]))
```

**Tokens per second:**
```promql
rate(ollama_proxy_api_tokens_used_total[1m])
```

**Most used model:**
```promql
topk(5, sum by (model) (rate(ollama_proxy_api_requests_total[5m])))
```

## Требования

### Функциональные

1. ✅ `/metrics` endpoint
2. ✅ HTTP metrics (requests, duration, status)
3. ✅ API usage metrics (tokens, requests)
4. ✅ Model metrics (requests, latency)
5. ✅ Quota metrics (usage, exceeded)
6. ✅ System metrics (loaded models, connections)
7. ✅ Prometheus format compliance
8. ✅ Optional basic auth для metrics endpoint
9. ✅ Grafana dashboard template
10. ✅ Documentation для setup

### Нефункциональные

1. **Performance**
   - Metrics collection < 1ms overhead
   - Efficient label cardinality

2. **Compatibility**
   - Prometheus format
   - Grafana integration
   - Standard metric naming

## Acceptance Criteria

- [ ] `/metrics` endpoint возвращает Prometheus metrics
- [ ] HTTP metrics собираются автоматически
- [ ] API usage metrics экспортируются
- [ ] Model metrics отражают latency
- [ ] Quota metrics обновляются
- [ ] Prometheus scraping работает
- [ ] Grafana dashboard отображает metrics
- [ ] Optional auth защищает endpoint
- [ ] Documentation содержит setup instructions
- [ ] Example queries работают
- [ ] Metric naming следует conventions
- [ ] Unit tests для metrics collection

## Риски и зависимости

### Риски

1. **High cardinality** - слишком много labels
   - Mitigation: Limit label values, aggregate

2. **Performance overhead** - metrics collection
   - Mitigation: Efficient collection, sampling

### Зависимости

1. **prometheus/client_golang** - Prometheus Go client
2. Existing middleware, usage tracking

## Связанные задачи

- **MODEL-02**: Model Preloading - экспорт loaded models count
- **QUOTA-01**: Usage Quotas - экспорт quota metrics
- **WS-01**: WebSocket - экспорт connection metrics

## Примечания

- Label cardinality важна для performance
- Не экспортировать sensitive data (API keys, passwords)
- Рекомендуется Grafana для visualization

---

**Статус:** 📋 Planned for v1.10.0  
**Последнее обновление:** 2025-10-11


