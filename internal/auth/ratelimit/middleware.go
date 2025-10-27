// Package ratelimit provides rate limiting middleware for API keys
package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/middleware"
	"aigateway/internal/metrics"
	"aigateway/internal/models"
)

// RateLimitMiddleware создает middleware для rate limiting
func (l *Limiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если rate limiting отключен, пропускаем
		if !l.enabled {
			c.Next()
			return
		}

		// Проверяем что пользователь аутентифицирован
		keyInfo, authenticated := middleware.GetAuthenticatedKeyInfo(c)
		if !authenticated {
			// Если не аутентифицирован, применяем global rate limiting (опционально)
			c.Next()
			return
		}

		keyID, _ := middleware.GetAuthenticatedKeyID(c)

		// 🔓 КРИТИЧНО: Пропускаем админские ключи (permissions содержит "*")
		if keyInfo.HasPermission("*") {
			l.logger.WithFields(logrus.Fields{
				"key_id":   keyID,
				"key_name": keyInfo.Name,
			}).Debug("⚡ Skipping rate limit for admin key")
			c.Next()
			return
		}

		// Оцениваем количество токенов для запроса
		estimatedTokens := l.estimateTokensFromRequest(c)

		// Проверяем rate limits
		result := l.CheckRateLimit(c.Request.Context(), keyID, keyInfo, estimatedTokens)

		if !result.Allowed {
			l.handleRateLimitExceeded(c, result, keyInfo)
			return
		}

		// Устанавливаем rate limit заголовки
		l.setRateLimitHeaders(c, keyInfo, result)

		c.Next()
	}
}

// estimateTokensFromRequest оценивает количество токенов в запросе
func (l *Limiter) estimateTokensFromRequest(c *gin.Context) int64 {
	// Для chat/completions оцениваем на основе размера body
	if c.Request.URL.Path == "/v1/chat/completions" {
		contentLength := c.Request.ContentLength
		if contentLength > 0 {
			// Примерно 4 символа = 1 токен
			return contentLength / 4
		}
		// Default оценка для chat
		return 50
	}

	// Для других эндпоинтов - minimal токены
	return 1
}

// handleRateLimitExceeded обрабатывает превышение rate limit
func (l *Limiter) handleRateLimitExceeded(c *gin.Context, result *RateLimitResult, keyInfo *models.APIKeyPublic) {
	// Record rate limit exceeded metric
	if metrics.DefaultMetrics != nil {
		metrics.DefaultMetrics.RecordAPIKeyRateLimitExceeded(keyInfo.ID)
	}

	l.logger.WithFields(logrus.Fields{
		"key_id":      keyInfo.ID,
		"key_name":    keyInfo.Name,
		"reason":      result.Reason,
		"retry_after": result.RetryAfter,
		"endpoint":    c.Request.URL.Path,
	}).Warn("Rate limit exceeded")

	// Устанавливаем заголовки для rate limiting
	if result.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int(result.RetryAfter.Seconds())))
	}

	c.Header("X-RateLimit-Limit", strconv.FormatInt(int64(keyInfo.RateLimits.RequestsPerMinute), 10))
	c.Header("X-RateLimit-Remaining", "0")

	if !result.ResetTime.IsZero() {
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"message": result.Reason,
			"type":    "rate_limit_error",
			"code":    "rate_limit_exceeded",
		},
	})
	c.Abort()
}

// setRateLimitHeaders устанавливает rate limit заголовки для успешных запросов
func (l *Limiter) setRateLimitHeaders(c *gin.Context, keyInfo *models.APIKeyPublic, result *RateLimitResult) {
	// X-RateLimit-Limit: лимит запросов в минуту
	c.Header("X-RateLimit-Limit", strconv.FormatInt(int64(keyInfo.RateLimits.RequestsPerMinute), 10))

	// X-RateLimit-Remaining: оставшиеся запросы
	if result.Remaining >= 0 {
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(result.Remaining, 10))
	}

	// X-RateLimit-Reset: время сброса (Unix timestamp)
	if !result.ResetTime.IsZero() {
		c.Header("X-RateLimit-Reset", strconv.FormatInt(result.ResetTime.Unix(), 10))
	} else {
		// Если точное время неизвестно, используем текущее время + 1 минута
		resetTime := time.Now().Add(time.Minute)
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
	}
}

// GetRateLimitStatus возвращает статус rate limiting для API ключа
func (l *Limiter) GetRateLimitStatus(keyID string) map[string]interface{} {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	keyLimiter, exists := l.limiters[keyID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
		}
	}

	status := map[string]interface{}{
		"exists":    true,
		"key_id":    keyLimiter.KeyID,
		"key_name":  keyLimiter.KeyName,
		"last_used": keyLimiter.LastUsed.Format(time.RFC3339),
		"stats":     keyLimiter.Stats,
	}

	// Добавляем информацию о remaining tokens/requests
	if keyLimiter.RequestsPerMin != nil {
		status["remaining_requests_minute"] = int64(keyLimiter.RequestsPerMin.Tokens())
	}

	if keyLimiter.RequestsPerHour != nil {
		status["remaining_requests_hour"] = keyLimiter.RequestsPerHour.GetRemaining()
	}

	if keyLimiter.RequestsPerDay != nil {
		status["remaining_requests_day"] = keyLimiter.RequestsPerDay.GetRemaining()
	}

	if keyLimiter.TokensPerMin != nil {
		status["remaining_tokens_minute"] = keyLimiter.TokensPerMin.GetRemaining()
	}

	if keyLimiter.TokensPerDay != nil {
		status["remaining_tokens_day"] = keyLimiter.TokensPerDay.GetRemaining()
	}

	return status
}

// ResetLimiter сбрасывает rate limiter для API ключа
func (l *Limiter) ResetLimiter(keyID string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if limiter, exists := l.limiters[keyID]; exists {
		// Сбрасываем все counters
		if limiter.RequestsPerHour != nil {
			limiter.RequestsPerHour.tokens = limiter.RequestsPerHour.Limit
			limiter.RequestsPerHour.windowStart = time.Now()
		}

		if limiter.RequestsPerDay != nil {
			limiter.RequestsPerDay.tokens = limiter.RequestsPerDay.Limit
			limiter.RequestsPerDay.windowStart = time.Now()
		}

		if limiter.TokensPerMin != nil {
			limiter.TokensPerMin.tokens = limiter.TokensPerMin.Limit
			limiter.TokensPerMin.windowStart = time.Now()
		}

		if limiter.TokensPerDay != nil {
			limiter.TokensPerDay.tokens = limiter.TokensPerDay.Limit
			limiter.TokensPerDay.windowStart = time.Now()
		}

		l.logger.WithField("key_id", keyID).Info("Rate limiter reset for API key")
	}
}

