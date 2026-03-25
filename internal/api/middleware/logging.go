// Package middleware provides HTTP middleware for Ollama-OpenAI Proxy
package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingConfig конфигурация middleware логирования
type LoggingConfig struct {
	Logger            *logrus.Logger
	DetailedLogger    *logrus.Logger // Логгер для детальных логов (в отдельный файл)
	SkipPaths         []string       // Пути, которые не нужно логировать
	EnableRequestBody bool           // Логировать тело запроса (опасно для production)
}

// RequestLogging создает middleware для structured логирования HTTP запросов
func RequestLogging(config LoggingConfig) gin.HandlerFunc {
	logger := config.Logger
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	// Отдельный логгер для детальных логов (если настроен)
	detailedLogger := config.DetailedLogger

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Пропускаем логирование для определенных путей и UI endpoints (v3.0.6+)
		if skipPaths[path] || strings.HasPrefix(path, "/api/ui/") {
			c.Next()
			return
		}

		// Записываем время начала
		start := time.Now()
		raw := c.Request.URL.RawQuery

		// Обрабатываем запрос
		c.Next()

		// Вычисляем параметры после обработки
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()
		userAgent := c.Request.UserAgent()

		if raw != "" {
			path = path + "?" + raw
		}

		// Детальные логи в отдельный файл (http.log) если настроен
		if detailedLogger != nil {
			detailedEntry := detailedLogger.WithFields(logrus.Fields{
				"timestamp":   start,
				"latency":     latency,
				"latency_ms":  float64(latency.Nanoseconds()) / 1e6,
				"method":      method,
				"path":        path,
				"status_code": statusCode,
				"body_size":   bodySize,
				"client_ip":   clientIP,
				"user_agent":  userAgent,
			})

			if len(c.Errors) > 0 {
				detailedEntry = detailedEntry.WithField("errors", c.Errors.String())
			}

			switch {
			case statusCode >= 500:
				detailedEntry.Error("HTTP request processed with server error")
			case statusCode >= 400:
				detailedEntry.Warn("HTTP request processed with client error")
			case statusCode >= 300:
				detailedEntry.Info("HTTP request processed with redirect")
			default:
				detailedEntry.Info("HTTP request processed successfully")
			}
		}

		// Краткие логи в основной файл (только для ошибок или если нет отдельного логгера)
		if detailedLogger == nil || statusCode >= 400 {
			entry := logger.WithFields(logrus.Fields{
				"method":    method,
				"path":      path,
				"status":    statusCode,
				"client_ip": clientIP,
				"latency":   latency.String(),
			})

			if len(c.Errors) > 0 {
				entry = entry.WithField("errors", c.Errors.String())
			}

			switch {
			case statusCode >= 500:
				entry.Error("HTTP 5xx")
			case statusCode >= 400:
				entry.Warn("HTTP 4xx")
			default:
				// Если нет отдельного логгера - логируем все; иначе - только ошибки
				if detailedLogger == nil {
					entry.Info("HTTP request")
				}
			}
		}
	}
}

// Recovery создает middleware для обработки panic с логированием
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.WithFields(logrus.Fields{
			"panic":     recovered,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"client_ip": c.ClientIP(),
		}).Error("Panic recovered in HTTP handler")

		c.AbortWithStatus(500)
	})
}
