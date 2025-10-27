// Package middleware provides usage tracking middleware
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// UsageTracking middleware записывает API usage в базу данных
func UsageTracking(db storage.Database, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем non-API endpoints
		if !shouldTrackUsage(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Засекаем время начала
		startTime := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Собираем данные после обработки
		duration := time.Since(startTime).Milliseconds()

		// Получаем user_id и api_key_id из контекста
		userID, _ := c.Get("user_id")
		apiKeyID, _ := c.Get("api_key_id")
		tenantID, _ := c.Get("tenant_id")

		// Получаем модель из контекста или request body
		model := extractModel(c)

		// Определяем успешность запроса
		statusCode := c.Writer.Status()
		success := statusCode >= 200 && statusCode < 400

		// Извлекаем токены из контекста (устанавливаются handler'ами)
		promptTokens := extractIntFromContext(c, "prompt_tokens")
		completionTokens := extractIntFromContext(c, "completion_tokens")
		totalTokens := extractIntFromContext(c, "total_tokens")

		// DEBUG: Логируем извлеченные данные
		logger.WithFields(logrus.Fields{
			"endpoint":          c.Request.URL.Path,
			"user_id":           userID,
			"api_key_id":        apiKeyID,
			"model":             model,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
			"status_code":       statusCode,
			"success":           success,
		}).Debug("Usage tracking: extracted data from context")

		// Создаем запись usage
		usage := &models.APIUsage{
			ID:       uuid.New().String(),
			Endpoint: c.Request.URL.Path,
			Method:   c.Request.Method,
			Model:    model,

			StatusCode:   statusCode,
			Success:      success,
			ErrorMessage: extractErrorMessage(c),

			// Токены из контекста (устанавливаются handler'ами после получения ответа)
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,

			DurationMS: duration,
			CreatedAt:  startTime,

			UserAgent:      c.Request.UserAgent(),
			IPAddress:      c.ClientIP(),
			ConversationID: extractConversationID(c),
			Metadata:       extractMetadata(c),
		}

		// Устанавливаем владельца
		if userID != nil {
			// user_id может быть string (от JWT) или *string (от API key)
			switch v := userID.(type) {
			case string:
				usage.UserID = v
			case *string:
				if v != nil {
					usage.UserID = *v
				}
			}
		}

		// API Key ID (nullable для JWT auth) - v1.5.12: BUG-03 fix
		if apiKeyID != nil {
			keyIDStr := apiKeyID.(string)
			usage.APIKeyID = &keyIDStr
		} else {
			// Если нет API key (JWT auth), оставляем NULL вместо "jwt_auth"/"unknown"
			usage.APIKeyID = nil
		}

		if tenantID != nil {
			// tenant_id тоже может быть string или *string
			switch v := tenantID.(type) {
			case string:
				usage.TenantID = &v
			case *string:
				usage.TenantID = v
			}
		}

		// Export metrics to Prometheus (v1.11.6+)
		// Экспортируем метрики синхронно для немедленной доступности
		apiKeyIDForMetrics := "unknown"
		if usage.APIKeyID != nil {
			apiKeyIDForMetrics = *usage.APIKeyID
		} else if usage.UserID != "" {
			// For JWT auth, use user_id as identifier
			apiKeyIDForMetrics = "jwt:" + usage.UserID
		}
		
		metrics.RecordAPIUsage(
			apiKeyIDForMetrics,
			model,
			promptTokens,
			completionTokens,
			success,
		)
		
		// Also record model request duration
		if model != "" && model != "unknown" {
			metrics.RecordModelRequest(model, time.Duration(duration)*time.Millisecond)
		}
		
		// Record model errors if any
		if !success && model != "" {
			errorType := "unknown"
			if statusCode >= 500 {
				errorType = "server_error"
			} else if statusCode == 404 {
				errorType = "not_found"
			} else if statusCode == 400 {
				errorType = "bad_request"
			} else if statusCode == 401 || statusCode == 403 {
				errorType = "auth_error"
			}
			metrics.RecordModelError(model, errorType)
		}

		// Записываем в БД асинхронно (не блокируем ответ)
		// Важно: создаем новый контекст, т.к. c.Request.Context() отменяется после завершения запроса
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := db.RecordAPIUsage(ctx, usage); err != nil {
				logger.WithError(err).Warn("Failed to record API usage")
			}
		}()
	}
}

// shouldTrackUsage определяет нужно ли трекать этот endpoint
func shouldTrackUsage(path string) bool {
	// Трекаем только OpenAI API endpoints
	trackPrefixes := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/embeddings",
		"/v1/models",
	}

	for _, prefix := range trackPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// extractModel извлекает модель из request
func extractModel(c *gin.Context) string {
	// Пытаемся получить из контекста (устанавливается в chat handler)
	if model, exists := c.Get("model"); exists {
		return model.(string)
	}

	// Для /v1/models возвращаем пустую строку
	if c.Request.URL.Path == "/v1/models" {
		return ""
	}

	return "unknown"
}

// extractErrorMessage извлекает сообщение об ошибке
func extractErrorMessage(c *gin.Context) string {
	if c.Writer.Status() >= 400 {
		// Пытаемся получить из контекста
		if err, exists := c.Get("error"); exists {
			return err.(string)
		}
		// Возвращаем общее сообщение на основе статус кода
		return http.StatusText(c.Writer.Status())
	}
	return ""
}

// extractConversationID извлекает conversation_id из контекста
func extractConversationID(c *gin.Context) *string {
	if convID, exists := c.Get("conversation_id"); exists {
		convIDStr := convID.(string)
		return &convIDStr
	}
	return nil
}

// extractMetadata извлекает дополнительные метаданные
func extractMetadata(c *gin.Context) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Добавляем auth type
	if authType, exists := c.Get("auth_type"); exists {
		metadata["auth_type"] = authType
	}

	return metadata
}

// extractIntFromContext извлекает int значение из контекста
func extractIntFromContext(c *gin.Context, key string) int {
	if value, exists := c.Get(key); exists {
		if intVal, ok := value.(int); ok {
			return intVal
		}
	}
	return 0
}

