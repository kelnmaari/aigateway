# RATE-02: Advanced Rate Limiting

**Версия:** 1.10.0  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 8-10 часов

## Описание

Enhanced rate limiting система с per-tenant, per-model limits, sliding window algorithm, и rate limit headers в responses.

## Проблема

После базового rate limiting (AUTH-03):
- Только global rate limits per API key
- Нет per-tenant rate limiting
- Нет per-model rate limiting
- Fixed window (может быть обойден)
- Нет rate limit headers в response

## Решение

Advanced rate limiting с flexible configuration и sliding window.

### Архитектура

```
Request → Check Rate Limit → [Allow/Deny] → Response with Headers
              ↓
    Per-Key, Per-Tenant, Per-Model Limits
              ↓
         Sliding Window Algorithm
```

## Технические детали

### Enhanced Rate Limit Model

```go
// internal/models/rate_limit.go (extend AUTH-03)

type RateLimitConfig struct {
    ID       string `json:"id" db:"id"`
    Name     string `json:"name" db:"name"`
    Scope    string `json:"scope" db:"scope"` // "global", "tenant", "user", "api_key", "model"
    TargetID string `json:"target_id" db:"target_id"`
    
    // Rate limits
    RequestsPerSecond *int `json:"requests_per_second,omitempty" db:"requests_per_second"`
    RequestsPerMinute *int `json:"requests_per_minute,omitempty" db:"requests_per_minute"`
    RequestsPerHour   *int `json:"requests_per_hour,omitempty" db:"requests_per_hour"`
    RequestsPerDay    *int `json:"requests_per_day,omitempty" db:"requests_per_day"`
    
    // Burst allowance
    BurstSize *int `json:"burst_size,omitempty" db:"burst_size"`
    
    // Model-specific (optional)
    ModelName *string `json:"model_name,omitempty" db:"model_name"`
    
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
```

### Database Schema (extend AUTH-03)

```sql
ALTER TABLE rate_limits ADD COLUMN scope TEXT DEFAULT 'api_key';
ALTER TABLE rate_limits ADD COLUMN target_id TEXT; -- user_id, tenant_id, model_name, etc.
ALTER TABLE rate_limits ADD COLUMN model_name TEXT;
ALTER TABLE rate_limits ADD COLUMN burst_size INTEGER;

CREATE INDEX idx_rate_limits_scope_target ON rate_limits(scope, target_id);
CREATE INDEX idx_rate_limits_model ON rate_limits(model_name) WHERE model_name IS NOT NULL;
```

### Sliding Window Rate Limiter

```go
// internal/services/ratelimit/sliding_window.go

type SlidingWindowLimiter struct {
    db    storage.Database
    cache *sync.Map // target_id -> []timestamp
    mu    sync.RWMutex
}

func NewSlidingWindowLimiter(db storage.Database) *SlidingWindowLimiter {
    return &SlidingWindowLimiter{
        db:    db,
        cache: &sync.Map{},
    }
}

func (l *SlidingWindowLimiter) CheckLimit(
    ctx context.Context,
    targetID string,
    limit int,
    window time.Duration,
) (allowed bool, remaining int, resetAt time.Time, err error) {
    now := time.Now()
    windowStart := now.Add(-window)
    
    // Get or create timestamps list
    value, _ := l.cache.LoadOrStore(targetID, &[]time.Time{})
    timestamps := value.(*[]time.Time)
    
    l.mu.Lock()
    defer l.mu.Unlock()
    
    // Remove old timestamps (outside window)
    newTimestamps := make([]time.Time, 0)
    for _, ts := range *timestamps {
        if ts.After(windowStart) {
            newTimestamps = append(newTimestamps, ts)
        }
    }
    
    // Check if limit exceeded
    currentCount := len(newTimestamps)
    if currentCount >= limit {
        return false, 0, windowStart.Add(window), nil
    }
    
    // Add current request
    newTimestamps = append(newTimestamps, now)
    *timestamps = newTimestamps
    l.cache.Store(targetID, timestamps)
    
    remaining = limit - len(newTimestamps)
    resetAt = windowStart.Add(window)
    
    return true, remaining, resetAt, nil
}
```

### Advanced Rate Limit Service

```go
// internal/services/ratelimit/advanced.go

type AdvancedRateLimiter struct {
    db              storage.Database
    slidingWindow   *SlidingWindowLimiter
    logger          *logrus.Logger
}

func (r *AdvancedRateLimiter) CheckRateLimit(
    ctx context.Context,
    userID string,
    tenantID *string,
    apiKeyID *string,
    modelName string,
) (*RateLimitResult, error) {
    // Priority: API Key → User → Tenant → Model → Global
    
    // 1. Check API key rate limit
    if apiKeyID != nil {
        result, err := r.checkLimit(ctx, "api_key", *apiKeyID, nil)
        if err != nil || !result.Allowed {
            return result, err
        }
    }
    
    // 2. Check user rate limit
    result, err := r.checkLimit(ctx, "user", userID, nil)
    if err != nil || !result.Allowed {
        return result, err
    }
    
    // 3. Check tenant rate limit
    if tenantID != nil {
        result, err := r.checkLimit(ctx, "tenant", *tenantID, nil)
        if err != nil || !result.Allowed {
            return result, err
        }
    }
    
    // 4. Check model-specific rate limit
    result, err = r.checkLimit(ctx, "model", modelName, &modelName)
    if err != nil || !result.Allowed {
        return result, err
    }
    
    // 5. Check global rate limit
    result, err = r.checkLimit(ctx, "global", "all", nil)
    if err != nil || !result.Allowed {
        return result, err
    }
    
    return result, nil
}

func (r *AdvancedRateLimiter) checkLimit(
    ctx context.Context,
    scope string,
    targetID string,
    modelName *string,
) (*RateLimitResult, error) {
    // Get rate limit config
    config, err := r.db.GetRateLimitConfig(ctx, scope, targetID, modelName)
    if err != nil {
        // No limit configured = allow
        return &RateLimitResult{Allowed: true}, nil
    }
    
    // Check per-second limit
    if config.RequestsPerSecond != nil {
        allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
            ctx,
            fmt.Sprintf("%s:%s:second", scope, targetID),
            *config.RequestsPerSecond,
            time.Second,
        )
        if err != nil || !allowed {
            return &RateLimitResult{
                Allowed:   allowed,
                Remaining: remaining,
                ResetAt:   resetAt,
                Limit:     *config.RequestsPerSecond,
                Window:    "second",
            }, err
        }
    }
    
    // Check per-minute limit
    if config.RequestsPerMinute != nil {
        allowed, remaining, resetAt, err := r.slidingWindow.CheckLimit(
            ctx,
            fmt.Sprintf("%s:%s:minute", scope, targetID),
            *config.RequestsPerMinute,
            time.Minute,
        )
        if err != nil || !allowed {
            return &RateLimitResult{
                Allowed:   allowed,
                Remaining: remaining,
                ResetAt:   resetAt,
                Limit:     *config.RequestsPerMinute,
                Window:    "minute",
            }, err
        }
    }
    
    // Similar checks for hour/day...
    
    return &RateLimitResult{Allowed: true}, nil
}

type RateLimitResult struct {
    Allowed   bool
    Remaining int
    ResetAt   time.Time
    Limit     int
    Window    string
}
```

### Middleware with Headers

```go
// internal/api/middleware/rate_limit.go (update from AUTH-03)

func AdvancedRateLimitMiddleware(limiter *ratelimit.AdvancedRateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, _ := c.Get("user_id")
        tenantID, _ := c.Get("tenant_id")
        apiKeyID, _ := c.Get("api_key_id")
        
        // Extract model from request (if chat endpoint)
        modelName := extractModelFromRequest(c)
        
        // Check rate limit
        result, err := limiter.CheckRateLimit(
            c.Request.Context(),
            userID.(string),
            stringPtrOrNil(tenantID),
            stringPtrOrNil(apiKeyID),
            modelName,
        )
        
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "rate limit check failed"})
            c.Abort()
            return
        }
        
        // Set rate limit headers (RFC 6585)
        c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
        c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
        c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
        
        if !result.Allowed {
            c.Header("Retry-After", strconv.Itoa(int(time.Until(result.ResetAt).Seconds())))
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error":   "rate limit exceeded",
                "message": fmt.Sprintf("Rate limit of %d requests per %s exceeded", result.Limit, result.Window),
                "retry_after": result.ResetAt.Unix(),
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### Configuration

```yaml
# config.yaml
rate_limiting:
  enabled: true
  algorithm: "sliding_window" # or "fixed_window"
  
  # Default limits (if not configured per scope)
  defaults:
    requests_per_second: 10
    requests_per_minute: 100
    requests_per_hour: 1000
    requests_per_day: 10000
    burst_size: 20
```

### Admin UI

**Admin Panel → Rate Limits:**

```html
<div id="rate-limits-tab" class="tab-pane">
  <h3>Rate Limiting Configuration</h3>
  
  <button onclick="createRateLimit()">Create Rate Limit</button>
  
  <!-- Rate Limits List -->
  <table class="rate-limits-table">
    <thead>
      <tr>
        <th>Scope</th>
        <th>Target</th>
        <th>Model</th>
        <th>Limits</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody id="rate-limits-list">
      <tr>
        <td>Tenant</td>
        <td>engineering</td>
        <td>-</td>
        <td>100/min, 1000/hour</td>
        <td><button onclick="editRateLimit()">Edit</button></td>
      </tr>
      <tr>
        <td>Model</td>
        <td>-</td>
        <td>gpt-oss:latest</td>
        <td>50/min</td>
        <td><button onclick="editRateLimit()">Edit</button></td>
      </tr>
    </tbody>
  </table>
  
  <!-- Create/Edit Modal -->
  <div id="rate-limit-modal" class="modal">
    <h3>Configure Rate Limit</h3>
    
    <select id="rl-scope">
      <option value="global">Global</option>
      <option value="tenant">Tenant</option>
      <option value="user">User</option>
      <option value="model">Model</option>
      <option value="api_key">API Key</option>
    </select>
    
    <select id="rl-target"><!-- populated based on scope --></select>
    
    <input type="number" id="rl-per-second" placeholder="Requests per second" />
    <input type="number" id="rl-per-minute" placeholder="Requests per minute" />
    <input type="number" id="rl-per-hour" placeholder="Requests per hour" />
    <input type="number" id="rl-per-day" placeholder="Requests per day" />
    <input type="number" id="rl-burst" placeholder="Burst size" />
    
    <button onclick="saveRateLimit()">Save</button>
  </div>
</div>
```

## Требования

### Функциональные

1. ✅ Multi-scope rate limiting (API key, user, tenant, model, global)
2. ✅ Sliding window algorithm
3. ✅ Multiple time windows (second, minute, hour, day)
4. ✅ Burst allowance
5. ✅ Rate limit headers в response
6. ✅ Admin UI для configuration
7. ✅ Per-model rate limits
8. ✅ Priority-based checking
9. ✅ Graceful error messages
10. ✅ Configurable via YAML

### Нефункциональные

1. **Performance**
   - Rate limit check < 5ms
   - Efficient sliding window implementation

2. **Accuracy**
   - No race conditions
   - Consistent limits across requests

## Acceptance Criteria

- [ ] Rate limits работают per API key, user, tenant, model
- [ ] Sliding window algorithm корректно подсчитывает requests
- [ ] Multiple time windows (second, minute, hour, day) работают
- [ ] Burst allowance применяется
- [ ] Rate limit headers присутствуют в response
- [ ] 429 status code при rate limit exceeded
- [ ] Retry-After header указывает когда retry
- [ ] Admin UI позволяет настроить rate limits
- [ ] Per-model limits ограничивают specific models
- [ ] Priority checking работает правильно
- [ ] Unit tests для rate limiter
- [ ] Integration tests для rate limiting flow

## Риски и зависимости

### Риски

1. **Race conditions** - concurrent requests
   - Mitigation: Mutex, atomic operations

2. **Memory leaks** - unbounded cache
   - Mitigation: TTL, LRU eviction

### Зависимости

1. Existing rate limiting (AUTH-03)
2. Database storage

## Связанные задачи

- **AUTH-03**: Rate Limiting per Key - базовая версия
- **QUOTA-01**: Usage Quotas - дополнительные limits
- **METRICS-01**: Prometheus - экспорт rate limit metrics

## Примечания

- Sliding window более fair чем fixed window
- Rate limit headers следуют RFC 6585
- Consider Redis для distributed rate limiting (future)

---

**Статус:** 📋 Planned for v1.10.0  
**Последнее обновление:** 2025-10-11


