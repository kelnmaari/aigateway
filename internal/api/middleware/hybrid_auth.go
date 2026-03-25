// Package middleware provides HTTP middleware for authentication
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/jwt"
	"aigateway/internal/config"
	"aigateway/internal/storage"
)

// HybridAuth creates middleware that accepts EITHER JWT tokens OR API Keys
// Priority: API Key (if looks like sk-*) → JWT token → API Key fallback
// Version 1.3.0+: Uses database-backed API keys + bootstrap admin key from config
// Version 4.1.2+: API key checked first to avoid noisy JWT validation logs
func HybridAuth(jwtManager *jwt.Manager, cfg *config.Config, db storage.Database, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if after, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
			tokenString := after

			// Check if token looks like an API key (sk-* prefix) - try API key first
			if strings.HasPrefix(tokenString, "sk-") {
				apiKeyDBAuth := APIKeyDBAuth(cfg, db, logger)
				apiKeyDBAuth(c)
				if !c.IsAborted() {
					c.Next()
					return
				}
				// API key auth failed, don't try JWT for sk-* tokens
				return
			}

			// Token doesn't look like API key - try JWT first
			claims, err := jwtManager.ValidateAccessToken(tokenString)
			if err == nil {
				// JWT is valid - set user context
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("auth_type", "jwt")

				// Don't spam logs for UI endpoints (v3.0.6+)
				if !strings.HasPrefix(c.Request.URL.Path, "/api/ui/") {
					logger.WithFields(logrus.Fields{
						"user_id":  claims.UserID,
						"username": claims.Username,
					}).Debug("JWT authentication successful")
				}

				c.Next()
				return
			}

			// JWT validation failed - token might be a non-standard API key
			logger.WithError(err).Debug("JWT validation failed, trying as API key")

			// Try to authenticate as API key from database
			apiKeyDBAuth := APIKeyDBAuth(cfg, db, logger)
			apiKeyDBAuth(c)
			if !c.IsAborted() {
				// API key authentication succeeded
				c.Next()
				return
			}

			// Both JWT and API key failed
			return
		}

		// Try API Key (x-api-key header)
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			// Authenticate using database-backed API keys
			apiKeyDBAuth := APIKeyDBAuth(cfg, db, logger)
			apiKeyDBAuth(c)
			if !c.IsAborted() {
				c.Next()
				return
			}
			return
		}

		// No valid authentication found
		logger.Warn("Hybrid auth: No valid JWT or API key found")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Authentication required. Provide either Bearer token (JWT) or x-api-key header",
				"type":    "invalid_request_error",
				"code":    "authentication_required",
			},
		})
		c.Abort()
	}
}
