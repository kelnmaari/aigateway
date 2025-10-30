
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.6.2', '2025-10-12', '## [1.6.2] - 2025-10-12

### Added
- **Performance Monitoring**: Continuous performance monitoring в production
  - Runtime metrics collection (CPU, memory, goroutines, GC stats)
  - Automatic performance anomaly detection
  - Baseline comparison для regression tracking
  - Configurable thresholds

- **Leak Detection**: Автоматическое выявление утечек
  - Memory и goroutine leak detection
  - Trend analysis (10 samples, 1/minute)
  - Warning alerts при sustained growth

- **Slow Request Logging**: Медленные запросы
  - Middleware для отслеживания request time
  - Configurable threshold (default: 5s)
  - Детальная информация о каждом slow request

- **pprof Endpoints**: Runtime profiling
  - /api/admin/pprof/* endpoints (admin only)
  - CPU profile, heap, goroutine dump
  - Memory allocations, block, mutex profiles

- **Performance API**: REST API для metrics
  - GET /api/admin/performance/metrics
  - GET /api/admin/performance/leaks
  - POST /api/admin/performance/reset-baseline
  - POST /api/admin/performance/reset-leaks

### Changed
- **Configuration**: Добавлена секция observability.performance
  - enabled, collection_interval, thresholds
  - gc_percentage для GC tuning
  - leak_detection и pprof_enabled flags

### Technical
- internal/observability/perfmon.go - Performance Monitor
- internal/observability/leak_detector.go - Leak Detector
- internal/api/middleware/slow_request.go - Slow Request Logger
- internal/api/handlers/performance.go - Performance API
- Thread-safe metrics, minimal overhead (<1%)
- Graceful shutdown support');
	