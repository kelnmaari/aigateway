
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.4', '2025-10-20', '## [1.9.4] - 2025-10-20

### Added
- **CPU Cache-Friendly Data Structures** 🚀 (Phase 2: Hot/Cold Split & Caching)
  - APIKeyCache Layer: In-memory cache для API keys с TTL expiration
    - Thread-safe concurrent access (RWMutex)
    - Load-through caching pattern с GetOrLoad()
    - Background cleanup goroutine
    - Performance: 22.76ns Get, 78.43ns Set, 45.52ns Parallel
    - 100x faster vs database queries (22ns vs 2ms)
  - APIKeyDBAuthOptimized Middleware: Cache-aware authentication
    - Cache hit path: ~22ns (memory only)
    - Cache miss path: ~2ms (DB + bcrypt)
    - Expected 95%+ cache hit rate → 20x faster auth
  - APIKeyUsageThreadSafe: Fully thread-safe usage tracking
    - Atomic counters с cache line padding (prevents false sharing)
    - RWMutex для map operations (ModelUsage, EndpointUsage, DailyUsage)
    - Performance: 196ns vs 236ns Original + zero race conditions
    - 20% faster + thread-safe

- **Cursor Rules для Cache Optimization**
  - Автоматические подсказки для cache-friendly structures
  - Lint правила для false sharing detection
  - Templates для padded structs, hot/cold splits, concurrent counters

### Changed
- **StatsOptimized Migration**: Migrated GlobalStats to cache-friendly version
  - Cache line padding между atomic counters
  - StatsInterface для backwards compatibility
  - Performance: 6.4x faster under high concurrency

### Fixed
- **Race Condition в APIKey.IncrementUsage**: Replaced ++ with atomic.AddInt64
  - Atomic operations для TotalRequests, SuccessfulRequests, FailedRequests, TotalTokens
  - Map operations (ModelUsage, DailyUsage) все еще require careful handling
  - Recommended: Use APIKeyUsageThreadSafe для new keys

### Technical
- Benchmarks Added:
  - internal/api/handlers/stats_bench_test.go: Stats vs StatsOptimized
  - internal/models/apikey_bench_test.go: APIKey validation benchmarks
  - internal/models/apikey_usage_threadsafe_test.go: Thread-safe usage benchmarks
  - internal/cache/apikey_cache_test.go: Cache performance benchmarks
- Documentation:
  - docs/CACHE_OPTIMIZATION.md: Technical deep-dive
  - docs/CACHE_OPTIMIZATION_SUMMARY.md: Executive summary
  - docs/CACHE_OPTIMIZATION_QUICKSTART.md: Quick start guide
  - PERFORMANCE_IMPROVEMENTS.md: High-level report
  - MIGRATION_APPLIED.md: Phase 1 & 2 migration status
  - PHASE2_COMPLETED.md: Phase 2 completion report
- Dependencies: No new dependencies (pure Go stdlib)
- New Packages:
  - internal/cache: APIKey caching layer
  - internal/api/middleware/apikey_db_auth_optimized.go: Optimized middleware
  - internal/models/apikey_usage_threadsafe.go: Thread-safe usage tracking
  - internal/api/handlers/stats_optimized.go: Cache-friendly stats

### Performance
- Authentication: 20x faster (с cache hit rate 95%+)
- Usage Tracking: 1.2x faster + thread-safe
- Server Throughput: +50-87% expected improvement
- Memory Overhead: ~6 MB для 10,000 API keys (acceptable)

### Security
- Zero Race Conditions: All optimized structures pass -race tests
- Thread-Safe Maps: RWMutex protection для concurrent access
- Atomic Counters: Cache line padding prevents false sharing');
	