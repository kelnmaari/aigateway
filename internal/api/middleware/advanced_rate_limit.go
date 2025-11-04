// Package middleware provides advanced rate limiting middleware
// Version 1.12.2+: Advanced Rate Limiting (RATE-02)
package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/ratelimit"
	"aigateway/internal/utils"
)

// AdvancedRateLimitMiddleware middleware для advanced rate limiting с RFC 6585 headers
func AdvancedRateLimitMiddleware(limiter *ratelimit.AdvancedRateLimiter, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user/tenant/apikey/model from context
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")
		apiKeyID, _ := c.Get("api_key_id")

		// Extract model from request body или context
		modelName := extractModelFromContext(c)

		// Convert to strings with nil-safety
		userIDStr := stringValue(userID)
		tenantIDPtr := utils.PtrOrNil(stringValue(tenantID))
		apiKeyIDPtr := utils.PtrOrNil(stringValue(apiKeyID))

		if userIDStr == "" {
			// No user context, skip rate limiting (legacy API keys?)
			c.Next()
			return
		}

		// Check rate limit
		result, err := limiter.CheckRateLimit(
			c.Request.Context(),
			userIDStr,
			tenantIDPtr,
			apiKeyIDPtr,
			modelName,
		)

		if err != nil {
			logger.WithError(err).Error("Rate limit check failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Rate limit check failed",
					"type":    "internal_error",
					"code":    "rate_limit_check_error",
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers (RFC 6585)
		c.Header("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetAt.Unix(), 10))
		c.Header("X-RateLimit-Window", result.Window)
		if result.Scope != "" && result.Scope != "none" {
			c.Header("X-RateLimit-Scope", result.Scope)
		}

		if !result.Allowed {
			// Calculate retry-after in seconds
			retryAfter := int(time.Until(result.ResetAt).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}

			c.Header("Retry-After", strconv.Itoa(retryAfter))

			logger.WithFields(logrus.Fields{
				"user_id":     userIDStr,
				"tenant_id":   tenantIDPtr,
				"api_key_id":  apiKeyIDPtr,
				"model":       modelName,
				"scope":       result.Scope,
				"limit":       result.Limit,
				"window":      result.Window,
				"retry_after": retryAfter,
			}).Warn("Rate limit exceeded")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"message": formatRateLimitMessage(result),
					"type":    "rate_limit_exceeded",
					"code":    "rate_limit_exceeded",
					"details": gin.H{
						"limit":       result.Limit,
						"window":      result.Window,
						"scope":       result.Scope,
						"retry_after": retryAfter,
						"reset_at":    result.ResetAt.Unix(),
					},
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// extractModelFromContext извлекает model name из gin context или request
func extractModelFromContext(c *gin.Context) string {
	// Try context first (set by previous middleware)
	if model, exists := c.Get("model"); exists {
		if modelStr, ok := model.(string); ok {
			return modelStr
		}
	}

	// Try from request body (для chat/completions endpoints)
	// Note: This doesn't consume the body as it was already parsed
	if c.ContentType() == "application/json" {
		// Не парсим body повторно, используем то что уже в контексте
		if modelValue, exists := c.Get("parsed_model"); exists {
			if modelStr, ok := modelValue.(string); ok {
				return modelStr
			}
		}
	}

	return ""
}

// stringValue безопасное извлечение string из interface{}
func stringValue(val interface{}) string {
	if val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return ""
}

// stringPtr replaced with utils.PtrOrNil[T] (Go 1.25 generics)
// utils.PtrOrNil returns pointer to value, or nil if value is zero-value

// formatRateLimitMessage форматирует сообщение об ошибке rate limit
func formatRateLimitMessage(result *models.RateLimitResult) string {
	if result.Window == "second" {
		return "Rate limit exceeded: too many requests per second"
	}
	if result.Window == "minute" {
		return "Rate limit exceeded: too many requests per minute"
	}
	if result.Window == "hour" {
		return "Rate limit exceeded: too many requests per hour"
	}
	if result.Window == "day" {
		return "Rate limit exceeded: too many requests per day"
	}
	return "Rate limit exceeded"
}


