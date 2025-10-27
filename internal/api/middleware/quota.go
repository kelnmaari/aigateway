// Package middleware provides quota middleware for Gin
// Version: 1.11.7+ (Enterprise Suite - Usage Quotas System)
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
	"aigateway/internal/services/quota"
)

// QuotaMiddleware provides quota checking middleware
type QuotaMiddleware struct {
	quotaService *quota.Service
	logger       *logrus.Logger
}

// NewQuotaMiddleware создает новый quota middleware
func NewQuotaMiddleware(quotaService *quota.Service, logger *logrus.Logger) *QuotaMiddleware {
	return &QuotaMiddleware{
		quotaService: quotaService,
		logger:       logger,
	}
}

// CheckQuotas middleware для проверки квот перед обработкой запроса
func (m *QuotaMiddleware) CheckQuotas() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check quotas for chat/completion endpoints
		if !m.isQuotaEnabledEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Extract user and tenant from context (set by auth middleware)
		userIDValue, exists := c.Get("user_id")
		if !exists {
			// No user = no quota check (might be admin or public endpoint)
			c.Next()
			return
		}

		userID, ok := userIDValue.(string)
		if !ok || userID == "" {
			c.Next()
			return
		}

		// Get tenant ID if present
		var tenantID *string
		if tenantIDValue, exists := c.Get("tenant_id"); exists {
			if tid, ok := tenantIDValue.(string); ok && tid != "" {
				tenantID = &tid
			}
		}

		// Estimate tokens needed (can be refined with request body parsing)
		estimatedTokens := m.estimateTokens(c)

		// Get model from request (if available)
		model := m.getModelFromRequest(c)

		// Check quota
		quotaCheck, err := m.quotaService.CheckQuota(
			c.Request.Context(),
			userID,
			tenantID,
			estimatedTokens,
			model,
		)
		if err != nil {
			m.logger.WithError(err).Error("Quota check failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "quota check failed",
			})
			c.Abort()
			return
		}

		// If quota exceeded, return 429
		if !quotaCheck.Allowed {
			m.logger.WithFields(logrus.Fields{
				"user_id":    userID,
				"tenant_id":  tenantID,
				"error_type": quotaCheck.ErrorType,
				"limit":      quotaCheck.Limit,
				"used":       quotaCheck.Used,
			}).Warn("Quota exceeded")

			// Record quota exceeded metric
			quotaType := quotaCheck.ErrorType
			targetID := userID
			if tenantID != nil {
				targetID = *tenantID
			}
			metrics.QuotaExceeded.WithLabelValues(targetID, quotaType).Inc()

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "quota exceeded",
				"type":    quotaCheck.ErrorType,
				"limit":   quotaCheck.Limit,
				"used":    quotaCheck.Used,
				"message": quotaCheck.Message,
			})
			c.Abort()
			return
		}

		// Increment concurrent requests counter
		if err := m.quotaService.IncrementConcurrent(c.Request.Context(), userID, tenantID); err != nil {
			m.logger.WithError(err).Warn("Failed to increment concurrent counter")
		}

		// Decrement on completion (defer)
		defer func() {
			if err := m.quotaService.DecrementConcurrent(c.Request.Context(), userID, tenantID); err != nil {
				m.logger.WithError(err).Warn("Failed to decrement concurrent counter")
			}
		}()

		c.Next()
	}
}

// isQuotaEnabledEndpoint проверяет нужно ли проверять квоты для данного endpoint
func (m *QuotaMiddleware) isQuotaEnabledEndpoint(path string) bool {
	// Check quotas только для chat/completion endpoints
	quotaEnabledPrefixes := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/api/v1/chat",
		"/api/chat",
	}

	for _, prefix := range quotaEnabledPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// estimateTokens оценивает количество токенов для запроса
// TODO: Parse request body to get exact prompt/max_tokens
func (m *QuotaMiddleware) estimateTokens(c *gin.Context) int64 {
	// For now, use conservative estimate
	// In production, parse request body and sum:
	// - prompt tokens (estimate from message content)
	// - max_tokens from request (default 2048)
	return 3000 // Conservative estimate
}

// getModelFromRequest извлекает model name из request
func (m *QuotaMiddleware) getModelFromRequest(c *gin.Context) string {
	// Try to parse request body
	var req struct {
		Model string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err == nil && req.Model != "" {
		return req.Model
	}

	return "unknown"
}


