# OBSERV-02: Performance Monitoring

**Версия:** v1.6.2  
**Приоритет:** LOW  
**Оценка времени:** 6-8 часов  
**Статус:** 📋 Не начато  

---

## 📋 Описание

Continuous performance monitoring в production. Автоматическое профилирование CPU/memory, детекция утечек памяти, анализ медленных запросов и выявление performance regression.

## 🎯 Цели

1. **Continuous Profiling** - постоянное профилирование в production
2. **Memory Leak Detection** - автоматическое выявление утечек
3. **Slow Request Analysis** - анализ медленных запросов
4. **Performance Regression Detection** - сравнение с baseline

## 🔧 Технические детали

### 1. Runtime Profiling

**Пакеты:**
```go
runtime/pprof
net/http/pprof
runtime
runtime/debug
```

**Endpoints:**
```
GET /debug/pprof/          - index page
GET /debug/pprof/profile   - CPU profile (30s default)
GET /debug/pprof/heap      - memory heap profile
GET /debug/pprof/goroutine - goroutine dump
GET /debug/pprof/allocs    - all memory allocations
GET /debug/pprof/block     - blocking profile
GET /debug/pprof/mutex     - mutex contention profile
GET /debug/pprof/trace     - execution trace
```

### 2. Performance Monitor Service

**Файл:** `internal/observability/perfmon.go`

```go
package observability

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type PerformanceMonitor struct {
	logger    *logrus.Logger
	config    PerfMonConfig
	metrics   *PerformanceMetrics
	baseline  *PerformanceMetrics
	mu        sync.RWMutex
	stopCh    chan struct{}
}

type PerfMonConfig struct {
	Enabled              bool
	CollectionInterval   time.Duration // Default: 30s
	MemoryThresholdMB    int64         // Alert if exceeds (default: 1024 MB)
	GoroutineThreshold   int           // Alert if exceeds (default: 1000)
	SlowRequestThreshold time.Duration // Log if slower (default: 5s)
	GCPercentage         int           // GOGC value (default: 100)
}

type PerformanceMetrics struct {
	Timestamp        time.Time
	HeapAllocMB      float64
	HeapSysMB        float64
	NumGC            uint32
	NumGoroutines    int
	NumCPU           int
	GCPauseMS        float64
	AllocRate        float64 // MB/s
	RequestsPerSec   float64
	AvgLatencyMS     float64
	P95LatencyMS     float64
	P99LatencyMS     float64
}

func NewPerformanceMonitor(logger *logrus.Logger, cfg PerfMonConfig) *PerformanceMonitor {
	if !cfg.Enabled {
		return nil
	}

	// Set default values
	if cfg.CollectionInterval == 0 {
		cfg.CollectionInterval = 30 * time.Second
	}
	if cfg.MemoryThresholdMB == 0 {
		cfg.MemoryThresholdMB = 1024
	}
	if cfg.GoroutineThreshold == 0 {
		cfg.GoroutineThreshold = 1000
	}
	if cfg.SlowRequestThreshold == 0 {
		cfg.SlowRequestThreshold = 5 * time.Second
	}
	if cfg.GCPercentage == 0 {
		cfg.GCPercentage = 100
	}

	// Configure garbage collector
	debug.SetGCPercent(cfg.GCPercentage)

	pm := &PerformanceMonitor{
		logger:  logger,
		config:  cfg,
		metrics: &PerformanceMetrics{},
		stopCh:  make(chan struct{}),
	}

	// Collect initial baseline
	pm.collectMetrics()
	pm.baseline = pm.metrics

	return pm
}

// Start begins continuous monitoring
func (pm *PerformanceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(pm.config.CollectionInterval)
	defer ticker.Stop()

	pm.logger.WithFields(logrus.Fields{
		"interval":  pm.config.CollectionInterval,
		"mem_limit": pm.config.MemoryThresholdMB,
		"gc_pct":    pm.config.GCPercentage,
	}).Info("Performance monitor started")

	for {
		select {
		case <-ticker.C:
			pm.collectMetrics()
			pm.analyzeMetrics()
		case <-ctx.Done():
			pm.logger.Info("Performance monitor stopped")
			return
		case <-pm.stopCh:
			return
		}
	}
}

// Stop halts monitoring
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopCh)
}

// collectMetrics gathers current runtime metrics
func (pm *PerformanceMonitor) collectMetrics() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.metrics = &PerformanceMetrics{
		Timestamp:     time.Now(),
		HeapAllocMB:   float64(m.HeapAlloc) / 1024 / 1024,
		HeapSysMB:     float64(m.HeapSys) / 1024 / 1024,
		NumGC:         m.NumGC,
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
	}

	// Calculate GC pause time (average of last 256 pauses)
	if m.NumGC > 0 {
		totalPause := uint64(0)
		count := int(m.NumGC)
		if count > 256 {
			count = 256
		}
		for i := 0; i < count; i++ {
			totalPause += m.PauseNs[(m.NumGC-uint32(i)+255)%256]
		}
		pm.metrics.GCPauseMS = float64(totalPause) / float64(count) / 1e6
	}

	// Calculate allocation rate
	if pm.baseline != nil && !pm.baseline.Timestamp.IsZero() {
		duration := pm.metrics.Timestamp.Sub(pm.baseline.Timestamp).Seconds()
		if duration > 0 {
			allocDiff := pm.metrics.HeapAllocMB - pm.baseline.HeapAllocMB
			pm.metrics.AllocRate = allocDiff / duration
		}
	}
}

// analyzeMetrics checks for anomalies and logs warnings
func (pm *PerformanceMonitor) analyzeMetrics() {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	m := pm.metrics

	// Memory threshold check
	if int64(m.HeapAllocMB) > pm.config.MemoryThresholdMB {
		pm.logger.WithFields(logrus.Fields{
			"heap_alloc_mb": m.HeapAllocMB,
			"threshold_mb":  pm.config.MemoryThresholdMB,
			"goroutines":    m.NumGoroutines,
		}).Warn("Memory usage exceeds threshold")
	}

	// Goroutine leak detection
	if m.NumGoroutines > pm.config.GoroutineThreshold {
		pm.logger.WithFields(logrus.Fields{
			"num_goroutines": m.NumGoroutines,
			"threshold":      pm.config.GoroutineThreshold,
		}).Warn("Goroutine count exceeds threshold - possible leak")
	}

	// Memory leak detection (continuously growing allocation)
	if pm.baseline != nil && m.AllocRate > 10 { // >10 MB/s sustained growth
		pm.logger.WithFields(logrus.Fields{
			"alloc_rate_mb_s": m.AllocRate,
			"heap_alloc_mb":   m.HeapAllocMB,
			"duration":        m.Timestamp.Sub(pm.baseline.Timestamp),
		}).Warn("High memory allocation rate - possible memory leak")
	}

	// Log periodic metrics
	pm.logger.WithFields(logrus.Fields{
		"heap_alloc_mb":   fmt.Sprintf("%.2f", m.HeapAllocMB),
		"heap_sys_mb":     fmt.Sprintf("%.2f", m.HeapSysMB),
		"num_gc":          m.NumGC,
		"gc_pause_ms":     fmt.Sprintf("%.2f", m.GCPauseMS),
		"num_goroutines":  m.NumGoroutines,
		"alloc_rate_mb_s": fmt.Sprintf("%.2f", m.AllocRate),
	}).Debug("Performance metrics collected")
}

// GetMetrics returns current metrics snapshot
func (pm *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	// Return copy to avoid race conditions
	metrics := *pm.metrics
	return &metrics
}

// ResetBaseline resets the baseline metrics
func (pm *PerformanceMonitor) ResetBaseline() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.baseline = pm.metrics
}
```

### 3. Slow Request Detection

**Файл:** `internal/api/middleware/slow_request.go`

```go
package middleware

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// SlowRequestLogger logs requests that exceed threshold
func SlowRequestLogger(logger *logrus.Logger, threshold time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			// Log slow requests
			if duration > threshold {
				logger.WithFields(logrus.Fields{
					"method":      r.Method,
					"path":        r.URL.Path,
					"duration_ms": duration.Milliseconds(),
					"status":      ww.statusCode,
					"user_agent":  r.UserAgent(),
					"remote_addr": r.RemoteAddr,
				}).Warn("Slow request detected")
			}
		})
	}
}
```

### 4. Memory Leak Detection

**Файл:** `internal/observability/leak_detector.go`

```go
package observability

import (
	"context"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

type LeakDetector struct {
	logger    *logrus.Logger
	samples   []MemorySample
	maxSamples int
}

type MemorySample struct {
	Timestamp   time.Time
	HeapAllocMB float64
	NumGoroutines int
}

func NewLeakDetector(logger *logrus.Logger) *LeakDetector {
	return &LeakDetector{
		logger:     logger,
		samples:    make([]MemorySample, 0, 10),
		maxSamples: 10,
	}
}

// CheckForLeaks analyzes memory trends
func (ld *LeakDetector) CheckForLeaks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ld.collectSample()
			ld.analyzeTrend()
		case <-ctx.Done():
			return
		}
	}
}

func (ld *LeakDetector) collectSample() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	sample := MemorySample{
		Timestamp:     time.Now(),
		HeapAllocMB:   float64(m.HeapAlloc) / 1024 / 1024,
		NumGoroutines: runtime.NumGoroutine(),
	}

	ld.samples = append(ld.samples, sample)
	if len(ld.samples) > ld.maxSamples {
		ld.samples = ld.samples[1:]
	}
}

func (ld *LeakDetector) analyzeTrend() {
	if len(ld.samples) < 5 {
		return // Need at least 5 samples
	}

	// Check if memory is consistently growing
	growthCount := 0
	for i := 1; i < len(ld.samples); i++ {
		if ld.samples[i].HeapAllocMB > ld.samples[i-1].HeapAllocMB {
			growthCount++
		}
	}

	// If >80% of samples show growth
	if float64(growthCount)/float64(len(ld.samples)-1) > 0.8 {
		first := ld.samples[0]
		last := ld.samples[len(ld.samples)-1]
		growth := last.HeapAllocMB - first.HeapAllocMB
		duration := last.Timestamp.Sub(first.Timestamp)

		ld.logger.WithFields(logrus.Fields{
			"growth_mb":      growth,
			"duration":       duration,
			"initial_mb":     first.HeapAllocMB,
			"current_mb":     last.HeapAllocMB,
			"goroutines":     last.NumGoroutines,
			"rate_mb_min":    growth / duration.Minutes(),
		}).Warn("Possible memory leak detected")
	}
}
```

### 5. Performance Dashboard API

**Файл:** `internal/api/handlers/performance.go`

```go
package handlers

import (
	"encoding/json"
	"net/http"

	"internal/observability"
)

type PerformanceHandler struct {
	monitor *observability.PerformanceMonitor
}

func NewPerformanceHandler(monitor *observability.PerformanceMonitor) *PerformanceHandler {
	return &PerformanceHandler{monitor: monitor}
}

// GetMetrics returns current performance metrics
// GET /api/admin/performance/metrics
func (h *PerformanceHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		http.Error(w, "Performance monitoring disabled", http.StatusServiceUnavailable)
		return
	}

	metrics := h.monitor.GetMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   metrics,
	})
}

// ResetBaseline resets performance baseline
// POST /api/admin/performance/reset-baseline
func (h *PerformanceHandler) ResetBaseline(w http.ResponseWriter, r *http.Request) {
	if h.monitor == nil {
		http.Error(w, "Performance monitoring disabled", http.StatusServiceUnavailable)
		return
	}

	h.monitor.ResetBaseline()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Baseline reset successfully",
	})
}
```

## 📁 Структура файлов

```
internal/
├── observability/
│   ├── perfmon.go          # Performance monitor
│   ├── perfmon_test.go
│   ├── leak_detector.go    # Memory leak detection
│   └── leak_detector_test.go
├── api/
│   ├── handlers/
│   │   ├── performance.go  # Performance API
│   │   └── performance_test.go
│   └── middleware/
│       ├── slow_request.go # Slow request logger
│       └── slow_request_test.go
└── web/
    ├── performance.html    # Performance dashboard
    └── js/
        └── performance.js  # Charts and real-time updates
```

## 🧪 Тестирование

### Unit Tests

```go
func TestPerformanceMonitor_CollectMetrics(t *testing.T) {
	logger := logrus.New()
	cfg := PerfMonConfig{
		Enabled:            true,
		CollectionInterval: 1 * time.Second,
		MemoryThresholdMB:  512,
	}
	
	pm := NewPerformanceMonitor(logger, cfg)
	require.NotNil(t, pm)
	
	metrics := pm.GetMetrics()
	assert.Greater(t, metrics.HeapAllocMB, 0.0)
	assert.Greater(t, metrics.NumGoroutines, 0)
}

func TestLeakDetector_AnalyzeTrend(t *testing.T) {
	logger := logrus.New()
	ld := NewLeakDetector(logger)
	
	// Simulate growing memory
	for i := 0; i < 10; i++ {
		ld.samples = append(ld.samples, MemorySample{
			Timestamp:   time.Now().Add(time.Duration(i) * time.Minute),
			HeapAllocMB: float64(100 + i*10),
		})
	}
	
	// Should detect leak
	ld.analyzeTrend()
}
```

## 📊 WebUI Dashboard

**Страница:** `web/performance.html`

**Компоненты:**
1. **Real-time Metrics**
   - Heap memory graph (last 1 hour)
   - Goroutine count graph
   - GC pause times
   - Allocation rate

2. **Slow Requests Table**
   - Path, duration, timestamp
   - Filter по threshold
   - Export to CSV

3. **Profile Download**
   - CPU profile (30s)
   - Heap profile
   - Goroutine dump
   - Trace (10s)

4. **Leak Detection Status**
   - Current trend
   - Warning indicators
   - Historical data

## 🔗 Зависимости

- `runtime/pprof` (стандартная библиотека)
- `net/http/pprof` (стандартная библиотека)
- OBSERV-01 (опционально, для span timing analysis)

## 📝 Конфигурация

```yaml
# configs/dev.yaml
observability:
  performance:
    enabled: true
    collection_interval: 30s
    memory_threshold_mb: 1024
    goroutine_threshold: 1000
    slow_request_threshold: 5s
    gc_percentage: 100
    
  pprof:
    enabled: true
    endpoint: "/debug/pprof"  # Admin only
```

## 🎯 Критерии успеха

- [ ] PerformanceMonitor собирает метрики
- [ ] Memory leak detection работает
- [ ] Slow requests логируются
- [ ] pprof endpoints доступны (admin only)
- [ ] WebUI dashboard отображает метрики
- [ ] Profile download работает
- [ ] Unit tests покрывают >90%
- [ ] Performance overhead <1%

## 📖 Документация

**Обновить:**
- `docs/PERFORMANCE.md` - секция "Monitoring in Production"
- `docs/TROUBLESHOOTING.md` - как использовать pprof
- `docs/API_DOCUMENTATION.md` - performance endpoints

**Добавить примеры:**
```bash
# CPU profile
curl -o cpu.prof http://localhost:8080/debug/pprof/profile?seconds=30

# Analyze
go tool pprof cpu.prof

# Memory heap
curl -o heap.prof http://localhost:8080/debug/pprof/heap

# Goroutine dump
curl http://localhost:8080/debug/pprof/goroutine?debug=1
```

---

**Оценка времени:** 6-8 часов  
**Сложность:** MEDIUM  
**Зависимости:** Опционально OBSERV-01  
**Блокирует:** Нет

