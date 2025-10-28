package middleware

import (
	"context"
	"net/http"
	"time"

	servicerouter "aigateway/internal/services/router"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ModelRouterMiddleware middleware для smart routing model requests
type ModelRouterMiddleware struct {
	router *servicerouter.ModelRouter
	logger *logrus.Logger
}

// NewModelRouterMiddleware создает новый ModelRouterMiddleware
func NewModelRouterMiddleware(router *servicerouter.ModelRouter, logger *logrus.Logger) *ModelRouterMiddleware {
	return &ModelRouterMiddleware{
		router: router,
		logger: logger,
	}
}

// contextKey тип для ключей context
type contextKey string

const (
	// RouteResultKey ключ для сохранения результата маршрутизации в context
	RouteResultKey contextKey = "route_result"
)

// RouteModel middleware который маршрутизирует запрос к нужному provider
func (m *ModelRouterMiddleware) RouteModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем model из request body или query params
		modelID := m.extractModelID(c)
		if modelID == "" {
			c.Next() // Нет model ID - пропускаем маршрутизацию
			return
		}

		// Конфигурация маршрутизации
		config := servicerouter.RouteConfig{
			ModelID:        modelID,
			RequireHealthy: true,  // Требуем healthy providers
			AllowFallback:  true,  // Разрешаем fallback
			Timeout:        30 * time.Second,
		}

		// Маршрутизируем запрос
		startTime := time.Now()
		result, err := m.router.Route(c.Request.Context(), config)
		routingDuration := time.Since(startTime)

		if err != nil {
			m.logger.WithError(err).WithFields(logrus.Fields{
				"model_id": modelID,
				"duration": routingDuration,
			}).Error("Model routing failed")

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "No available provider for the requested model",
					"type":    "service_unavailable",
					"details": err.Error(),
				},
			})
			c.Abort()
			return
		}

		// Сохраняем результат маршрутизации в context для использования в handlers
		ctx := context.WithValue(c.Request.Context(), RouteResultKey, result)
		c.Request = c.Request.WithContext(ctx)

		// Добавляем метаданные маршрутизации в response headers
		c.Header("X-Provider-ID", result.ProviderID)
		c.Header("X-Provider-Type", string(result.Provider.GetType()))
		if result.IsFallback {
			c.Header("X-Provider-Fallback", "true")
			c.Header("X-Fallback-Reason", result.FallbackReason)
		}

		// Логируем успешную маршрутизацию
		m.logger.WithFields(logrus.Fields{
			"model_id":      modelID,
			"provider_id":   result.ProviderID,
			"provider_type": result.Provider.GetType(),
			"is_fallback":   result.IsFallback,
			"duration":      routingDuration,
		}).Info("Model request routed successfully")

		c.Next()
	}
}

// extractModelID извлекает model ID из request
func (m *ModelRouterMiddleware) extractModelID(c *gin.Context) string {
	// Для chat completions
	if c.Request.Method == "POST" && c.Request.URL.Path == "/v1/chat/completions" {
		var req struct {
			Model string `json:"model"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.Model != "" {
			return req.Model
		}
	}

	// Для embeddings
	if c.Request.Method == "POST" && c.Request.URL.Path == "/v1/embeddings" {
		var req struct {
			Model string `json:"model"`
		}
		if err := c.ShouldBindJSON(&req); err == nil && req.Model != "" {
			return req.Model
		}
	}

	// Query parameter fallback
	if model := c.Query("model"); model != "" {
		return model
	}

	return ""
}

// GetRouteResult извлекает RouteResult из context (helper для handlers)
func GetRouteResult(ctx context.Context) (*servicerouter.RouteResult, bool) {
	result, ok := ctx.Value(RouteResultKey).(*servicerouter.RouteResult)
	return result, ok
}

// RouteResultLogger middleware для логирования деталей маршрутизации после обработки запроса
func (m *ModelRouterMiddleware) RouteResultLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // Сначала обрабатываем запрос

		// После обработки логируем результат с provider info
		result, ok := GetRouteResult(c.Request.Context())
		if !ok {
			return // Нет route result - пропускаем
		}

		// Логируем финальный результат с metrics
		m.logger.WithFields(logrus.Fields{
			"model_id":         result.Model.ModelID,
			"provider_id":      result.ProviderID,
			"provider_type":    result.Provider.GetType(),
			"is_fallback":      result.IsFallback,
			"status_code":      c.Writer.Status(),
			"response_size":    c.Writer.Size(),
			"attempted_providers": result.AttemptedProviders,
		}).Info("Request completed with routing info")
	}
}

// ProviderHealthCheck middleware который проверяет health provider перед обработкой
func (m *ModelRouterMiddleware) ProviderHealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		result, ok := GetRouteResult(c.Request.Context())
		if !ok {
			c.Next() // Нет route result - пропускаем health check
			return
		}

		// Проверяем health provider перед обработкой запроса
		healthCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := result.Provider.HealthCheck(healthCtx); err != nil {
			m.logger.WithError(err).WithFields(logrus.Fields{
				"provider_id":   result.ProviderID,
				"provider_type": result.Provider.GetType(),
			}).Warn("Provider health check failed during request")

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"message": "Provider is currently unavailable",
					"type":    "provider_unavailable",
					"provider": result.ProviderID,
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

