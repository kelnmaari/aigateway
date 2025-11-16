# JWT Sessions Storage in Redis

> **Version:** v3.0.6+  
> **Retention:** 7 days (configurable)  
> **Status:** Production Ready

---

## 🎯 Overview

JWT сессии теперь хранятся в Redis с автоматической очисткой через 7 дней. Это дает:

- ✅ **Visibility:** Видны все активные сессии пользователя
- ✅ **Security:** Revoke конкретной сессии или всех сразу
- ✅ **Tracking:** IP, Device, User-Agent, Location, Last Activity
- ✅ **Analytics:** Статистика по устройствам, локациям, активным пользователям
- ✅ **Auto-cleanup:** Автоматическое удаление через 7 дней

---

## 📦 Data Structure

```go
type JWTSession struct {
    SessionID    string                 // Unique session ID (from JWT jti claim)
    TokenID      string                 // JWT token ID (jti)
    UserID       string                 
    Username     string                 
    Email        string                 
    TenantID     string                 
    Role         string                 
    Permissions  []string               
    IPAddress    string                 
    UserAgent    string                 
    DeviceType   string                 // web, mobile, desktop
    DeviceName   string                 // Chrome, Safari, Mobile App
    Location     string                 // City, Country
    IssuedAt     time.Time              
    ExpiresAt    time.Time              
    LastActivity time.Time              
    ActivityCount int64                 // Number of requests
    Extra        map[string]interface{} // Custom data
}
```

---

## 💻 Usage Examples

### 1. Save JWT Session (on Login)

```go
import "aigateway/internal/cache/redis"

// After successful JWT generation
session := &redis.JWTSession{
    SessionID:   jti,                    // JWT jti claim
    TokenID:     jti,
    UserID:      user.ID,
    Username:    user.Username,
    Email:       user.Email,
    TenantID:    user.TenantID,
    Role:        user.Role,
    Permissions: user.Permissions,
    IPAddress:   c.ClientIP(),
    UserAgent:   c.GetHeader("User-Agent"),
    DeviceType:  detectDeviceType(userAgent),  // web, mobile, desktop
    DeviceName:  parseDeviceName(userAgent),   // Chrome 120, Safari, etc.
    Location:    getLocationFromIP(ip),        // Optional: GeoIP lookup
}

// Save with default 7 days TTL
err := redisManager.JWT.SaveJWTSession(ctx, session)

// Or with custom TTL
err := redisManager.JWT.SaveJWTSession(ctx, session, 14*24*time.Hour)
```

### 2. Validate JWT Session (on Request)

```go
// In JWT middleware, after token validation
sessionID := claims.JTI  // Get jti from JWT claims

// Check if session is valid
valid, err := redisManager.JWT.IsJWTSessionValid(ctx, sessionID)
if !valid {
    return c.JSON(401, gin.H{"error": "Session expired or revoked"})
}

// Update activity (optional, for tracking)
if err := redisManager.JWT.UpdateJWTSessionActivity(ctx, sessionID); err != nil {
    logger.WithError(err).Warn("Failed to update session activity")
}
```

### 3. Get User's Active Sessions

```go
// Get all active sessions for a user
sessions, err := redisManager.JWT.GetUserJWTSessions(ctx, userID)

// Example response:
/*
[
    {
        "session_id": "sess_abc123",
        "username": "john.doe",
        "ip_address": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "device_type": "web",
        "device_name": "Chrome 120",
        "location": "New York, US",
        "issued_at": "2025-11-12T10:00:00Z",
        "last_activity": "2025-11-12T15:30:00Z",
        "activity_count": 245
    },
    {
        "session_id": "sess_def456",
        "username": "john.doe",
        "ip_address": "10.0.0.5",
        "device_type": "mobile",
        "device_name": "Mobile App v1.0",
        "location": "London, UK",
        "issued_at": "2025-11-11T08:00:00Z",
        "last_activity": "2025-11-12T12:00:00Z",
        "activity_count": 89
    }
]
*/

// Count active sessions
count, err := redisManager.JWT.CountUserJWTSessions(ctx, userID)
```

### 4. Logout (Revoke Session)

```go
// Revoke single session (current device logout)
err := redisManager.JWT.RevokeJWTSession(ctx, sessionID)

// This will:
// 1. Delete the session from Redis
// 2. Add token to blacklist
// 3. Remove from user's sessions set
```

### 5. Logout from All Devices

```go
// Revoke all user sessions (security breach, password change)
err := redisManager.JWT.RevokeUserJWTSessions(ctx, userID)

// This will:
// 1. Delete all sessions
// 2. Blacklist all tokens
// 3. Clean up user's sessions set
```

### 6. Refresh Session (Extend TTL)

```go
// Extend session by another 7 days
err := redisManager.JWT.RefreshJWTSession(ctx, sessionID, 7*24*time.Hour)

// Use case: "Remember me" feature
// On each request, extend session if < 24h left
session, err := redisManager.JWT.GetJWTSession(ctx, sessionID)
if err == nil {
    timeLeft := time.Until(session.ExpiresAt)
    if timeLeft < 24*time.Hour {
        redisManager.JWT.RefreshJWTSession(ctx, sessionID, 7*24*time.Hour)
    }
}
```

### 7. Get Session Statistics

```go
// Get comprehensive stats
stats, err := redisManager.JWT.GetJWTSessionsStats(ctx)

/* Returns:
{
    "total_sessions": 1234,
    "unique_users": 567,
    "by_device": {
        "web": 789,
        "mobile": 345,
        "desktop": 100
    },
    "by_location": {
        "New York, US": 234,
        "London, UK": 123,
        "Tokyo, JP": 89
    }
}
*/
```

---

## 🔧 Integration Examples

### Middleware: Save Session on Login

```go
// internal/auth/middleware/jwt_session.go
func SaveJWTSession(redisManager *redis.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        
        // After successful JWT generation (check context for token)
        jti, exists := c.Get("jwt_id")
        if !exists {
            return
        }
        
        userID, _ := c.Get("user_id")
        username, _ := c.Get("username")
        
        session := &redis.JWTSession{
            SessionID:   jti.(string),
            TokenID:     jti.(string),
            UserID:      userID.(string),
            Username:    username.(string),
            IPAddress:   c.ClientIP(),
            UserAgent:   c.GetHeader("User-Agent"),
            DeviceType:  detectDevice(c.GetHeader("User-Agent")),
            DeviceName:  parseDevice(c.GetHeader("User-Agent")),
        }
        
        if err := redisManager.JWT.SaveJWTSession(c.Request.Context(), session); err != nil {
            logrus.WithError(err).Warn("Failed to save JWT session")
        }
    }
}
```

### Middleware: Validate Session on Request

```go
// internal/auth/middleware/jwt_validator.go
func ValidateJWTSession(redisManager *redis.Manager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get JWT claims (after JWT validation)
        claims, exists := c.Get("jwt_claims")
        if !exists {
            c.Next()
            return
        }
        
        jti := claims.(JWTClaims).JTI
        
        // Check if session is valid
        valid, err := redisManager.JWT.IsJWTSessionValid(c.Request.Context(), jti)
        if err != nil {
            logrus.WithError(err).Warn("Failed to validate JWT session")
            c.Next()
            return
        }
        
        if !valid {
            c.JSON(401, gin.H{
                "error": "Session expired or revoked",
            })
            c.Abort()
            return
        }
        
        // Update activity (optional)
        go func() {
            ctx := context.Background()
            redisManager.JWT.UpdateJWTSessionActivity(ctx, jti)
        }()
        
        c.Next()
    }
}
```

### API Handler: List User Sessions

```go
// internal/api/handlers/user_sessions.go
func (h *Handler) GetUserSessions(c *gin.Context) {
    userID := c.GetString("user_id")
    
    sessions, err := h.redisManager.JWT.GetUserJWTSessions(c.Request.Context(), userID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to get sessions"})
        return
    }
    
    // Format response
    type SessionResponse struct {
        SessionID    string    `json:"session_id"`
        DeviceName   string    `json:"device_name"`
        DeviceType   string    `json:"device_type"`
        IPAddress    string    `json:"ip_address"`
        Location     string    `json:"location"`
        LastActivity time.Time `json:"last_activity"`
        IsCurrent    bool      `json:"is_current"`
    }
    
    currentSessionID := c.GetString("jwt_id")
    
    response := make([]SessionResponse, len(sessions))
    for i, session := range sessions {
        response[i] = SessionResponse{
            SessionID:    session.SessionID,
            DeviceName:   session.DeviceName,
            DeviceType:   session.DeviceType,
            IPAddress:    session.IPAddress,
            Location:     session.Location,
            LastActivity: session.LastActivity,
            IsCurrent:    session.SessionID == currentSessionID,
        }
    }
    
    c.JSON(200, gin.H{
        "sessions": response,
        "total":    len(response),
    })
}
```

### API Handler: Revoke Session

```go
func (h *Handler) RevokeSession(c *gin.Context) {
    sessionID := c.Param("session_id")
    userID := c.GetString("user_id")
    
    // Verify session belongs to user
    session, err := h.redisManager.JWT.GetJWTSession(c.Request.Context(), sessionID)
    if err != nil {
        c.JSON(404, gin.H{"error": "Session not found"})
        return
    }
    
    if session.UserID != userID {
        c.JSON(403, gin.H{"error": "Forbidden"})
        return
    }
    
    // Revoke session
    if err := h.redisManager.JWT.RevokeJWTSession(c.Request.Context(), sessionID); err != nil {
        c.JSON(500, gin.H{"error": "Failed to revoke session"})
        return
    }
    
    c.JSON(200, gin.H{"message": "Session revoked successfully"})
}
```

---

## 🔒 Security Benefits

1. **Session Tracking:**
   - See all devices user is logged in from
   - Detect suspicious activity (unusual locations/devices)

2. **Instant Revocation:**
   - Logout from all devices instantly
   - Revoke compromised sessions

3. **Account Protection:**
   - Automatic logout after 7 days inactivity
   - Force logout on password change

4. **Audit Trail:**
   - Track last activity per session
   - Monitor device/location patterns

---

## 📊 Redis Keys Structure

```
# Active JWT session
jwt_session:{session_id}                → Session data (TTL: 7 days)

# User's sessions set
user_jwt_sessions:{user_id}             → Set of session IDs (TTL: 7 days)

# Token blacklist
jwt_blacklist:{token_id}                → "revoked" (TTL: token expiry)

# User blacklist (all tokens)
jwt_user_blacklist:{user_id}            → timestamp (TTL: 7 days)
```

---

## ⚡ Performance Considerations

1. **Activity Tracking:**
   - Call `UpdateJWTSessionActivity()` asynchronously (goroutine)
   - Or batch updates every N requests

2. **Session Validation:**
   - Cache validation result in request context
   - Avoid multiple Redis calls per request

3. **Cleanup:**
   - Redis TTL handles automatic cleanup
   - Manual cleanup rarely needed

---

## 🎨 UI Examples

### User Profile → Active Sessions

```
┌──────────────────────────────────────────────────────────────┐
│ 🔐 Active Sessions (3)                                       │
├──────────────────────────────────────────────────────────────┤
│ 💻 Chrome on Windows                       [Current Device] │
│    IP: 192.168.1.100                                         │
│    Location: New York, US                                    │
│    Last active: 2 minutes ago                                │
│    Activity: 245 requests                                    │
│                                          [View Details]       │
├──────────────────────────────────────────────────────────────┤
│ 📱 Mobile App v1.0                         [Revoke]          │
│    IP: 10.0.0.5                                              │
│    Location: London, UK                                      │
│    Last active: 3 hours ago                                  │
│    Activity: 89 requests                                     │
│                                          [View Details]       │
├──────────────────────────────────────────────────────────────┤
│ 🖥️ Safari on macOS                         [Revoke]          │
│    IP: 172.16.0.10                                           │
│    Location: Tokyo, JP                                       │
│    Last active: 1 day ago                                    │
│    Activity: 12 requests                                     │
│                                          [View Details]       │
└──────────────────────────────────────────────────────────────┘
                    [Logout from All Devices]
```

---

## 📝 TODO List

- [ ] Integrate in JWT middleware
- [ ] Add API endpoints (list, revoke)
- [ ] Create admin UI for session monitoring
- [ ] Add GeoIP lookup for location
- [ ] Device fingerprinting
- [ ] Session anomaly detection

---

**Related:** [Redis Integration Guide](REDIS_INTEGRATION.md) | [JWT Authentication](../internal/auth/README.md)

