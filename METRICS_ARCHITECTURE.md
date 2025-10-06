# Metrics Architecture - Ollama-OpenAI Proxy

Архитектура системы мониторинга и визуализации метрик.

## 🏗️ Общая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    Ollama-OpenAI Proxy                      │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │   HTTP API   │───▶│   Metrics    │◀───│ Ollama Client│ │
│  │   Handlers   │    │  Collector   │    │              │ │
│  └──────────────┘    └──────────────┘    └──────────────┘ │
│         │                    │                             │
│         │                    │                             │
│         ▼                    ▼                             │
│  ┌──────────────────────────────────────────────────────┐ │
│  │          Prometheus Metrics Registry                 │ │
│  │  - HTTP metrics (latency, status, endpoints)         │ │
│  │  - Ollama metrics (requests, errors, duration)       │ │
│  │  - API Key metrics (usage, rate limits)              │ │
│  │  - System metrics (goroutines, memory, GC)           │ │
│  └──────────────────────────────────────────────────────┘ │
│                           │                                │
└───────────────────────────┼────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
  │  /metrics   │   │     TUI     │   │   WebUI     │
  │  endpoint   │   │  Dashboard  │   │  Dashboard  │
  │             │   │             │   │             │
  │ (Prometheus)│   │ (Terminal)  │   │   (Web)     │
  └─────────────┘   └─────────────┘   └─────────────┘
         │                  │                  │
         │                  │                  │
         ▼                  ▼                  ▼
  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
  │ Prometheus  │   │  Real-time  │   │  Real-time  │
  │   Server    │   │   Graphs    │   │   Charts    │
  │             │   │             │   │             │
  │  + Grafana  │   │ Live updates│   │  WebSocket  │
  └─────────────┘   └─────────────┘   └─────────────┘
```

## 📊 Типы метрик

### 1. HTTP Request Metrics

```go
// Counter: Общее количество запросов
http_requests_total{method, endpoint, status_code}

// Histogram: Длительность запросов
http_request_duration_seconds{method, endpoint}

// Gauge: Активные запросы
http_requests_in_flight{endpoint}
```

### 2. Ollama Client Metrics

```go
// Counter: Запросы к Ollama
ollama_requests_total{model, operation, status}

// Histogram: Латентность Ollama
ollama_request_duration_seconds{model, operation}

// Counter: Ошибки Ollama
ollama_errors_total{model, operation, error_type}

// Gauge: Активные подключения
ollama_connections_active
```

### 3. API Key Metrics

```go
// Counter: Использование по ключам
api_key_requests_total{key_id, endpoint}

// Counter: Rate limit hits
api_key_rate_limit_exceeded_total{key_id}

// Gauge: Активные ключи
api_keys_active_total

// Counter: Токены использованные
api_key_tokens_used_total{key_id, model}
```

### 4. System Metrics

```go
// Gauge: Goroutines
go_goroutines

// Gauge: Memory allocated
go_memstats_alloc_bytes

// Counter: GC runs
go_gc_duration_seconds

// Gauge: Heap objects
go_memstats_heap_objects
```

## 🎨 TUI Integration (Фаза 10.2)

### Dashboard View

```
╔════════════════════════════════════════════════════════════╗
║             Ollama-OpenAI Proxy - Dashboard                ║
╠════════════════════════════════════════════════════════════╣
║                                                            ║
║  📊 Request Rate (last 1m):  ▂▃▅▇█▇▅▃▂   1,234 req/s     ║
║  ⏱️  Avg Latency:            ▂▂▃▄▃▂▂▁    23.5 ms          ║
║  ❌ Error Rate:              ▁▁▁▂▁▁▁▁    0.12%            ║
║                                                            ║
║  🔑 Top API Keys (by requests):                            ║
║    1. ak_prod_***cc75     12,345 req  (45.2%)            ║
║    2. ak_test_***ab12      8,901 req  (32.6%)            ║
║    3. ak_dev_***xy89       6,123 req  (22.4%)            ║
║                                                            ║
║  🤖 Models Usage:                                          ║
║    llama2        ████████████░░░░  67%  (18,234 req)     ║
║    mistral       ████░░░░░░░░░░░░  23%  ( 6,234 req)     ║
║    qwen3-coder   ██░░░░░░░░░░░░░░  10%  ( 2,712 req)     ║
║                                                            ║
║  💾 System:  CPU: 45%  Memory: 234MB  Goroutines: 142    ║
╚════════════════════════════════════════════════════════════╝
```

### Metrics History

- **Ring buffer** для хранения последних N метрик
- **Real-time updates** через ticker (1s interval)
- **Spark lines** для визуализации трендов
- **Aggregation** для различных time windows (1m, 5m, 1h)

## 🌐 WebUI Integration (Фаза 14)

### Features

1. **Interactive Charts** (Chart.js / Recharts)
   - Line charts для request rate и latency
   - Bar charts для top API keys и models
   - Pie charts для status code distribution

2. **Real-time Updates** (WebSocket)
   - Live metrics streaming
   - Auto-refresh dashboard
   - Alerts и notifications

3. **Historical Data** (опционально)
   - Time range selection
   - Zoom и pan
   - Export to CSV/JSON

4. **Custom Dashboards**
   - Drag-and-drop widgets
   - Save/load configurations
   - Share dashboards

## 🔄 Data Flow

```
1. HTTP Request → Middleware
                     ↓
2. Record start time, increment counter
                     ↓
3. Process request → Handler → Ollama Client
                     ↓
4. Record metrics: duration, status, tokens
                     ↓
5. Update Prometheus registry
                     ↓
6. Metrics available via:
   - /metrics endpoint (pull-based, Prometheus)
   - TUI (push-based, in-memory)
   - WebUI (push-based, WebSocket)
```

## 🛠️ Implementation Plan

### Фаза 10.2: Prometheus + TUI

**Week 1:**
1. ✅ Добавить Prometheus client library
2. ✅ Создать metrics middleware
3. ✅ Implement counters, histograms, gauges
4. ✅ Export /metrics endpoint

**Week 2:**
5. ✅ TUI metrics fetcher
6. ✅ Ring buffer для history
7. ✅ Dashboard visualizations (sparklines, bars)
8. ✅ Real-time updates

### Фаза 14: WebUI

**Later:**
- Frontend setup (React/Vue)
- WebSocket server
- Charts integration
- API keys management UI
- Authentication

## 📦 Dependencies

### Go Packages
```go
// Prometheus
"github.com/prometheus/client_golang/prometheus"
"github.com/prometheus/client_golang/prometheus/promhttp"

// TUI (already have)
"github.com/charmbracelet/bubbles"
"github.com/charmbracelet/bubbletea"
"github.com/charmbracelet/lipgloss"
```

### Frontend (Фаза 14)
```json
{
  "react": "^18.0.0",
  "recharts": "^2.5.0",
  "socket.io-client": "^4.5.0"
}
```

## 🎯 Benefits

1. **Multiple interfaces**: Prometheus, TUI, WebUI
2. **Shared metrics**: One source of truth
3. **Flexibility**: Use what fits your needs
4. **Compatibility**: Standard Prometheus format
5. **Real-time**: Live updates in TUI and WebUI
6. **Historical**: Prometheus storage for long-term

---

**Next Steps**: Start implementing Prometheus metrics export! 🚀

