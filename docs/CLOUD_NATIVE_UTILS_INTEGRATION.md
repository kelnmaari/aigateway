# Cloud Native Utils Integration Plan

> Integration of alternative libraries (sony/gobreaker, avast/retry-go) into AIGateway v3.0.8+

**Status:** ✅ Phase 1-3 Completed  
**Target Version:** v3.0.8  
**Priority:** High (Stability), Medium (Efficiency, Security), Low (Extensibility)

---

## 📦 Completed Integrations

### ✅ Phase 1: Circuit Breaker (sony/gobreaker/v2)

**Статус:** ✅ Завершено

**Реализовано:**
- ✅ `internal/huggingface/client.go` - Circuit breaker для Hugging Face API
- ✅ `internal/providers/vllm_provider.go` - Circuit breaker для vLLM health checks
- ✅ Tests: `internal/huggingface/client_circuit_breaker_test.go` (4 tests)
- ✅ Библиотека: `github.com/sony/gobreaker/v2`

**Настройки:**
- **HuggingFace API:**
  - MaxRequests: 3 (concurrent half-open requests)
  - Timeout: 2 minutes (reset interval)
  - ReadyToTrip: 5 consecutive failures
  
- **vLLM Provider:**
  - MaxRequests: 2
  - Timeout: 1 minute
  - ReadyToTrip: 3 consecutive failures

---

### ✅ Phase 2: Retry Mechanism (avast/retry-go/v4)

**Статус:** ✅ Завершено

**Реализовано:**
- ✅ `internal/huggingface/downloader.go` - Retry-go для file operations
- ✅ Exponential backoff: 200ms → 5s
- ✅ GC между попытками (Windows file handle fix)
- ✅ Библиотека: `github.com/avast/retry-go/v4`

**Настройки:**
- Attempts: 10
- Delay: 200ms
- MaxDelay: 5s
- DelayType: Exponential backoff

---

### ✅ Phase 3: Efficiency - Sharding (Custom Implementation)

**Статус:** ✅ Завершено

**Реализовано:**

#### Sharding Layer (`internal/cache/redis/sharded_cache.go`)
- ✅ In-memory sharded cache над Redis (32 shards)
- ✅ Cache line padding для предотвращения false sharing
- ✅ Memory → Redis fallback pattern
- ✅ Exponential cleanup (30s interval)
- ✅ Per-shard metrics & access counts
- ✅ Tests: `sharded_cache_test.go` (5 tests + 2 benchmarks)

**Характеристики:**
- 32 shards (power of 2, FNV-1a hash)
- 1 minute in-memory TTL (configurable)
- Concurrent-safe RWMutex per shard
- Auto-population on Redis hit

**API:**
```go
cache := NewShardedCache(redis, ShardedCacheConfig{
    ShardCount: 32,
    TTL:        1 * time.Minute,
    Enabled:    true,
}, logger)

cache.Get(ctx, "key", &dest)  // Memory → Redis fallback
cache.Set(ctx, "key", value, ttl)
cache.Delete(ctx, "key")
cache.GetStats() // Shard statistics
```

---

### ✅ Phase 4: Efficiency - Worker Pool (Custom Implementation)

**Статус:** ✅ Завершено

**Реализовано:**

#### Worker Pool (`internal/concurrent/pool.go`)
- ✅ Bounded worker pool с task queue
- ✅ Batch processing utilities
- ✅ Parallel map implementation
- ✅ Graceful shutdown с timeout
- ✅ Padded metrics (false sharing prevention)
- ✅ Tests: `pool_test.go` (6 tests + 2 benchmarks)

**API:**
```go
// Worker Pool
pool := NewWorkerPool(PoolConfig{Workers: 10, QueueSize: 100})
pool.Submit(Task{...})
pool.GetMetrics() // TotalTasks, CompletedTasks, FailedTasks

// Batch Processing
bp := NewBatchProcessor[int](batchSize, pool)
bp.Process(ctx, items, func(item) error {...})

// Parallel Map
results, err := ParallelMap(ctx, items, fn, workers)
```

**Benefits:**
- ✅ Bounded concurrency для resource control
- ✅ Generic types для type-safety (Go 1.25)
- ✅ Graceful shutdown с timeout
- ✅ Real-time metrics tracking

---

### ✅ Phase 5: Security - Health Probes (Custom Implementation)

**Статус:** ✅ Завершено

**Реализовано:**

#### Health Probes (`internal/health/probes.go`)
- ✅ Liveness probe (process uptime)
- ✅ Readiness probe (dependencies check)
- ✅ Parallel probe execution
- ✅ Detailed health status reporting
- ✅ Tests: `probes_test.go` (7 tests + 1 benchmark)

**Integration:**
- `internal/api/router/router.go` - K8s probe endpoints

**Endpoints:**
- `GET /healthz/live` - Liveness probe (always healthy if running)
- `GET /healthz/ready` - Readiness probe (dependencies check)
- `GET /healthz/status` - Detailed health status

**Registered Probes:**
- Database probe (SQLite/PostgreSQL)
- Redis probe (cache availability)
- Yzma probe (model inference engine)

**Kubernetes Manifest Example:**
```yaml
livenessProbe:
  httpGet:
    path: /healthz/live
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## 📊 Summary

**Phases Completed:** 5/7

| Phase | Status | Library | Files | Tests |
|-------|--------|---------|-------|-------|
| Circuit Breaker | ✅ | sony/gobreaker/v2 | 2 | 4 |
| Retry Mechanism | ✅ | avast/retry-go/v4 | 1 | - |
| Sharding | ✅ | Custom | 2 | 5+2 |
| Worker Pool | ✅ | Custom | 2 | 6+2 |
| Health Probes | ✅ | Custom | 3 | 7+1 |
| AES-GCM Encryption | 📋 | TBD | - | - |
| Extensibility | 📋 | Research | - | - |

**Total Tests Added:** 22 unit tests + 5 benchmarks

---

## 📦 Original Cloud Native Utils Reference

### 1. ✅ **stability** (High Priority)

> **Note:** Replaced with `github.com/sony/gobreaker/v2`

#### Use Cases

**A. Circuit Breaker для External HTTP Providers**

**Note:** v3.0+ использует yzma для локального inference (без Ollama). Circuit breaker нужен только для:
- HuggingFace API (загрузка моделей)
- External providers (vLLM, если настроен)
- Redis (опционально)

```go
// internal/huggingface/client.go
import "github.com/andygeiss/cloud-native-utils/stability"

type Client struct {
    token          string
    httpClient     *http.Client
    circuitBreaker *stability.CircuitBreaker  // NEW
    logger         *logrus.Logger
}

func NewClient(token string, logger *logrus.Logger) *Client {
    return &Client{
        token: token,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        circuitBreaker: stability.NewCircuitBreaker(
            5,                // maxFailures: 5 consecutive failures (HF может временно быть недоступен)
            2 * time.Minute,  // timeout: 2 minutes перед повторной попыткой
        ),
        logger: logger,
    }
}

func (c *Client) GetRepoInfo(ctx context.Context, repoID string) (*RepoInfo, error) {
    var info *RepoInfo
    
    err := c.circuitBreaker.Execute(func() error {
        url := fmt.Sprintf("https://huggingface.co/api/models/%s", repoID)
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return err
        }
        
        if c.token != "" {
            req.Header.Set("Authorization", "Bearer "+c.token)
        }
        
        resp, err := c.httpClient.Do(req)
        if err != nil {
            return fmt.Errorf("HuggingFace API error: %w", err)
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
        }
        
        return json.NewDecoder(resp.Body).Decode(&info)
    })
    
    return info, err
}
```

```go
// internal/providers/vllm_provider.go (когда будет реализован)
import "github.com/andygeiss/cloud-native-utils/stability"

type VLLMProvider struct {
    name           string
    baseURL        string
    client         *http.Client
    circuitBreaker *stability.CircuitBreaker  // NEW
}

func NewVLLMProvider(name, baseURL string) *VLLMProvider {
    return &VLLMProvider{
        name:    name,
        baseURL: baseURL,
        client:  &http.Client{Timeout: 30 * time.Second},
        circuitBreaker: stability.NewCircuitBreaker(
            3,                // maxFailures
            1 * time.Minute,  // timeout
        ),
    }
}

func (p *VLLMProvider) HealthCheck(ctx context.Context) error {
    return p.circuitBreaker.Execute(func() error {
        req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/health", nil)
        if err != nil {
            return err
        }
        
        resp, err := p.client.Do(req)
        if err != nil {
            return fmt.Errorf("vLLM health check failed: %w", err)
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("vLLM unhealthy: status %d", resp.StatusCode)
        }
        
        return nil
    })
}
```

**Benefits:**
- ✅ Предотвращает cascade failures при сбое HuggingFace API
- ✅ Защита от timeout при загрузке больших моделей
- ✅ Автоматическое восстановление после temporary outages
- ✅ Ready for future external providers (vLLM, Replicate, etc.)

**B. Retry Mechanism для HuggingFace Downloads**

```go
// internal/huggingface/downloader.go
import "github.com/andygeiss/cloud-native-utils/stability"

func (d *Downloader) DownloadFile(ctx context.Context, download *models.ModelDownload) error {
    // Replace manual retry logic with stability.Retry
    return stability.Retry(
        5,                // maxAttempts: 5 retries
        2 * time.Second,  // initialDelay: exponential backoff starting from 2s
        func() error {
            return d.downloadFileInternal(ctx, download)
        },
    )
}

// Remove old retryRename() - use stability.Retry instead
func (d *Downloader) renamePartFile(partPath, destPath string) error {
    return stability.Retry(10, 100*time.Millisecond, func() error {
        return os.Rename(partPath, destPath)
    })
}
```

**Benefits:**
- ✅ Устраняет дублирование retry логики
- ✅ Exponential backoff из коробки
- ✅ Cleaner код

**C. Throttling для Rate Limiting**

```go
// internal/api/middleware/advanced_rate_limit.go
import "github.com/andygeiss/cloud-native-utils/stability"

type RateLimiter struct {
    limiter   *ratelimit.Limiter
    throttle  *stability.Throttle  // NEW: per-IP throttling
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    return &RateLimiter{
        limiter: ratelimit.New(requestsPerSecond),
        throttle: stability.NewThrottle(
            time.Second,      // interval
            requestsPerSecond, // maxCalls
        ),
    }
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        
        // Throttle by IP
        if !rl.throttle.Allow(clientIP) {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded",
                "retry_after": 1,
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

**Benefits:**
- ✅ Per-client throttling (защита от spamming)
- ✅ Дополнительный слой защиты к Redis rate limiter

---

### 2. ✅ **efficiency** (Medium Priority)

#### Use Cases

**A. Concurrent Processing для Background Workers**

```go
// internal/cache/redis/background_sync.go
import "github.com/andygeiss/cloud-native-utils/efficiency"

func (w *StatsWorker) updateStats() {
    stats := w.provider.GetStats()
    
    // Concurrent processing of metrics
    items := []interface{}{stats.TotalRequests, stats.TotalTokens, stats.ErrorRate}
    
    efficiency.ProcessConcurrently(items, func(item interface{}) error {
        metric := item.(int64)
        return w.manager.Cache.SetJSON(context.Background(), 
            fmt.Sprintf("metric:%d", metric), 
            metric, 
            10*time.Second,
        )
    })
}
```

**B. Sharding для Redis Key Distribution**

```go
// internal/cache/redis/cache.go
import "github.com/andygeiss/cloud-native-utils/efficiency"

type CacheService struct {
    client *Client
    logger *logrus.Logger
    shards *efficiency.ShardMap[string, interface{}]  // NEW
}

func NewCacheService(client *Client, logger *logrus.Logger) *CacheService {
    return &CacheService{
        client: client,
        logger: logger,
        shards: efficiency.NewShardMap[string, interface{}](32), // 32 shards
    }
}

func (s *CacheService) GetJSON(ctx context.Context, key string, dest interface{}) error {
    // In-memory shard cache before Redis lookup
    if value, ok := s.shards.Get(key); ok {
        s.logger.Debug("Cache hit from in-memory shard")
        return json.Unmarshal(value.([]byte), dest)
    }
    
    // Fallback to Redis
    return s.client.Get(ctx, key, dest)
}
```

**Benefits:**
- ✅ Reduced Redis load (local cache layer)
- ✅ Concurrent-safe sharding
- ✅ Better performance для hot keys

**C. Channel Utilities для Streaming**

```go
// internal/api/handlers/yzma_handler.go
import "github.com/andygeiss/cloud-native-utils/efficiency"

func (h *YzmaHandler) HandleChatCompletionStream(c *gin.Context) {
    // Generate tokens channel
    tokensCh := make(chan string)
    
    // Create read-only channel for consumers
    readOnlyCh := efficiency.ReadOnly(tokensCh)
    
    // Split stream: logging + client response
    logCh, clientCh := efficiency.Split(readOnlyCh)
    
    // Concurrent logging
    go func() {
        for token := range logCh {
            h.logger.Debug("Token generated: ", token)
        }
    }()
    
    // Stream to client
    for token := range clientCh {
        c.SSEvent("message", token)
        c.Writer.Flush()
    }
}
```

---

### 3. ✅ **security** (Medium Priority)

#### Use Cases

**A. Liveness & Readiness Probes для Kubernetes**

```go
// cmd/server/main.go
import "github.com/andygeiss/cloud-native-utils/security"

func setupHealthProbes(router *gin.Engine, deps *dependencies) {
    // Liveness: process alive
    router.GET("/healthz/live", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "alive"})
    })
    
    // Readiness: all dependencies ready
    router.GET("/healthz/ready", func(c *gin.Context) {
        ready := true
        errors := []string{}
        
        // Check Redis
        if err := deps.redisManager.Ping(c.Request.Context()); err != nil {
            ready = false
            errors = append(errors, fmt.Sprintf("redis: %v", err))
        }
        
        // Check SQLite
        if err := deps.db.Ping(); err != nil {
            ready = false
            errors = append(errors, fmt.Sprintf("database: %v", err))
        }
        
        // Check yzma
        if deps.yzmaClient != nil && !deps.yzmaClient.IsInitialized() {
            ready = false
            errors = append(errors, "yzma: not initialized")
        }
        
        if ready {
            c.JSON(http.StatusOK, gin.H{"status": "ready"})
        } else {
            c.JSON(http.StatusServiceUnavailable, gin.H{
                "status": "not ready",
                "errors": errors,
            })
        }
    })
}
```

**Kubernetes manifest:**

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aigateway
spec:
  template:
    spec:
      containers:
      - name: aigateway
        image: aigateway:v3.0.8
        livenessProbe:
          httpGet:
            path: /healthz/live
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /healthz/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

**B. AES-GCM Encryption для Sensitive Configuration**

```go
// internal/config/encryption.go
import "github.com/andygeiss/cloud-native-utils/security"

// Encrypt sensitive config values (API keys, DB passwords)
func (c *Config) EncryptSecrets(encryptionKey string) error {
    key := []byte(encryptionKey) // 32 bytes for AES-256
    
    // Encrypt HuggingFace token
    if c.HuggingFace.Token != "" {
        encrypted, err := security.Encrypt([]byte(c.HuggingFace.Token), key)
        if err != nil {
            return fmt.Errorf("failed to encrypt HF token: %w", err)
        }
        c.HuggingFace.Token = base64.StdEncoding.EncodeToString(encrypted)
    }
    
    return nil
}

func (c *Config) DecryptSecrets(encryptionKey string) error {
    key := []byte(encryptionKey)
    
    if c.HuggingFace.Token != "" {
        encrypted, _ := base64.StdEncoding.DecodeString(c.HuggingFace.Token)
        decrypted, err := security.Decrypt(encrypted, key)
        if err != nil {
            return fmt.Errorf("failed to decrypt HF token: %w", err)
        }
        c.HuggingFace.Token = string(decrypted)
    }
    
    return nil
}
```

**Usage:**

```bash
# Set encryption key via env
export CONFIG_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# Encrypt config on first run
aigateway --config configs/production.yaml --encrypt-secrets

# Config stored with encrypted values:
# huggingface:
#   token: "A3B5c9D7e8F... (encrypted)"
```

---

### 4. ⚠️ **extensibility** (Low Priority - Research)

#### Use Cases

**A. Plugin System для Custom Providers**

```go
// internal/providers/plugin_loader.go
import "github.com/andygeiss/cloud-native-utils/extensibility"

type ProviderPlugin interface {
    Name() string
    HealthCheck(ctx context.Context) error
    ListModels(ctx context.Context) ([]string, error)
}

func LoadProviderPlugin(pluginPath string) (ProviderPlugin, error) {
    plugin, err := extensibility.LoadPlugin(pluginPath, "NewProvider")
    if err != nil {
        return nil, err
    }
    
    providerFunc := plugin.(func() ProviderPlugin)
    return providerFunc(), nil
}
```

**Custom provider plugin example:**

```go
// plugins/vllm_provider.go
package main

import "context"

type VLLMProvider struct {
    endpoint string
}

func NewProvider() interface{} {
    return &VLLMProvider{endpoint: "http://localhost:8000"}
}

func (p *VLLMProvider) Name() string {
    return "vllm"
}

func (p *VLLMProvider) HealthCheck(ctx context.Context) error {
    // Implementation
    return nil
}

func (p *VLLMProvider) ListModels(ctx context.Context) ([]string, error) {
    // Implementation
    return []string{"model1", "model2"}, nil
}
```

**Build & Load:**

```bash
# Build plugin
go build -buildmode=plugin -o plugins/vllm.so plugins/vllm_provider.go

# Load at runtime
aigateway --config configs/dev.yaml --plugins plugins/vllm.so
```

**⚠️ Limitations:**
- Go версия plugin должна совпадать с main binary
- Работает только на Linux/macOS (Windows не поддерживает Go plugins)
- CGo required (усложняет deployment)

**Alternative:** Use gRPC/HTTP-based plugins вместо Go plugins

---

## 📋 Implementation Roadmap

### Phase 1: Stability (v3.0.8) - HIGH PRIORITY

**Sprint 1 (Week 1):**
- [ ] Add dependency: `go get github.com/andygeiss/cloud-native-utils/stability`
- [ ] Implement circuit breaker для HuggingFace API (critical path)
- [ ] Implement circuit breaker для vLLM provider (future-proofing)
- [ ] Add tests для circuit breaker scenarios

**Sprint 2 (Week 2):**
- [ ] Replace manual retry logic с `stability.Retry` в downloader
- [ ] Replace manual retry logic в file rename operations
- [ ] Add configurable retry settings в config.yaml
- [ ] Integration tests для retry scenarios

**Sprint 3 (Week 3):**
- [ ] Implement throttling middleware using `stability.Throttle`
- [ ] Add per-IP throttling to rate limiter
- [ ] Load tests для throttling effectiveness
- [ ] Documentation update

### Phase 2: Efficiency (v3.0.9) - MEDIUM PRIORITY

**Sprint 4 (Week 4):**
- [ ] Add dependency: `go get github.com/andygeiss/cloud-native-utils/efficiency`
- [ ] Implement sharding для Redis cache layer
- [ ] Benchmark: Redis throughput improvement
- [ ] Concurrent processing для background workers

**Sprint 5 (Week 5):**
- [ ] Channel utilities для streaming responses
- [ ] Concurrent model loading optimization
- [ ] Performance benchmarks
- [ ] Documentation update

### Phase 3: Security (v3.0.10) - MEDIUM PRIORITY

**Sprint 6 (Week 6):**
- [ ] Add dependency: `go get github.com/andygeiss/cloud-native-utils/security`
- [ ] Implement liveness/readiness probes
- [ ] Kubernetes manifest with health checks
- [ ] Test deployment to k8s cluster

**Sprint 7 (Week 7):**
- [ ] Implement AES-GCM config encryption
- [ ] CLI flag для encrypt/decrypt secrets
- [ ] Secure key management documentation
- [ ] Production deployment guide

### Phase 4: Extensibility (Research) - LOW PRIORITY

**Sprint 8 (Week 8):**
- [ ] Research Go plugins limitations
- [ ] Prototype vLLM plugin
- [ ] Evaluate: Go plugins vs gRPC-based approach
- [ ] Decision: implement or skip

---

## 🎯 Success Metrics

### Stability
- **Circuit Breaker Effectiveness:**
  - Reduced HuggingFace API failures impact: <1% (target: 0%)
  - Recovery time: <120s after HF API comes back online
  - vLLM provider resilience: automatic failover
  
- **Retry Reliability:**
  - Download success rate: >99.5% (up from ~95%)
  - Reduced manual interventions: -80%

### Efficiency
- **Redis Performance:**
  - Cache hit rate: >90% (with sharding layer)
  - Redis queries reduction: -50% (via local shards)
  
- **Concurrent Processing:**
  - Background worker throughput: +30%
  - Model loading time: -20% (parallel operations)

### Security
- **Kubernetes Readiness:**
  - Zero-downtime deployments
  - Fast startup detection: <5s
  
- **Config Security:**
  - No plaintext secrets в configs
  - Automated key rotation support

---

## 📦 Dependencies

```go
// go.mod additions
require (
    github.com/andygeiss/cloud-native-utils v0.2.8
)
```

**Size:** ~150KB additional binary size  
**License:** MIT (compatible)  
**Go Version:** 1.23+ (matches project)

---

## 🚨 Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Circuit breaker too aggressive | Service unavailable false positives | Tunable thresholds (config) |
| Sharding memory overhead | Increased RAM usage | Configurable shard count (16-64) |
| Plugin instability | Crashes from bad plugins | Sandbox plugins, recovery mechanisms |
| Dependency maintenance | Abandoned library | Fork if needed, simple code |

---

## ✅ Next Steps

1. **Add dependency:**
   ```bash
   go get github.com/andygeiss/cloud-native-utils/stability
   ```

2. **Start with circuit breaker для HuggingFace:**
   - File: `internal/huggingface/client.go`
   - Functions: `GetRepoInfo()`, `ListFiles()`, `DownloadFile()`
   
3. **Test resilience:**
   - Simulate HuggingFace API timeout/downtime
   - Verify circuit opens after 5 failures
   - Verify recovery after 2 minutes
   - Test concurrent download resilience

4. **Iterate through roadmap phases**

---

**Ready to start implementation?** Begin with Phase 1: Stability - Circuit Breaker.

