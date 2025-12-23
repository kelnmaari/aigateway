// Package middleware provides authentication middleware for database-backed API keys
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// APIKeyDBAuth middleware для аутентификации по API ключам из базы данных
// Также проверяет bootstrap admin_key из конфигурации
func APIKeyDBAuth(cfg *config.Config, db storage.Database, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем API ключ из заголовков
		apiKey := c.GetHeader("x-api-key")
		if apiKey == "" {
			// Также проверяем Authorization: Bearer <api-key>
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			logger.Debug("API key not provided")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key required. Provide x-api-key header or Bearer token",
					"type":    "authentication_error",
					"code":    "api_key_missing",
				},
			})
			c.Abort()
			return
		}

		// 🔑 ПРИОРИТЕТ 1: Проверяем Bootstrap Admin Key из конфигурации
		if cfg != nil && cfg.Auth.AdminKey != "" && apiKey == cfg.Auth.AdminKey {
			logger.WithField("key_type", "bootstrap_admin").Info("Bootstrap admin key authenticated")

			// Устанавливаем контекст для admin key
			c.Set("auth_type", "bootstrap_admin")
			c.Set("api_key_id", "bootstrap_admin_key")
			c.Set("is_admin", true)

			c.Next()
			return
		}

		// Извлекаем key_id из ключа (формат: sk-<key_id>-<random>)
		keyID := extractKeyID(apiKey)
		if keyID == "" {
			logger.Warn("Invalid API key format")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key format",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// Получаем ключ из БД
		storedKey, err := db.GetAPIKey(ctx, keyID)
		if err != nil {
			logger.WithError(err).Warn("API key not found in database")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		// Проверяем статус ключа
		if storedKey.Status != models.APIKeyStatusActive {
			logger.WithField("key_id", keyID).Warn("API key is not active")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key is disabled or revoked",
					"type":    "authentication_error",
					"code":    "api_key_disabled",
				},
			})
			c.Abort()
			return
		}

		// Проверяем срок действия
		if storedKey.ExpiresAt != nil && storedKey.ExpiresAt.Before(time.Now()) {
			logger.WithField("key_id", keyID).Warn("API key has expired")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "API key has expired",
					"type":    "authentication_error",
					"code":    "api_key_expired",
				},
			})
			c.Abort()
			return
		}

		// Проверяем хеш ключа
		if err := bcrypt.CompareHashAndPassword([]byte(storedKey.KeyHash), []byte(apiKey)); err != nil {
			logger.WithField("key_id", keyID).Warn("API key hash mismatch")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Invalid API key",
					"type":    "authentication_error",
					"code":    "invalid_api_key",
				},
			})
			c.Abort()
			return
		}

		// Аутентификация успешна - сохраняем информацию в контекст
		c.Set("api_key_id", storedKey.ID)
		c.Set("user_id", storedKey.UserID)
		if storedKey.TenantID != nil {
			c.Set("tenant_id", *storedKey.TenantID)
		}
		c.Set("auth_type", "api_key_db")

		// Обновляем время последнего использования (в фоне)
		go func() {
			updateCtx, updateCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer updateCancel()

			now := time.Now()
			storedKey.LastUsedAt = &now
			if err := db.UpdateAPIKey(updateCtx, storedKey); err != nil {
				logger.WithError(err).Warn("Failed to update API key last_used_at")
			}
		}()

		logger.WithFields(logrus.Fields{
			"key_id":  storedKey.ID,
			"user_id": storedKey.UserID,
		}).Debug("API key authentication successful")

		c.Next()
	}
}

// extractKeyID извлекает ID ключа из полного API ключа
// Формат: sk-<key_id>-<random_suffix>
func extractKeyID(apiKey string) string {
	parts := strings.Split(apiKey, "-")
	if len(parts) < 3 || parts[0] != "sk" {
		return ""
	}

	// Новый формат: sk-proj-<keyid>-<random>
	// parts[0] = "sk", parts[1] = "proj", parts[2] = keyid, parts[3+] = random
	if parts[1] == "proj" && len(parts) >= 4 {
		return parts[2]
	}

	// Старый формат: sk-<keyid>-<random>
	return parts[1]
}
