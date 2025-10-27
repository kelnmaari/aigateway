// Package middleware provides HTTP middleware for metrics collection
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"aigateway/internal/metrics"
)

// PrometheusMetrics middleware для сбора метрик HTTP запросов
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		if metrics.DefaultMetrics == nil {
			c.Next()
			return
		}

		// Получаем endpoint path
		endpoint := c.FullPath()
		if endpoint == "" {
			endpoint = c.Request.URL.Path
		}

		// Увеличиваем счетчик активных запросов
		metrics.DefaultMetrics.IncHTTPRequestsInFlight(endpoint)
		defer metrics.DefaultMetrics.DecHTTPRequestsInFlight(endpoint)

		// Засекаем время начала
		start := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Вычисляем длительность
		duration := time.Since(start)

		// Получаем размер ответа
		responseSize := c.Writer.Size()
		if responseSize < 0 {
			responseSize = 0
		}

		// Записываем метрики
		metrics.DefaultMetrics.RecordHTTPRequest(
			c.Request.Method,
			endpoint,
			c.Writer.Status(),
			duration,
			responseSize,
		)

		// Если есть API key в контексте, записываем метрику для ключа
		if keyID, exists := c.Get("key_id"); exists {
			if keyIDStr, ok := keyID.(string); ok {
				metrics.DefaultMetrics.RecordAPIKeyRequest(keyIDStr, "", endpoint, 0)
			}
		}
	}
}

