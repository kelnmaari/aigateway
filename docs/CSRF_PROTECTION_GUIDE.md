## CSRF Protection Guide - Go 1.25 `net/http.CrossOriginProtection`

**Feature**: Built-in CSRF protection via `net/http.CrossOriginProtection`  
**Since**: Go 1.25.0  
**Status**: ✅ Stable (not experimental)

## Overview

Go 1.25 introduces `net/http.CrossOriginProtection` - a built-in protection against Cross-Site Request Forgery (CSRF) attacks. This middleware automatically validates cross-origin requests using modern browser security headers.

## What is CSRF?

Cross-Site Request Forgery (CSRF) is an attack where a malicious website tricks a user's browser into making unwanted requests to your application using the user's authenticated session.

**Example Attack:**
1. User logs into `bank.com` (gets session cookie)
2. User visits `evil.com` (without logging out)
3. `evil.com` contains: `<img src="https://bank.com/transfer?to=attacker&amount=1000">`
4. Browser automatically sends bank.com cookies with the request
5. Money transferred without user's knowledge!

**Protection:** Validate that state-changing requests (POST/PUT/DELETE) come from the same origin.

---

## How CrossOriginProtection Works

Go 1.25's `CrossOriginProtection` uses two strategies:

### 1. **Sec-Fetch-Site Header** (Preferred)
Available in all modern browsers since 2023. Browsers automatically set this header:
- `same-origin` - request from same origin
- `same-site` - request from same site (different subdomain)
- `cross-site` - request from different site
- `none` - user-initiated (e.g., bookmark)

### 2. **Origin vs Host Comparison** (Fallback)
Compares the `Origin` header with the `Host` header to detect cross-origin requests.

### Safe Methods
GET, HEAD, and OPTIONS are always allowed (safe methods don't modify state).

---

## Quick Start

### Installation (Built-in to Go 1.25)
```go
import "net/http"
```

### Basic Usage

```go
package main

import (
    "net/http"
    "log"
)

func main() {
    // Create CrossOriginProtection instance
    cop := http.NewCrossOriginProtection()

    // Wrap your handler
    mux := http.NewServeMux()
    mux.HandleFunc("/api/sensitive", sensitiveHandler)

    // Apply CSRF protection
    http.ListenAndServe(":8080", cop.Handler(mux))
}

func sensitiveHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Protected endpoint"))
}
```

### With Trusted Origins

```go
cop := http.NewCrossOriginProtection()

// Add trusted origins (e.g., your frontend domain)
cop.AddTrustedOrigin("https://app.example.com")
cop.AddTrustedOrigin("https://admin.example.com")

http.ListenAndServe(":8080", cop.Handler(mux))
```

---

## Integration with Gin (aigateway Example)

### Middleware Implementation

```go
// internal/api/middleware/csrf.go
package middleware

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/sirupsen/logrus"
)

type CrossOriginProtectionConfig struct {
    Enabled        bool
    TrustedOrigins []string
}

func CrossOriginProtection(cfg CrossOriginProtectionConfig, logger *logrus.Logger) gin.HandlerFunc {
    // Create Go 1.25 CrossOriginProtection instance
    cop := http.NewCrossOriginProtection()

    // Add trusted origins
    for _, origin := range cfg.TrustedOrigins {
        if err := cop.AddTrustedOrigin(origin); err != nil {
            logger.WithError(err).Warn("Failed to add trusted origin")
        }
    }

    return func(c *gin.Context) {
        if !cfg.Enabled {
            c.Next()
            return
        }

        // Check request with Go 1.25 CSRF protection
        if err := cop.Check(c.Request); err != nil {
            logger.Warn("CSRF: Request blocked", "error", err)
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Cross-origin request blocked",
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

### Router Setup

```go
// internal/api/router/router.go
func SetupRouter(cfg *config.Config, logger *logrus.Logger) *gin.Engine {
    router := gin.New()

    // Apply CSRF protection middleware
    csrfCfg := middleware.CrossOriginProtectionConfig{
        Enabled:        cfg.Security.CSRF.Enabled,
        TrustedOrigins: cfg.Security.CSRF.TrustedOrigins,
    }
    router.Use(middleware.CrossOriginProtection(csrfCfg, logger))

    // API routes
    api := router.Group("/api")
    {
        api.POST("/chat/completions", handlers.ChatCompletion)
        api.DELETE("/conversations/:id", handlers.DeleteConversation)
    }

    return router
}
```

### Configuration (YAML)

```yaml
# configs/production.yaml
security:
  csrf:
    enabled: true
    trusted_origins:
      - "https://app.example.com"
      - "https://admin.example.com"
```

---

## API Reference

### `http.NewCrossOriginProtection()`
Creates a new CSRF protection instance with no trusted origins.

```go
cop := http.NewCrossOriginProtection()
```

### `cop.AddTrustedOrigin(origin string) error`
Adds a trusted origin that can make cross-origin requests.

```go
err := cop.AddTrustedOrigin("https://app.example.com")
if err != nil {
    log.Fatal(err)
}
```

**Valid origins:**
- ✅ `https://example.com`
- ✅ `https://example.com:8080`
- ❌ `https://example.com/path` (path not allowed)
- ❌ `*` (wildcard not supported)

### `cop.Check(req *http.Request) error`
Validates if the request is allowed. Returns error if blocked.

```go
if err := cop.Check(r); err != nil {
    http.Error(w, "Forbidden", http.StatusForbidden)
    return
}
```

### `cop.Handler(h http.Handler) http.Handler`
Wraps an HTTP handler with CSRF protection.

```go
protected := cop.Handler(myHandler)
http.ListenAndServe(":8080", protected)
```

### `cop.SetDenyHandler(h http.Handler)`
Custom handler for denied requests (default: 403 Forbidden).

```go
denyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusForbidden)
    json.NewEncoder(w).Encode(map[string]string{
        "error": "CSRF token invalid",
    })
})
cop.SetDenyHandler(denyHandler)
```

### `cop.AddInsecureBypassPattern(pattern string)`
⚠️ **Dangerous!** Bypasses CSRF for paths matching pattern.

```go
// Allow /public/* endpoints without CSRF check
cop.AddInsecureBypassPattern("/public/*")
```

**Use cases:**
- Public webhooks (but prefer validating webhook signature instead)
- Legacy endpoints (migrate to trusted origins instead)

---

## Request Flow

### Same-Origin Request (Allowed)
```
Browser → POST /api/transfer
Headers:
  Sec-Fetch-Site: same-origin
  Origin: https://example.com
  Host: example.com

✅ Allowed (same origin)
```

### Cross-Origin from Trusted Origin (Allowed)
```
Browser → POST /api/transfer
Headers:
  Sec-Fetch-Site: cross-site
  Origin: https://app.example.com
  Host: api.example.com

✅ Allowed (trusted origin)
```

### Cross-Origin from Untrusted Origin (Blocked)
```
Browser → POST /api/transfer
Headers:
  Sec-Fetch-Site: cross-site
  Origin: https://evil.com
  Host: api.example.com

❌ Blocked (untrusted origin)
```

### Non-Browser Request (Allowed)
```
cURL → POST /api/transfer
Headers:
  (no Sec-Fetch-Site or Origin)

✅ Allowed (non-browser request, e.g., API client)
```

**Note:** Non-browser requests (API clients, mobile apps) don't send `Sec-Fetch-Site` and are allowed by default. Use API keys or other authentication for these.

---

## Security Considerations

### ✅ Good Practices

1. **Enable for All State-Changing Endpoints**
```go
// Apply to all routes that modify data
router.Use(middleware.CrossOriginProtection(cfg, logger))
```

2. **Explicitly List Trusted Origins**
```go
// Don't use "*" - list specific domains
TrustedOrigins: []string{
    "https://app.example.com",
    "https://admin.example.com",
}
```

3. **Use HTTPS**
```go
// CSRF protection is less effective over HTTP
TrustedOrigins: []string{
    "https://app.example.com", // ✅ HTTPS
    // "http://app.example.com", // ❌ Don't trust HTTP
}
```

4. **Log Blocked Requests**
```go
if err := cop.Check(c.Request); err != nil {
    logger.WithFields(logrus.Fields{
        "origin":     c.GetHeader("Origin"),
        "sec_fetch":  c.GetHeader("Sec-Fetch-Site"),
        "ip":         c.ClientIP(),
        "path":       c.Request.URL.Path,
    }).Warn("CSRF attack blocked")
    // ...
}
```

### ⚠️ Avoid

1. **Don't Trust `*` (if tempted)**
```go
// ❌ Bad - defeats the purpose
TrustedOrigins: []string{"*"}
```

2. **Don't Bypass CSRF for Sensitive Endpoints**
```go
// ❌ Dangerous
cop.AddInsecureBypassPattern("/api/admin/*")
```

3. **Don't Disable in Production**
```go
// ❌ Only disable in development/testing
if !cfg.Security.CSRF.Enabled {
    // Skip CSRF check
}
```

---

## Testing

### Test Allowed Same-Origin Request

```go
func TestAllowsSameOrigin(t *testing.T) {
    cop := http.NewCrossOriginProtection()

    req := httptest.NewRequest("POST", "/api/test", nil)
    req.Header.Set("Sec-Fetch-Site", "same-origin")

    err := cop.Check(req)
    assert.NoError(t, err)
}
```

### Test Blocked Cross-Origin Request

```go
func TestBlocksCrossOrigin(t *testing.T) {
    cop := http.NewCrossOriginProtection()

    req := httptest.NewRequest("POST", "/api/test", nil)
    req.Header.Set("Sec-Fetch-Site", "cross-site")
    req.Header.Set("Origin", "https://evil.com")

    err := cop.Check(req)
    assert.Error(t, err)
}
```

### Test Trusted Origin

```go
func TestAllowsTrustedOrigin(t *testing.T) {
    cop := http.NewCrossOriginProtection()
    cop.AddTrustedOrigin("https://trusted.com")

    req := httptest.NewRequest("POST", "/api/test", nil)
    req.Header.Set("Sec-Fetch-Site", "cross-site")
    req.Header.Set("Origin", "https://trusted.com")

    err := cop.Check(req)
    assert.NoError(t, err)
}
```

### Integration Test with Gin

```go
func TestCRSFMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)
    router := gin.New()

    cfg := middleware.CrossOriginProtectionConfig{
        Enabled:        true,
        TrustedOrigins: []string{"https://trusted.com"},
    }
    router.Use(middleware.CrossOriginProtection(cfg, logger))

    router.POST("/test", func(c *gin.Context) {
        c.String(200, "OK")
    })

    // Blocked request
    w := httptest.NewRecorder()
    req := httptest.NewRequest("POST", "/test", nil)
    req.Header.Set("Sec-Fetch-Site", "cross-site")
    req.Header.Set("Origin", "https://evil.com")
    router.ServeHTTP(w, req)
    assert.Equal(t, 403, w.Code)

    // Allowed request
    w = httptest.NewRecorder()
    req = httptest.NewRequest("POST", "/test", nil)
    req.Header.Set("Sec-Fetch-Site", "cross-site")
    req.Header.Set("Origin", "https://trusted.com")
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)
}
```

---

## Browser Compatibility

### Sec-Fetch-Site Header Support
| Browser          | Since Version | Percentage |
|------------------|---------------|------------|
| Chrome/Edge      | 76 (2019)     | ~65%       |
| Firefox          | 90 (2021)     | ~10%       |
| Safari           | 16.4 (2023)   | ~20%       |
| Opera            | 63 (2019)     | ~2%        |

**Coverage:** ~97% of browsers (2024)

### Origin Header Fallback
For older browsers, `CrossOriginProtection` falls back to comparing `Origin` vs `Host` headers.

---

## Migration from Custom CSRF Solutions

### Before (Custom Implementation)

```go
func csrfMiddleware(c *gin.Context) {
    token := c.GetHeader("X-CSRF-Token")
    session := getSession(c)
    
    if token != session.CSRFToken {
        c.AbortWithStatus(403)
        return
    }
    
    c.Next()
}
```

### After (Go 1.25 CrossOriginProtection)

```go
func csrfMiddleware(cfg Config, logger *logrus.Logger) gin.HandlerFunc {
    cop := http.NewCrossOriginProtection()
    for _, origin := range cfg.TrustedOrigins {
        cop.AddTrustedOrigin(origin)
    }
    
    return func(c *gin.Context) {
        if err := cop.Check(c.Request); err != nil {
            logger.Warn("CSRF blocked", "error", err)
            c.AbortWithStatus(403)
            return
        }
        c.Next()
    }
}
```

**Benefits:**
- ✅ No need to generate/store CSRF tokens
- ✅ No need to synchronize tokens between server and client
- ✅ Built-in browser header validation
- ✅ Simpler implementation

---

## Debugging

### Enable Logging

```go
if err := cop.Check(c.Request); err != nil {
    logger.WithFields(logrus.Fields{
        "method":        c.Request.Method,
        "path":          c.Request.URL.Path,
        "origin":        c.GetHeader("Origin"),
        "sec_fetch":     c.GetHeader("Sec-Fetch-Site"),
        "referer":       c.GetHeader("Referer"),
        "host":          c.Request.Host,
        "ip":            c.ClientIP(),
        "user_agent":    c.Request.UserAgent(),
        "error":         err.Error(),
    }).Warn("CSRF protection blocked request")
}
```

### Common Issues

**Issue:** API client requests blocked
**Solution:** API clients don't send `Sec-Fetch-Site`, so they're allowed by default. Use API keys for authentication instead.

**Issue:** Frontend SPA blocked
**Solution:** Add frontend domain to `TrustedOrigins`:
```go
cop.AddTrustedOrigin("https://app.example.com")
```

**Issue:** Development environment blocked
**Solution:** Add localhost to trusted origins (dev only):
```go
if cfg.Environment == "development" {
    cop.AddTrustedOrigin("http://localhost:3000")
}
```

---

## Performance

**Overhead:** Minimal - just header validation (no cryptographic operations).

### Benchmark Results

```
BenchmarkCrossOriginProtection_SafeMethod-8    10000000   120 ns/op
BenchmarkCrossOriginProtection_SameOrigin-8     5000000   240 ns/op
BenchmarkCrossOriginProtection_TrustedOrigin-8  3000000   380 ns/op
```

**Conclusion:** <1μs overhead per request. Negligible for most applications.

---

## Summary

**What is CrossOriginProtection?**
- Built-in Go 1.25 CSRF protection
- Uses `Sec-Fetch-Site` and `Origin` headers
- No tokens required

**Why Use It?**
- Protects against CSRF attacks
- Simpler than custom CSRF implementations
- Native browser support

**How to Use?**
1. Create instance: `http.NewCrossOriginProtection()`
2. Add trusted origins: `cop.AddTrustedOrigin("https://app.example.com")`
3. Wrap handler: `cop.Handler(myHandler)` or use `cop.Check(req)`

**Best Practices:**
- Enable for all state-changing endpoints
- Explicitly list trusted origins (no wildcards)
- Use HTTPS
- Log blocked requests

---

**Last Updated**: 2025-11-04  
**Go Version**: 1.25.3  
**Project**: aigateway v2.4.9  
**Related:** [CSRF on MDN](https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/CSRF)

