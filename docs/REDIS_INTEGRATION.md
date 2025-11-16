# Redis Integration Guide

> **Version:** v3.0.6+  
> **Status:** Production Ready  
> **Last Updated:** 2025-11-12

---

## 🚀 Overview

AIGateway now includes comprehensive Redis integration for:
- ✅ **Distributed Rate Limiting** (sliding window algorithm)
- ✅ **Persistent Sessions** (user sessions + OIDC state)
- ✅ **JWT Blacklist** (logout/revoke tokens)
- ✅ **Model Metadata Cache** (performance optimization)
- ✅ **Request Deduplication** (idempotency keys)
- ✅ **Multi-Instance Coordination** (Pub/Sub messaging)
- ✅ **Instance Discovery** (health monitoring)

---

## 📦 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Redis Manager                            │
├─────────────────────────────────────────────────────────────────┤
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │  Rate Limiting │  │    Sessions    │  │      JWT       │   │
│  │   (sliding)    │  │   (storage)    │  │  (blacklist)   │   │
│  └────────────────┘  └────────────────┘  └────────────────┘   │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐   │
│  │     Cache      │  │    Pub/Sub     │  │     Stats      │   │
│  │  (metadata)    │  │  (messaging)   │  │  (monitoring)  │   │
│  └────────────────┘  └────────────────┘  └────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                             ↓
                    ┌─────────────────┐
                    │  Redis Server   │
                    │  (Standalone    │
                    │   or Cluster)   │
                    └─────────────────┘
```

---

## 🔧 Configuration

### `configs/dev.yaml`

```yaml
rate_limiting:
  enabled: true
  default_requests_per_minute: 60
  default_requests_per_hour: 1000
  
  redis:
    enabled: true                              # Enable Redis rate limiting
    url: "redis://192.168.1.101:30897"        # Redis URL
    key_prefix: "aigateway_rl:"               # Key prefix

oidc:
  enabled: true
  session_store: "redis"                       # Use Redis for OIDC sessions
  session_ttl: "10m"

# Optional: Redis-specific settings
redis:
  url: "redis://192.168.1.101:30897"
  db: 0                                        # Redis database number
  max_retries: 3
  pool_size: 10
  key_prefix: "aigateway:"                     # Global prefix
```

---

## 💻 Usage Examples

### 1. Initialize Redis Manager

```go
import (
    "aigateway/internal/cache/redis"
    "github.com/sirupsen/logrus"
)

// Create Redis manager
redisManager, err := redis.NewManager(redis.ManagerConfig{
    URL:       cfg.RateLimiting.Redis.URL,
    KeyPrefix: cfg.RateLimiting.Redis.KeyPrefix,
    DB:        0,
    MaxRetries: 3,
    PoolSize:  10,
    Enabled:   cfg.RateLimiting.Redis.Enabled,
}, logger)
if err != nil {
    logger.WithError(err).Fatal("Failed to initialize Redis")
}
defer redisManager.Close()

// Health check
if err := redisManager.Ping(ctx); err != nil {
    logger.WithError(err).Warn("Redis is not available")
}
```

### 2. Distributed Rate Limiting

```go
// Check rate limit (60 requests per minute)
allowed, remaining, resetAt, err := redisManager.RateLimit.CheckLimit(
    ctx,
    apiKey.ID,                    // Key ID
    60,                           // Limit
    1*time.Minute,                // Window
)

if !allowed {
    return fmt.Errorf("rate limit exceeded, reset at: %s", resetAt)
}

logger.Infof("Rate limit: %d/%d, resets at: %s", remaining, 60, resetAt)
```

### 3. Session Management

```go
// Create session
session := &redis.Session{
    SessionID: "sess_123",
    UserID:    "user_456",
    Username:  "john.doe",
    TenantID:  "tenant_789",
    IPAddress: "192.168.1.100",
    UserAgent: "Chrome/120.0",
    Data: map[string]interface{}{
        "role": "admin",
        "permissions": []string{"read", "write"},
    },
}

err := redisManager.Session.CreateSession(ctx, session, 24*time.Hour)

// Get session
session, err := redisManager.Session.GetSession(ctx, "sess_123")

// Refresh session (extend TTL)
err := redisManager.Session.RefreshSession(ctx, "sess_123", 24*time.Hour)

// Delete session (logout)
err := redisManager.Session.DeleteSession(ctx, "sess_123")

// Get all user sessions
sessions, err := redisManager.Session.GetUserSessions(ctx, "user_456")
```

### 4. JWT Blacklist (Logout/Revoke)

```go
// Blacklist a single token (on logout)
err := redisManager.JWT.BlacklistToken(ctx, tokenID, tokenTTL)

// Check if token is blacklisted
isBlacklisted, err := redisManager.JWT.IsTokenBlacklisted(ctx, tokenID)

// Blacklist ALL user tokens (on password change, account compromise)
err := redisManager.JWT.BlacklistUserTokens(ctx, userID, 7*24*time.Hour)

// Check if all user tokens are blacklisted
timestamp, err := redisManager.JWT.AreUserTokensBlacklisted(ctx, userID)
if timestamp > 0 {
    // All tokens issued before this timestamp are invalid
}
```

### 5. Refresh Token Management

```go
// Save refresh token
refreshToken := &redis.RefreshToken{
    TokenID:   "refresh_123",
    UserID:    "user_456",
    TenantID:  "tenant_789",
    IssuedAt:  time.Now(),
    ExpiresAt: time.Now().Add(7*24*time.Hour),
    IPAddress: "192.168.1.100",
    UserAgent: "Mobile App v1.0",
}

err := redisManager.JWT.SaveRefreshToken(ctx, refreshToken, 7*24*time.Hour)

// Get refresh token
token, err := redisManager.JWT.GetRefreshToken(ctx, "refresh_123")

// Revoke specific refresh token
err := redisManager.JWT.RevokeRefreshToken(ctx, "refresh_123")

// Revoke ALL user refresh tokens
err := redisManager.JWT.RevokeUserRefreshTokens(ctx, "user_456")

// List user's active refresh tokens
tokens, err := redisManager.JWT.GetUserRefreshTokens(ctx, "user_456")
```

### 6. Model Metadata Caching

```go
// Cache model metadata
metadata := &redis.ModelMetadata{
    ModelID:         "llama-3.2-3b-q8",
    ModelPath:       "data/models/llama-3.2-3b-q8.gguf",
    Alias:           "llama-3.2-3b-q8",
    Architecture:    "llama",
    Parameters:      3000000000,
    Quantization:    "Q8_0",
    ContextSize:     4096,
    IsVLM:           false,
    IsLoaded:        true,
    LoadedAt:        time.Now(),
    UsageCount:      0,
    AvgTokensPerSec: 0,
}

err := redisManager.Cache.CacheModelMetadata(ctx, metadata)

// Get model metadata
metadata, err := redisManager.Cache.GetModelMetadata(ctx, "llama-3.2-3b-q8")

// Update model stats (after inference)
err := redisManager.Cache.UpdateModelStats(ctx, "llama-3.2-3b-q8", 45.5) // 45.5 tokens/sec

// List all loaded models
models, err := redisManager.Cache.ListLoadedModels(ctx)

// Invalidate cache
err := redisManager.Cache.InvalidateModelCache(ctx, "llama-3.2-3b-q8")
```

### 7. Request Deduplication (Idempotency)

```go
// Save request response for deduplication
record := &redis.IdempotencyRecord{
    Key:        idempotencyKey,
    Response:   responseData,
    StatusCode: 200,
}

err := redisManager.Cache.SaveIdempotencyRecord(ctx, idempotencyKey, record)

// Get cached response (if exists)
record, err := redisManager.Cache.GetIdempotencyRecord(ctx, idempotencyKey)
if err == nil {
    // Return cached response
    return record.Response, record.StatusCode
}
```

### 8. Pub/Sub Messaging (Multi-Instance Coordination)

```go
// Publish model loaded event
err := redisManager.PubSub.PublishModelLoaded(
    ctx,
    "llama-3.2-3b-q8",        // Model ID
    "data/models/...",        // Model path
    "llama-3.2-3b-q8",        // Alias
    "instance-1",             // Instance ID
)

// Subscribe to model events
subscription, err := redisManager.PubSub.Subscribe(ctx, "models", func(msg *redis.Message) error {
    logger.Infof("Received event: %s, payload: %v", msg.Type, msg.Payload)
    
    switch msg.Type {
    case redis.EventModelLoaded:
        // Handle model loaded
    case redis.EventModelUnloaded:
        // Handle model unloaded
    }
    
    return nil
})

// Unsubscribe when done
defer redisManager.PubSub.Unsubscribe(subscription)
```

### 9. Instance Discovery & Health Monitoring

```go
// Register this instance
err := redisManager.RegisterInstance(
    ctx,
    "instance-1",              // Instance ID
    "server-node-01",          // Hostname
    "192.168.1.10",            // IP
    8080,                      // Port
)

// List all registered instances
instances, err := redisManager.PubSub.ListInstances(ctx)
for _, instance := range instances {
    logger.Infof("Instance: %s (%s) - Status: %s, Last seen: %s",
        instance.InstanceID,
        instance.IP,
        instance.Status,
        instance.LastSeen,
    )
}

// Unregister on shutdown
defer redisManager.PubSub.UnregisterInstance(ctx, "instance-1")
```

### 10. Generic Caching

```go
// Cache any data
data := map[string]interface{}{
    "key1": "value1",
    "key2": 12345,
}

err := redisManager.Cache.SetWithTTL(ctx, "my_data", data, 1*time.Hour)

// Get cached data
var cached map[string]interface{}
err := redisManager.Cache.Get(ctx, "my_data", &cached)

// Check if key exists
exists, err := redisManager.Cache.Exists(ctx, "my_data")

// Delete cached data
err := redisManager.Cache.Delete(ctx, "my_data")
```

---

## 📊 Monitoring & Statistics

```go
// Get comprehensive Redis stats
stats, err := redisManager.GetStats(ctx)

/* Returns:
{
    "enabled": true,
    "db_size": 1523,
    "pool": {
        "hits": 12345,
        "misses": 234,
        "timeouts": 0,
        "total_conns": 10,
        "idle_conns": 5,
        "stale_conns": 0
    },
    "cache": {
        "total_keys": 1523,
        "model_metadata": 12,
        "sessions": 45,
        "idempotency_keys": 230,
        "rate_limits": 1200,
        "other": 36
    },
    "instances": 3
}
*/

// Ping Redis
err := redisManager.Ping(ctx)

// Get DB size
dbSize, err := redisManager.Client.DBSize(ctx)

// Get cache stats
cacheStats, err := redisManager.Cache.GetStats(ctx)
```

---

## 🔄 Migration from In-Memory to Redis

### Before (In-Memory)
```go
// In-memory rate limiting
limiter := ratelimit.NewSlidingWindowLimiter(db, logger)
allowed, remaining, resetAt, err := limiter.CheckLimit(ctx, ...)
```

### After (Redis)
```go
// Redis-based distributed rate limiting
allowed, remaining, resetAt, err := redisManager.RateLimit.CheckLimit(ctx, ...)

// Fallback to in-memory if Redis is down
if err != nil && strings.Contains(err.Error(), "redis") {
    // Use in-memory limiter as fallback
    allowed, remaining, resetAt, err = inMemoryLimiter.CheckLimit(ctx, ...)
}
```

---

## ⚡ Performance Considerations

1. **Connection Pooling:**
   - Default pool size: 10 connections
   - Adjust based on concurrent load

2. **TTL Strategy:**
   - Sessions: 24 hours (refreshed on activity)
   - Rate limits: 2x window duration
   - JWT blacklist: Token expiration time
   - Model metadata: 24 hours (long-lived)
   - Idempotency: 1 hour

3. **Pipeline Usage:**
   - Use `client.Pipeline()` for batch operations
   - Reduces round trips by ~10x

4. **Key Patterns:**
   - Use consistent prefixes for key organization
   - Enable key pattern scanning for monitoring
   - Avoid wildcard scans in production (use SCAN instead)

---

## 🔒 Security Best Practices

1. **Redis Authentication:**
   ```bash
   redis://user:password@host:port/db
   ```

2. **TLS/SSL:**
   ```bash
   rediss://host:port/db
   ```

3. **Network Security:**
   - Use Redis in private network
   - Enable Redis AUTH
   - Limit connections to known IPs

4. **Key Expiration:**
   - Always set TTL on keys
   - Prevents memory leaks
   - Automatic cleanup

---

## 🐛 Troubleshooting

### Redis Connection Fails
```go
// Check Redis health
if err := redisManager.Ping(ctx); err != nil {
    logger.WithError(err).Error("Redis is unavailable")
    // Fallback to in-memory
}
```

### High Memory Usage
```bash
# Monitor Redis memory
redis-cli INFO memory

# Check key count by pattern
redis-cli --scan --pattern "aigateway_*" | wc -l

# Flush specific keys
redis-cli DEL aigateway_cache:*
```

### Slow Operations
```bash
# Monitor slow commands
redis-cli SLOWLOG GET 10

# Check latency
redis-cli --latency
```

---

## 📚 References

- [Redis Go Client Documentation](https://redis.uptrace.dev/)
- [Redis Commands Reference](https://redis.io/commands/)
- [Rate Limiting Algorithms](https://engineering.classdojo.com/blog/2015/02/06/rolling-rate-limiter/)

---

**Next:** [Multi-Instance Deployment Guide](MULTI_INSTANCE.md)

