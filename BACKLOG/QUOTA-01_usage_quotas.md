# QUOTA-01: Usage Quotas System

**Версия:** 1.10.0  
**Приоритет:** High  
**Сложность:** Medium  
**Оценка:** 10-14 часов

## Описание

Система квот для ограничения использования ресурсов на уровне пользователей и tenants. Админы смогут устанавливать лимиты на tokens, requests, storage для контроля расходов и fair usage.

## Проблема

В текущей версии:
- Нет ограничений на использование
- Пользователи могут делать unlimited requests
- Нет контроля расходов
- Риск ресурсного истощения (resource exhaustion)
- Нет fair usage policy

## Решение

Flexible quota system с per-user и per-tenant limits.

### Архитектура

```
Request → Check Quota → [Allow/Deny] → Process → Update Usage
                                                         ↓
                                                  Quota Tracking
```

## Технические детали

### Quota Types

**1. Token Quotas**
- Total tokens per day/month
- Prompt tokens limit
- Completion tokens limit

**2. Request Quotas**
- Max requests per day/month
- Max concurrent requests

**3. Storage Quotas**
- Max file upload size
- Max total storage per user/tenant
- Max conversations count

**4. Model-specific Quotas**
- Per-model token limits
- Restricted models for certain users/tenants

### Data Model

```go
// internal/models/quota.go

type Quota struct {
    ID       string  `json:"id" db:"id"`
    Name     string  `json:"name" db:"name"`
    Scope    string  `json:"scope" db:"scope"` // "user", "tenant"
    TargetID string  `json:"target_id" db:"target_id"`
    
    // Token limits
    TokensPerDay   *int64 `json:"tokens_per_day,omitempty" db:"tokens_per_day"`
    TokensPerMonth *int64 `json:"tokens_per_month,omitempty" db:"tokens_per_month"`
    
    // Request limits
    RequestsPerDay   *int64 `json:"requests_per_day,omitempty" db:"requests_per_day"`
    RequestsPerMonth *int64 `json:"requests_per_month,omitempty" db:"requests_per_month"`
    MaxConcurrent    *int   `json:"max_concurrent,omitempty" db:"max_concurrent"`
    
    // Storage limits
    MaxStorageBytes *int64 `json:"max_storage_bytes,omitempty" db:"max_storage_bytes"`
    MaxConversations *int  `json:"max_conversations,omitempty" db:"max_conversations"`
    MaxFileSize     *int64 `json:"max_file_size,omitempty" db:"max_file_size"`
    
    // Model restrictions
    AllowedModels []string `json:"allowed_models,omitempty" db:"allowed_models"` // JSON
    
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type QuotaUsage struct {
    ID       string `json:"id" db:"id"`
    QuotaID  string `json:"quota_id" db:"quota_id"`
    TargetID string `json:"target_id" db:"target_id"`
    
    // Current usage
    TokensUsedToday   int64     `json:"tokens_used_today" db:"tokens_used_today"`
    TokensUsedMonth   int64     `json:"tokens_used_month" db:"tokens_used_month"`
    RequestsToday     int64     `json:"requests_today" db:"requests_today"`
    RequestsMonth     int64     `json:"requests_month" db:"requests_month"`
    CurrentConcurrent int       `json:"current_concurrent" db:"current_concurrent"`
    StorageUsedBytes  int64     `json:"storage_used_bytes" db:"storage_used_bytes"`
    ConversationsCount int      `json:"conversations_count" db:"conversations_count"`
    
    // Reset timestamps
    LastDailyReset   time.Time `json:"last_daily_reset" db:"last_daily_reset"`
    LastMonthlyReset time.Time `json:"last_monthly_reset" db:"last_monthly_reset"`
    
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
```

### Database Schema

```sql
CREATE TABLE quotas (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    scope TEXT NOT NULL, -- 'user', 'tenant'
    target_id TEXT NOT NULL, -- user_id or tenant_id
    
    tokens_per_day INTEGER,
    tokens_per_month INTEGER,
    requests_per_day INTEGER,
    requests_per_month INTEGER,
    max_concurrent INTEGER,
    max_storage_bytes INTEGER,
    max_conversations INTEGER,
    max_file_size INTEGER,
    allowed_models TEXT, -- JSON array
    
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(scope, target_id)
);

CREATE TABLE quota_usage (
    id TEXT PRIMARY KEY,
    quota_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    
    tokens_used_today INTEGER DEFAULT 0,
    tokens_used_month INTEGER DEFAULT 0,
    requests_today INTEGER DEFAULT 0,
    requests_month INTEGER DEFAULT 0,
    current_concurrent INTEGER DEFAULT 0,
    storage_used_bytes INTEGER DEFAULT 0,
    conversations_count INTEGER DEFAULT 0,
    
    last_daily_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_monthly_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (quota_id) REFERENCES quotas(id) ON DELETE CASCADE,
    UNIQUE(quota_id, target_id)
);

CREATE INDEX idx_quotas_target ON quotas(scope, target_id);
CREATE INDEX idx_quota_usage_target ON quota_usage(target_id);
```

### Quota Service

```go
// internal/services/quota/service.go

type QuotaService struct {
    db     storage.Database
    logger *logrus.Logger
    mu     sync.RWMutex
}

func (s *QuotaService) CheckQuota(
    ctx context.Context,
    userID string,
    tenantID *string,
    tokensNeeded int64,
) error {
    // Get quota (user or tenant)
    quota, usage, err := s.getQuotaAndUsage(ctx, userID, tenantID)
    if err != nil {
        // No quota = unlimited
        return nil
    }
    
    // Auto-reset if needed
    s.autoReset(ctx, usage)
    
    // Check daily token limit
    if quota.TokensPerDay != nil {
        if usage.TokensUsedToday+tokensNeeded > *quota.TokensPerDay {
            return ErrQuotaExceeded{Type: "daily_tokens", Limit: *quota.TokensPerDay, Used: usage.TokensUsedToday}
        }
    }
    
    // Check monthly token limit
    if quota.TokensPerMonth != nil {
        if usage.TokensUsedMonth+tokensNeeded > *quota.TokensPerMonth {
            return ErrQuotaExceeded{Type: "monthly_tokens", Limit: *quota.TokensPerMonth, Used: usage.TokensUsedMonth}
        }
    }
    
    // Check daily request limit
    if quota.RequestsPerDay != nil {
        if usage.RequestsToday+1 > *quota.RequestsPerDay {
            return ErrQuotaExceeded{Type: "daily_requests", Limit: *quota.RequestsPerDay, Used: usage.RequestsToday}
        }
    }
    
    // Check monthly request limit
    if quota.RequestsPerMonth != nil {
        if usage.RequestsMonth+1 > *quota.RequestsPerMonth {
            return ErrQuotaExceeded{Type: "monthly_requests", Limit: *quota.RequestsPerMonth, Used: usage.RequestsMonth}
        }
    }
    
    // Check concurrent requests
    if quota.MaxConcurrent != nil {
        if usage.CurrentConcurrent >= *quota.MaxConcurrent {
            return ErrQuotaExceeded{Type: "concurrent", Limit: int64(*quota.MaxConcurrent), Used: int64(usage.CurrentConcurrent)}
        }
    }
    
    return nil
}

func (s *QuotaService) RecordUsage(
    ctx context.Context,
    userID string,
    tenantID *string,
    tokens int64,
) error {
    quota, usage, err := s.getQuotaAndUsage(ctx, userID, tenantID)
    if err != nil {
        return nil // No quota tracking
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    usage.TokensUsedToday += tokens
    usage.TokensUsedMonth += tokens
    usage.RequestsToday += 1
    usage.RequestsMonth += 1
    usage.UpdatedAt = time.Now()
    
    return s.db.UpdateQuotaUsage(ctx, usage)
}

func (s *QuotaService) IncrementConcurrent(ctx context.Context, userID string, tenantID *string) error {
    _, usage, err := s.getQuotaAndUsage(ctx, userID, tenantID)
    if err != nil {
        return nil
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    usage.CurrentConcurrent += 1
    return s.db.UpdateQuotaUsage(ctx, usage)
}

func (s *QuotaService) DecrementConcurrent(ctx context.Context, userID string, tenantID *string) error {
    _, usage, err := s.getQuotaAndUsage(ctx, userID, tenantID)
    if err != nil {
        return nil
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if usage.CurrentConcurrent > 0 {
        usage.CurrentConcurrent -= 1
    }
    return s.db.UpdateQuotaUsage(ctx, usage)
}

func (s *QuotaService) autoReset(ctx context.Context, usage *models.QuotaUsage) {
    now := time.Now()
    
    // Daily reset (if last reset was yesterday or earlier)
    if now.Sub(usage.LastDailyReset) >= 24*time.Hour {
        usage.TokensUsedToday = 0
        usage.RequestsToday = 0
        usage.LastDailyReset = now
        s.db.UpdateQuotaUsage(ctx, usage)
    }
    
    // Monthly reset (if different month)
    if now.Month() != usage.LastMonthlyReset.Month() || now.Year() != usage.LastMonthlyReset.Year() {
        usage.TokensUsedMonth = 0
        usage.RequestsMonth = 0
        usage.LastMonthlyReset = now
        s.db.UpdateQuotaUsage(ctx, usage)
    }
}

type ErrQuotaExceeded struct {
    Type  string
    Limit int64
    Used  int64
}

func (e ErrQuotaExceeded) Error() string {
    return fmt.Sprintf("quota exceeded: %s (limit: %d, used: %d)", e.Type, e.Limit, e.Used)
}
```

### Middleware Integration

```go
// internal/api/middleware/quota.go

func QuotaMiddleware(quotaService *quota.QuotaService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Only check quotas for chat endpoints
        if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/chat") {
            c.Next()
            return
        }
        
        userID, _ := c.Get("user_id")
        tenantID, _ := c.Get("tenant_id")
        
        // Estimate tokens (or use actual from request body)
        estimatedTokens := int64(2000) // Rough estimate
        
        // Check quota
        if err := quotaService.CheckQuota(
            c.Request.Context(),
            userID.(string),
            stringPtrOrNil(tenantID),
            estimatedTokens,
        ); err != nil {
            if quotaErr, ok := err.(quota.ErrQuotaExceeded); ok {
                c.JSON(http.StatusTooManyRequests, gin.H{
                    "error": "quota exceeded",
                    "details": gin.H{
                        "type":  quotaErr.Type,
                        "limit": quotaErr.Limit,
                        "used":  quotaErr.Used,
                    },
                })
                c.Abort()
                return
            }
        }
        
        // Increment concurrent
        quotaService.IncrementConcurrent(c.Request.Context(), userID.(string), stringPtrOrNil(tenantID))
        defer quotaService.DecrementConcurrent(c.Request.Context(), userID.(string), stringPtrOrNil(tenantID))
        
        c.Next()
        
        // Record actual usage after request
        tokens := getActualTokensFromResponse(c)
        quotaService.RecordUsage(c.Request.Context(), userID.(string), stringPtrOrNil(tenantID), tokens)
    }
}
```

### Admin UI

**Admin Panel → Quotas Management:**

```html
<div id="quotas-tab" class="tab-pane">
  <h3>Usage Quotas</h3>
  
  <button onclick="createQuota()">Create Quota</button>
  
  <!-- Quotas List -->
  <table class="quotas-table">
    <thead>
      <tr>
        <th>Target</th>
        <th>Scope</th>
        <th>Daily Tokens</th>
        <th>Monthly Tokens</th>
        <th>Daily Requests</th>
        <th>Usage</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody id="quotas-list">
      <!-- Populated by JS -->
    </tbody>
  </table>
  
  <!-- Create/Edit Quota Modal -->
  <div id="quota-modal" class="modal">
    <h3>Create Quota</h3>
    
    <select id="quota-scope">
      <option value="user">User</option>
      <option value="tenant">Tenant</option>
    </select>
    
    <select id="quota-target"><!-- populated --></select>
    
    <h4>Token Limits</h4>
    <input type="number" id="tokens-per-day" placeholder="Tokens per day" />
    <input type="number" id="tokens-per-month" placeholder="Tokens per month" />
    
    <h4>Request Limits</h4>
    <input type="number" id="requests-per-day" placeholder="Requests per day" />
    <input type="number" id="requests-per-month" placeholder="Requests per month" />
    <input type="number" id="max-concurrent" placeholder="Max concurrent requests" />
    
    <h4>Storage Limits</h4>
    <input type="number" id="max-storage" placeholder="Max storage (bytes)" />
    <input type="number" id="max-conversations" placeholder="Max conversations" />
    
    <button onclick="saveQuota()">Save Quota</button>
  </div>
</div>
```

### User UI (Profile)

**Display Quota Usage:**

```html
<!-- web/profile.html → My Usage & Quotas -->
<div class="user-quota-display">
  <h3>My Usage & Quotas</h3>
  
  <div class="quota-card">
    <h4>Daily Tokens</h4>
    <div class="progress-bar">
      <div class="progress" style="width: 65%"></div>
    </div>
    <p>6,500 / 10,000 tokens used today</p>
  </div>
  
  <div class="quota-card">
    <h4>Monthly Tokens</h4>
    <div class="progress-bar">
      <div class="progress" style="width: 45%"></div>
    </div>
    <p>450,000 / 1,000,000 tokens used this month</p>
  </div>
  
  <div class="quota-card">
    <h4>Daily Requests</h4>
    <p>45 / 100 requests today</p>
  </div>
</div>
```

## Требования

### Функциональные

1. ✅ Quota model (tokens, requests, storage limits)
2. ✅ Per-user и per-tenant quotas
3. ✅ Quota checking перед request
4. ✅ Usage tracking после request
5. ✅ Auto-reset (daily, monthly)
6. ✅ Concurrent request limiting
7. ✅ Admin UI для quota management
8. ✅ User UI для viewing own usage
9. ✅ Quota exceeded error handling
10. ✅ Model-specific restrictions

### Нефункциональные

1. **Performance**
   - Quota check < 5ms
   - Atomic usage updates

2. **Accuracy**
   - Precise token counting
   - No race conditions в concurrent tracking

## Acceptance Criteria

- [ ] Quotas создаются для users/tenants
- [ ] Quota check блокирует requests при превышении
- [ ] Usage tracking обновляется после каждого request
- [ ] Auto-reset сбрасывает daily/monthly counters
- [ ] Concurrent request limiting работает
- [ ] Admin UI позволяет создать/edit quotas
- [ ] User UI отображает current usage
- [ ] 429 status code при quota exceeded
- [ ] Storage quotas работают для file uploads
- [ ] Model restrictions применяются
- [ ] Unit tests для quota service
- [ ] Integration tests для quota flow

## Риски и зависимости

### Риски

1. **Race conditions** - concurrent updates
   - Mitigation: Mutex, atomic operations

2. **Inaccurate token counting** - estimation vs actual
   - Mitigation: Use actual tokens from Ollama response

### Зависимости

1. Existing usage tracking (api_usage table)
2. Token counting from Ollama responses

## Связанные задачи

- **RATE-02**: Advanced Rate Limiting - дополнительное rate limiting
- **MODEL-01**: Dynamic Parameters - ограничение num_ctx via quotas
- **METRICS-01**: Prometheus - экспорт quota metrics

## Примечания

- Quotas не применяются к admin users (optional override)
- Soft limits vs hard limits - в future versions
- Quota alerts/notifications - в future versions

---

**Статус:** 📋 Planned for v1.10.0  
**Последнее обновление:** 2025-10-11


