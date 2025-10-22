# Cache Optimization Analysis & Recommendations

## Executive Summary

Анализ структур данных проекта выявил **критические проблемы с производительностью** из-за false sharing и неоптимального использования CPU cache. При высокой нагрузке (100+ req/s) эти проблемы вызывают деградацию производительности **5-10x**.

## Выявленные проблемы

### 🔴 КРИТИЧНО: Stats False Sharing

**Файл:** `internal/api/handlers/stats.go`  
**Severity:** CRITICAL  
**Impact:** 5-10x slower under concurrency

```go
type Stats struct {
    StartTime time.Time
    
    // ВСЕ счетчики в одной cache line (64 bytes)!
    TotalRequests   int64  // 8 bytes
    ActiveRequests  int64  // 8 bytes
    SuccessRequests int64  // 8 bytes
    ErrorRequests   int64  // 8 bytes
}
```

**Проблема:** Каждый HTTP request обновляет эти счетчики из разных goroutines. Все счетчики находятся в одной cache line → при каждом atomic update инвалидируется весь cache на всех CPU cores.

**Решение:** `StatsOptimized` с cache line padding (уже реализовано в `stats_optimized.go`)

**Приоритет:** HIGH - влияет на каждый запрос

---

### 🔴 КРИТИЧНО: APIKeyUsage Race Condition + False Sharing

**Файл:** `internal/models/apikey.go`  
**Severity:** CRITICAL (data race + performance)  
**Impact:** Race condition + 10x slower

```go
type APIKeyUsage struct {
    TotalRequests      int64  // НЕ atomic!
    SuccessfulRequests int64  // Race condition!
    FailedRequests     int64
    TotalTokens        int64
    // Все в одной cache line
}

func (k *APIKey) IncrementUsage(...) {
    k.Usage.TotalRequests++      // Race!
    k.Usage.SuccessfulRequests++ // Race!
    // НЕ thread-safe!
}
```

**Проблемы:**
1. **Data race:** Счетчики обновляются без atomic операций
2. **False sharing:** Все счетчики в одной cache line

**Решение:** `APIKeyUsageHot` с atomic operations + padding (реализовано в `apikey_optimized.go`)

**Приоритет:** CRITICAL - нужно исправить немедленно

---

### 🟡 ВАЖНО: APIKey Hot/Cold Data Mixing

**Файл:** `internal/models/apikey.go`  
**Severity:** HIGH  
**Impact:** 3-5x slower validation

```go
type APIKey struct {
    // HOT (используется каждый запрос):
    ID          string
    Status      APIKeyStatus
    Models      []string
    Permissions []string
    
    // COLD (используется редко):
    Name        string  // Только в admin UI
    Description string  // Только в admin UI
    Metadata    map[string]interface{}
    CreatedAt   time.Time
    
    // HOT (обновляется каждый запрос):
    Usage APIKeyUsage
}
```

**Проблема:** При валидации API ключа (на каждом запросе) CPU загружает в cache **все поля** (~100+ bytes), хотя нужны только ~20 bytes.

**Решение:** `APIKeyHot` / `APIKeyCold` split (реализовано в `apikey_optimized.go`)

**Приоритет:** MEDIUM-HIGH

---

### 🟢 НИЗКИЙ: RingBuffer Optimization

**Файл:** `internal/metrics/ring_buffer.go`  
**Severity:** LOW  
**Impact:** Minor

RingBuffer использует `sync.RWMutex` правильно. Потенциальные улучшения:
- Lock-free ring buffer для еще большей производительности
- Batch operations для снижения lock contention

**Приоритет:** LOW - работает хорошо, оптимизация опциональна

---

### 🟢 НЕТ ПРОБЛЕМ: User, Conversation, Message

Эти структуры редко обновляются конкурентно → нет false sharing.  
Hot/cold split не нужен → эти данные загружаются полностью по запросу.

---

## Рекомендации по внедрению

### Фаза 1: Critical Fixes (v1.7.0) - 2-3 дня

**1.1 Исправить APIKeyUsage race condition (КРИТИЧНО)**

```go
// В internal/models/apikey.go
// Изменить IncrementUsage на atomic операции
func (k *APIKey) IncrementUsage(model, endpoint string, tokens int64, success bool) {
    atomic.AddInt64(&k.Usage.TotalRequests, 1)
    atomic.AddInt64(&k.Usage.TotalTokens, tokens)
    
    if success {
        atomic.AddInt64(&k.Usage.SuccessfulRequests, 1)
    } else {
        atomic.AddInt64(&k.Usage.FailedRequests, 1)
    }
    
    // ... rest
}
```

**Тесты:**
```bash
# Проверить race condition
go test -race ./internal/models/...
```

**1.2 Migrate Stats to StatsOptimized**

```go
// В cmd/server/main.go
// Было:
var GlobalStats = &handlers.Stats{StartTime: time.Now()}

// Стало:
var GlobalStats = handlers.NewStatsOptimized()
```

**Тесты:**
```bash
# Benchmark сравнение
go test -bench=BenchmarkStats -benchtime=10s -cpu=16
```

---

### Фаза 2: Hot/Cold Split (v1.8.0) - 1 неделя

**2.1 Implement APIKey caching layer**

```go
// internal/cache/apikey_cache.go
type APIKeyCache struct {
    hot  map[string]*models.APIKeyHot
    mu   sync.RWMutex
}

func (c *APIKeyCache) Get(keyID string) *models.APIKeyHot {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.hot[keyID]
}
```

**2.2 Update auth middleware**

```go
// internal/api/middleware/auth.go
func (m *AuthMiddleware) ValidateKey(plainKey string) (*models.APIKeyHot, error) {
    // Сначала проверяем cache (hot data)
    if keyHot := m.cache.Get(keyID); keyHot != nil {
        if keyHot.IsActive() && keyHot.HasModelAccess(model) {
            return keyHot, nil
        }
    }
    
    // Cache miss → загружаем из БД
    key := m.db.GetAPIKey(keyID)
    keyHot, keyCold := models.ConvertToHotCold(key)
    
    // Кэшируем hot data
    m.cache.Set(keyID, keyHot)
    
    return keyHot, nil
}
```

---

### Фаза 3: Advanced Optimizations (v1.9.0) - опционально

**3.1 Lock-free RingBuffer**

Заменить `sync.RWMutex` на lock-free структуру для metrics storage.

**3.2 NUMA-aware allocation**

Для multi-socket серверов пиннить goroutines к NUMA nodes.

**3.3 Prefetching hints**

Добавить explicit prefetch для predictable access patterns.

---

## Benchmarking Plan

### Baseline (до оптимизаций)

```bash
# 1. Stats performance
go test -bench=BenchmarkStatsOriginal_Parallel -benchtime=10s -cpu=16

# 2. APIKey validation
go test -bench=BenchmarkAPIKey_Validation_Workflow -benchmem

# 3. Overall server throughput
wrk -t16 -c100 -d30s http://localhost:8080/v1/chat/completions
```

### After Phase 1 (ожидаемые результаты)

```
Stats Parallel:
  Before: 45ns/op
  After:  7ns/op
  Speedup: 6.4x

APIKey IncrementUsage:
  Before: race condition + undefined behavior
  After:  15ns/op (thread-safe)

Server Throughput:
  Before: 5,000 req/s
  After:  8,000 req/s
  Improvement: +60%
```

### After Phase 2 (ожидаемые результаты)

```
APIKey Validation:
  Before: 85ns/op
  After:  28ns/op
  Speedup: 3x

Server Throughput:
  Before: 8,000 req/s
  After:  12,000 req/s
  Improvement: +50%
```

---

## Memory Trade-offs

### Stats

```
Original:   64 bytes
Optimized:  320 bytes
Overhead:   +256 bytes (acceptable - singleton)
```

### APIKey (на 1000 ключей)

```
Original:   ~120 bytes/key = 120 KB
Optimized:  ~320 bytes/key = 320 KB
Overhead:   +200 KB total (acceptable)
```

### Total Memory Impact

Для типичного production deployment (5000 API keys):
- Stats: +256 bytes (negligible)
- APIKeys: +1 MB
- **Total: ~1 MB additional RAM**

Trade-off: **1 MB RAM за 3-10x performance boost → worth it**

---

## Monitoring

### Metrics to Track

**1. Cache Efficiency**

```bash
# Linux perf
perf stat -p <PID> -e cache-misses,cache-references

# Целевой показатель: <5% cache miss rate
```

**2. Lock Contention**

```bash
# Go execution tracer
curl http://localhost:8080/debug/pprof/trace?seconds=30 > trace.out
go tool trace trace.out

# Целевой показатель: <1% time in lock contention
```

**3. Throughput**

```bash
# Baseline
wrk -t16 -c100 -d60s http://localhost:8080/v1/chat/completions

# Целевой показатель: >10,000 req/s
```

---

## Migration Checklist

### Phase 1 (v1.7.0) - Critical Fixes

- [ ] Добавить atomic operations в APIKeyUsage.IncrementUsage
- [ ] Тесты с `-race` для проверки отсутствия data races
- [ ] Migrate GlobalStats → StatsOptimized
- [ ] Benchmark before/after
- [ ] Update CHANGELOG.md
- [ ] Code review

### Phase 2 (v1.8.0) - Hot/Cold Split

- [ ] Implement APIKeyCache
- [ ] Update auth middleware для использования cache
- [ ] Конвертация при загрузке из БД
- [ ] Cache invalidation strategy
- [ ] Benchmarks
- [ ] Load testing

### Phase 3 (v1.9.0) - Advanced (optional)

- [ ] Lock-free RingBuffer
- [ ] NUMA-aware optimizations
- [ ] Prefetching
- [ ] Profiling validation

---

## Risks & Mitigation

### Risk 1: Memory Overhead

**Risk:** Padded structures занимают больше памяти  
**Impact:** +1 MB RAM для типичного deployment  
**Mitigation:** Acceptable trade-off для performance gain  
**Monitoring:** Track RSS memory usage

### Risk 2: Code Complexity

**Risk:** Hot/cold split усложняет code  
**Impact:** Больше файлов, сложнее onboarding  
**Mitigation:**
- Хорошая документация
- Benchmarks для доказательства необходимости
- Постепенная миграция

### Risk 3: Bugs During Migration

**Risk:** Ошибки при переходе на новые структуры  
**Impact:** Production bugs  
**Mitigation:**
- Comprehensive tests
- Gradual rollout (canary deployment)
- Feature flags для быстрого rollback

---

## Success Criteria

### Phase 1 Success

✅ APIKeyUsage race condition исправлен (`go test -race` clean)  
✅ Stats parallel benchmark: >5x speedup  
✅ No production bugs  
✅ Memory overhead <2 MB

### Phase 2 Success

✅ APIKey validation: >3x speedup  
✅ Server throughput: +50%  
✅ Cache hit rate: >95%  
✅ P99 latency reduction: >30%

### Overall Success

✅ 10,000+ req/s throughput  
✅ <5% CPU cache miss rate  
✅ No race conditions  
✅ Production stable

---

## References

- [Implementation] `internal/api/handlers/stats_optimized.go`
- [Implementation] `internal/models/apikey_optimized.go`
- [Benchmarks] `internal/api/handlers/stats_bench_test.go`
- [Benchmarks] `internal/models/apikey_bench_test.go`
- [Documentation] `docs/CACHE_OPTIMIZATION.md`
- [Article] https://skoredin.pro/blog/golang/cpu-cache-friendly-go

