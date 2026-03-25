// Package middleware provides authentication middleware
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
)

// APIKeyAuth создает middleware для аутентификации API ключей
func APIKeyAuth(cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если аутентификация отключена, пропускаем
		if !cfg.Auth.Enabled {
			c.Next()
			return
		}

		// Извлекаем API ключ из заголовков
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			logger.Warn("Missing API key in request")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key required",
					"type":    "authentication_error",
					"code":    "missing_api_key",
				},
			})
			c.Abort()
			return
		}

		// TODO: Проверить валидность API ключа через API Key Manager
		// Пока для MVP просто логируем
		logger.WithFields(logrus.Fields{
			"api_key_prefix": apiKey[:min(8, len(apiKey))] + "...",
			"endpoint":       c.Request.URL.Path,
			"method":         c.Request.Method,
		}).Info("API key authentication (placeholder)")

		// TODO: Добавить проверку прав доступа к моделям
		// TODO: Добавить rate limiting

		// Version 2.4.0+: Update last_seen_at for device keys
		// Note: This is a placeholder for now. When proper API key validation is implemented,
		// this logic should be moved there to update last_seen_at only for verified device keys.
		// Storage layer UpdateAPIKeyLastSeen is implemented and ready to use.

		// Для MVP пропускаем все запросы
		c.Set("api_key", apiKey)
		c.Next()
	}
}

// extractAPIKey извлекает API ключ из заголовков запроса
func extractAPIKey(c *gin.Context) string {
	// Проверяем заголовок Authorization (Bearer token)
	auth := c.GetHeader("Authorization")
	if auth != "" && strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	// Проверяем заголовок X-API-Key
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// Проверяем параметр api_key в query
	queryKey := c.Query("api_key")
	if queryKey != "" {
		return queryKey
	}

	return ""
}

// RequireAuth создает middleware который требует аутентификации для определенных путей
func RequireAuth(cfg *config.Config, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Список путей, которые требуют аутентификации
		protectedPaths := []string{
			"/v1/chat/completions",
			"/v1/completions",
			"/v1/embeddings",
			"/admin/",
		}

		path := c.Request.URL.Path
		requiresAuth := false

		for _, protectedPath := range protectedPaths {
			if strings.HasPrefix(path, protectedPath) {
				requiresAuth = true
				break
			}
		}

		// Если путь не требует аутентификации, пропускаем
		if !requiresAuth || !cfg.Auth.Enabled {
			c.Next()
			return
		}

		// Проверяем наличие API ключа
		if _, exists := c.Get("api_key"); !exists {
			logger.WithField("path", path).Warn("Protected endpoint accessed without API key")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "This endpoint requires authentication",
					"type":    "authentication_error",
					"code":    "authentication_required",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
