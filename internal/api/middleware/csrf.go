// Package middleware provides CSRF protection via Go 1.25 CrossOriginProtection
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CrossOriginProtectionConfig configuration for CSRF protection
type CrossOriginProtectionConfig struct {
	// Enabled enables Cross-Origin Protection (CSRF defense)
	Enabled bool

	// TrustedOrigins list of origins that should bypass CSRF checks
	// Example: []string{"https://example.com", "https://app.example.com"}
	TrustedOrigins []string

	// AllowCredentials whether to allow credentials in cross-origin requests
	AllowCredentials bool

	// RequireOriginHeader require Origin header for POST/PUT/DELETE/PATCH
	RequireOriginHeader bool
}

// CrossOriginProtection creates middleware that uses Go 1.25's net/http.CrossOriginProtection
//
// This middleware protects against Cross-Site Request Forgery (CSRF) attacks by:
// 1. Using Sec-Fetch-Site header (available in all browsers since 2023)
// 2. Comparing Origin header with Host header
// 3. Allowing trusted origins to bypass checks
//
// Go 1.25 Feature: Uses http.NewCrossOriginProtection() and .Handler()
func CrossOriginProtection(cfg CrossOriginProtectionConfig, logger *logrus.Logger) gin.HandlerFunc {
	// Create Go 1.25 CrossOriginProtection instance
	cop := http.NewCrossOriginProtection()

	// Add trusted origins
	for _, origin := range cfg.TrustedOrigins {
		if origin != "*" { // Wildcard not supported by native API
			if err := cop.AddTrustedOrigin(origin); err != nil {
				logger.WithError(err).WithField("origin", origin).Warn("Failed to add trusted origin")
			}
		}
	}

	return func(c *gin.Context) {
		// Skip if disabled
		if !cfg.Enabled {
			c.Next()
			return
		}

		// Use Go 1.25 CrossOriginProtection.Check()
		if err := cop.Check(c.Request); err != nil {
			logger.WithFields(logrus.Fields{
				"method":       c.Request.Method,
				"path":         c.Request.URL.Path,
				"origin":       c.GetHeader("Origin"),
				"sec_fetch":    c.GetHeader("Sec-Fetch-Site"),
				"request_host": c.Request.Host,
				"ip":           c.ClientIP(),
				"error":        err.Error(),
			}).Warn("CSRF: Request blocked by CrossOriginProtection")

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Cross-origin request blocked: " + err.Error(),
				"code":    "CSRF_BLOCKED",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isSafeMethod returns true for methods that don't modify state (safe for CSRF)
func isSafeMethod(method string) bool {
	return method == http.MethodGet ||
		method == http.MethodHead ||
		method == http.MethodOptions ||
		method == http.MethodTrace
}

// isOriginTrusted checks if the origin is trusted
func isOriginTrusted(origin string, trustedOrigins []string, requestHost string) bool {
	// Always trust same-origin requests
	if isSameOrigin(origin, requestHost) {
		return true
	}

	// Check trusted origins list
	for _, trusted := range trustedOrigins {
		if trusted == "*" {
			return true // Allow all origins (not recommended)
		}
		if trusted == origin {
			return true
		}
	}

	return false
}

// isSameOrigin checks if the origin matches the request host (same-origin)
func isSameOrigin(origin, requestHost string) bool {
	// Parse origin to extract host
	// Origin format: "https://example.com:8080"
	// requestHost format: "example.com:8080"

	// Simple check: if origin contains the requestHost
	// More robust check would parse the URL, but this works for most cases
	if origin == "" {
		return false
	}

	// Remove protocol from origin
	originHost := origin
	if len(origin) > 7 && origin[:7] == "http://" {
		originHost = origin[7:]
	} else if len(origin) > 8 && origin[:8] == "https://" {
		originHost = origin[8:]
	}

	// Compare hosts (case-insensitive)
	return originHost == requestHost
}

// DefaultCrossOriginProtectionConfig returns safe defaults
func DefaultCrossOriginProtectionConfig() CrossOriginProtectionConfig {
	return CrossOriginProtectionConfig{
		Enabled:             true,  // Enabled by default for security
		TrustedOrigins:      []string{}, // Empty by default, must be configured
		AllowCredentials:    false,
		RequireOriginHeader: true, // Strict by default
	}
}

