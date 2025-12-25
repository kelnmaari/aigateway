// Package middleware provides HTTP middleware for Ollama-OpenAI Proxy
package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
)

// MetricsCollectorConfig конфигурация для MetricsCollector middleware
type MetricsCollectorConfig struct {
	Storage        *metrics.MetricsStorage
	Logger         *logrus.Logger // Основной логгер (для ошибок)
	DetailedLogger *logrus.Logger // Логгер для детальных метрик (в отдельный файл)
}

// MetricsCollector middleware собирает детальные метрики для каждого запроса
// и записывает их в MetricsStorage для historical analysis
func MetricsCollector(storage *metrics.MetricsStorage, logger *logrus.Logger) gin.HandlerFunc {
	return MetricsCollectorWithConfig(MetricsCollectorConfig{
		Storage: storage,
		Logger:  logger,
	})
}

// MetricsCollectorWithConfig middleware с расширенной конфигурацией
func MetricsCollectorWithConfig(cfg MetricsCollectorConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем health checks, stats endpoints, UI endpoints и статические файлы
		path := c.Request.URL.Path
		if path == "/health" || path == "/healthz" || path == "/ready" ||
			path == "/api/stats" || path == "/api/config" || path == "/metrics" ||
			path == "/api/metrics/history" || path == "/api/metrics/recent" ||
			path == "/api/metrics/stats" || path == "/api/metrics/stats/all" ||
			path == "/ws" ||
			strings.HasPrefix(path, "/api/ui/") { // v3.0.6+: Skip UI endpoints
			c.Next()
			return
		}

		startTime := time.Now()

		// Увеличиваем счетчик активных запросов
		cfg.Storage.IncrementActiveRequests()
		defer cfg.Storage.DecrementActiveRequests()

		// Измеряем размер запроса
		var requestSize int64
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestSize = int64(len(bodyBytes))
				// Восстанавливаем body для последующих обработчиков
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// Создаем custom ResponseWriter для захвата размера ответа
		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// Обрабатываем запрос
		c.Next()

		// Вычисляем latency
		latency := time.Since(startTime)
		responseSize := int64(blw.body.Len())
		statusCode := c.Writer.Status()

		// Записываем метрики (в миллисекундах с дробной частью)
		latencyMs := float64(latency.Microseconds()) / 1000.0
		cfg.Storage.RecordLatency(int64(latencyMs))
		cfg.Storage.RecordRequestSize(requestSize)
		cfg.Storage.RecordResponseSize(responseSize)

		// Подсчитываем успешные и ошибочные запросы
		if statusCode >= 200 && statusCode < 300 {
			cfg.Storage.IncrementRequestCount()
		} else if statusCode >= 400 {
			cfg.Storage.IncrementErrorCount()
		}

		// Детальные логи в отдельный файл (если настроен)
		logTarget := cfg.DetailedLogger
		if logTarget == nil {
			logTarget = cfg.Logger
		}
		if logTarget != nil {
			logTarget.WithFields(logrus.Fields{
				"method":        c.Request.Method,
				"path":          path,
				"status":        statusCode,
				"latency_ms":    latencyMs,
				"request_size":  requestSize,
				"response_size": responseSize,
			}).Debug("Metrics collected")
		}
	}
}

// bodyLogWriter оборачивает gin.ResponseWriter для захвата body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write захватывает данные response body
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString захватывает строковые данные response body
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

