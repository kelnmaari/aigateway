# PERF-01: Performance Monitoring with MoniGo Integration

**Версия:** v1.9.3  
**Приоритет:** HIGH  
**Оценка:** 6-8 часов  
**Статус:** In Progress  

---

## 📋 Описание

Интеграция библиотеки [MoniGo](https://github.com/iyashjayesh/monigo) для real-time performance мониторинга Go приложения с реорганизацией структуры Admin Panel.

---

## 🎯 Цели

### 1. Performance Monitoring

- ✅ Real-time метрики CPU, Memory, Goroutines
- ✅ Service-level и Function-level metrics
- ✅ Historical data с графиками
- ✅ Health percentage индикаторы
- ✅ Export в Excel формат

### 2. Реорганизация Admin Panel

- ✅ Убрать "Available Models" из System → создать отдельную вкладку "Models"
- ✅ Убрать "System Logs" из System (перенести в отдельную вкладку или удалить)
- ✅ В System оставить только Performance мониторинг

---

## 🏗️ Архитектура

### Backend Components

```
cmd/server/main.go
├── MoniGo Initialization
│   ├── ServiceName: "aigateway"
│   ├── Dashboard Middleware (JWT Auth)
│   └── Custom Metrics (Ollama latency, API keys)
├── Gin Middleware Integration
│   └── monigoInstance.GetGinHandler()
└── Dashboard Route Registration
    └── /admin/performance
```

### Frontend Structure

```
web/
├── admin.html (updated)
│   ├── Tab: Users
│   ├── Tab: Tenants
│   ├── Tab: API Keys
│   ├── Tab: Models (NEW - moved from System)
│   ├── Tab: Logs (NEW - moved from System)
│   └── Tab: System (UPDATED - only Performance)
├── models.html (NEW)
│   ├── Available Models List
│   ├── Model Details (context, parameters)
│   └── Refresh Models Button
└── js/
    ├── performance.js (NEW)
    │   ├── PerformanceMonitor class
    │   ├── Real-time metrics polling
    │   └── Quick stats cards update
    └── models.js (NEW)
        ├── fetchModels()
        ├── renderModelsList()
        └── showModelDetails()
```

---

## 📊 MoniGo Metrics Available

### Load Statistics

| Metric | Description |
|--------|-------------|
| overall_load_of_service | Overall service load percentage |
| service_cpu_load | CPU load by service |
| service_memory_load | Memory load by service |
| system_cpu_load | System-wide CPU load |
| system_memory_load | System-wide memory load |

### CPU Statistics

| Metric | Description |
|--------|-------------|
| total_cores | Total CPU cores |
| cores_used_by_service | Cores used by service |
| cores_used_by_system | Cores used by system |

### Memory Statistics

| Metric | Description |
|--------|-------------|
| total_system_memory | Total system memory (MB) |
| memory_used_by_system | Memory used by system |
| memory_used_by_service | Memory used by service |
| available_memory | Available memory |
| gc_pause_duration | GC pause duration (ms) |
| stack_memory_usage | Stack memory usage |

### Health Metrics

| Metric | Description |
|--------|-------------|
| service_health_percent | Service health percentage |
| system_health_percent | System health percentage |

---

## 🔧 Технические детали

### Dependencies

```go
go get github.com/iyashjayesh/monigo
```

### Main.go Integration

```go
monigoInstance := &monigo.Monigo{
    ServiceName: "aigateway",
    DashboardMiddleware: []func(http.Handler) http.Handler{
        middleware.MonigoJWTAuth(jwtManager),
        monigo.LoggingMiddleware(),
    },
    CustomMetrics: map[string]interface{}{
        "ollama_url": cfg.Ollama.BaseURL,
        "database": cfg.Database.Type,
    },
}

monigoInstance.Initialize()
ginRouter.Use(monigoInstance.GetGinHandler())
monigoInstance.RegisterWithGinAtCustomPath(ginRouter, "/admin/performance")
```

### JWT Auth Middleware

```go
// internal/api/middleware/monigo_auth.go
func MonigoJWTAuth(jwtManager *jwt.Manager) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Static files bypass auth
            if isStaticFile(r.URL.Path) {
                next.ServeHTTP(w, r)
                return
            }
            
            token := extractTokenFromCookie(r)
            if token == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            claims, err := jwtManager.ValidateToken(token)
            if err != nil || !claims.IsAdmin {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Custom Metrics Collection

```go
// internal/metrics/custom_monigo.go
func SetupCustomMonigoMetrics(m *monigo.Monigo, db storage.Database, cfg *config.Config) {
    // Ollama latency metrics
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        for range ticker.C {
            stats, _ := db.GetPerformanceStats(context.Background(), 
                time.Now().Add(-10*time.Second), time.Now())
            if stats != nil {
                m.CustomMetrics["ollama_avg_latency_ms"] = stats.AvgLatencyMS
                m.CustomMetrics["ollama_p95_latency_ms"] = stats.P95LatencyMS
                m.CustomMetrics["requests_per_second"] = stats.RequestsPerSecond
            }
        }
    }()
    
    // Active API keys count
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        for range ticker.C {
            count, _ := db.CountActiveAPIKeys(context.Background())
            m.CustomMetrics["active_api_keys"] = count
        }
    }()
}
```

---

## 🎨 UI Changes

### Admin Panel Tabs (Before → After)

**Before:**

```
[Users] [Tenants] [API Keys] [System]
                                ├── Server Info
                                ├── Available Models
                                ├── Changelogs
                                └── System Logs
```

**After:**

```
[Users] [Tenants] [API Keys] [Models] [Logs] [System]
                              │        │      └── Performance Monitor
                              │        │          ├── Embedded MoniGo Dashboard
                              │        │          ├── Quick Stats Cards
                              │        │          └── Export Report Button
                              │        └── System Logs
                              │            ├── Real-time log streaming
                              │            └── Log level filtering
                              └── Available Models
                                  ├── Models List
                                  ├── Model Details
                                  └── Refresh Button
```

### Performance Tab UI Components

#### 1. Embedded MoniGo Dashboard

```html
<iframe 
    src="/admin/performance" 
    style="width: 100%; height: 800px; border: none;">
</iframe>
```

#### 2. Quick Stats Cards

```html
<div class="row">
    <div class="col-md-3">
        <div class="stats-card">
            <h6>CPU Usage</h6>
            <h3 id="cpu-usage">-</h3>
            <small>Service Load</small>
        </div>
    </div>
    <!-- Memory, Goroutines, Health cards -->
</div>
```

#### 3. Export Button

```html
<button onclick="performanceMonitor.exportReport('excel')">
    Export Report
</button>
```

---

## 🔒 Security

### Authentication

- ✅ JWT-based auth для dashboard
- ✅ Admin-only access
- ✅ Static files (CSS/JS) bypass auth
- ✅ Cookie-based token extraction

### Rate Limiting

- ✅ 100 requests/minute для API endpoints
- ✅ Защита от abuse

---

## 📈 API Endpoints

### MoniGo Built-in API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/performance/api/v1/metrics` | GET | Get all metrics |
| `/admin/performance/api/v1/go-routines-stats` | GET | Goroutines stats |
| `/admin/performance/api/v1/service-info` | GET | Service info |
| `/admin/performance/api/v1/service-metrics` | POST | Service metrics |
| `/admin/performance/api/v1/reports` | POST | Generate report |

---

## ✅ Acceptance Criteria

### Backend

- [ ] MoniGo dependency добавлен в go.mod
- [ ] Middleware интегрирован в Gin router
- [ ] JWT auth middleware реализован
- [ ] Custom metrics собираются (Ollama latency, API keys)
- [ ] Dashboard доступен на `/admin/performance`

### Frontend

- [ ] Admin panel реорганизован (Models, Logs отдельно)
- [ ] Performance tab создан с embedded dashboard
- [ ] Quick stats cards отображаются и обновляются
- [ ] Export report работает
- [ ] Models page работает (бывший Available Models)
- [ ] Logs page работает (бывший System Logs)

### Documentation

- [ ] CHANGELOG.md обновлен для v1.9.3
- [ ] SQL migration v24 создан
- [ ] VERSION обновлен до 1.9.3
- [ ] Roadmap.MD обновлен

---

## 🧪 Testing

### Manual Testing Checklist

- [ ] Dashboard загружается без ошибок
- [ ] Метрики обновляются в real-time
- [ ] JWT auth работает (admin access only)
- [ ] Quick stats cards показывают актуальные данные
- [ ] Export report генерирует Excel файл
- [ ] Models tab показывает список моделей
- [ ] Logs tab показывает логи

### Performance Impact

- [ ] Overhead < 5% CPU
- [ ] Memory impact < 50MB
- [ ] No goroutine leaks

---

## 📝 Implementation Steps

1. ✅ Create BACKLOG/PERF-01_monigo_integration.md
2. ✅ Update Roadmap.MD
3. ✅ Update VERSION → 1.9.3
4. ⏳ Add dependency: `go get github.com/iyashjayesh/monigo`
5. ⏳ Create internal/api/middleware/monigo_auth.go
6. ⏳ Create internal/metrics/custom_monigo.go
7. ⏳ Update cmd/server/main.go (MoniGo integration)
8. ⏳ Reorganize web/admin.html (tabs structure)
9. ⏳ Create web/models.html (new Models page)
10. ⏳ Create web/js/models.js
11. ⏳ Create web/logs.html (new Logs page)
12. ⏳ Create web/js/performance.js
13. ⏳ Update CHANGELOG.md
14. ⏳ Create SQL migration v24
15. ⏳ Test all functionality

---

## 🚀 Future Enhancements

### Phase 2 (v1.9.4+)

- [ ] Custom dashboards configuration
- [ ] Alert rules для критических метрик
- [ ] Slack/Email notifications
- [ ] Performance baselines и regression detection
- [ ] Multi-node monitoring (если будет clustering)

### Phase 3 (v2.0.0+)

- [ ] Prometheus export
- [ ] Grafana integration
- [ ] Distributed tracing correlation
- [ ] APM-style transaction tracking

---

## 📚 References

- MoniGo GitHub: <https://github.com/iyashjayesh/monigo>
- MoniGo Docs: <https://pkg.go.dev/github.com/iyashjayesh/monigo>
- Performance Monitoring Best Practices: <https://go.dev/blog/pprof>
- Gin Middleware Guide: <https://gin-gonic.com/docs/examples/custom-middleware/>

