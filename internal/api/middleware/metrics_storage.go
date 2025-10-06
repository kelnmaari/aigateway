package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"ollama-openai-proxy/internal/metrics"
)

// MetricsStorageMiddleware middleware для записи метрик в storage
func MetricsStorageMiddleware(storage *metrics.MetricsStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		if storage == nil {
			c.Next()
			return
		}

		start := time.Now()

		// Записываем активный запрос
		storage.Record(metrics.MetricTypeActiveRequests, 1, map[string]string{
			"endpoint": c.FullPath(),
			"method":   c.Request.Method,
		})

		// Обрабатываем запрос
		c.Next()

		// Вычисляем метрики после обработки
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		labels := map[string]string{
			"endpoint": c.FullPath(),
			"method":   c.Request.Method,
			"status":   string(rune(statusCode)),
		}

		// Записываем метрики
		storage.Record(metrics.MetricTypeRequestCount, 1, labels)
		storage.Record(metrics.MetricTypeRequestLatency, float64(duration.Microseconds()), labels)

		if statusCode >= 400 {
			storage.Record(metrics.MetricTypeErrorCount, 1, labels)
		}

		// Размеры запроса/ответа
		if c.Request.ContentLength > 0 {
			storage.Record(metrics.MetricTypeRequestSize, float64(c.Request.ContentLength), labels)
		}

		responseSize := c.Writer.Size()
		if responseSize > 0 {
			storage.Record(metrics.MetricTypeResponseSize, float64(responseSize), labels)
		}

		// Уменьшаем счетчик активных запросов
		storage.Record(metrics.MetricTypeActiveRequests, -1, labels)
	}
}
