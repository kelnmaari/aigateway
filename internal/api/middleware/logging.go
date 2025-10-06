// Package middleware provides HTTP middleware for Ollama-OpenAI Proxy
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingConfig конфигурация middleware логирования
type LoggingConfig struct {
	Logger            *logrus.Logger
	SkipPaths         []string // Пути, которые не нужно логировать
	EnableRequestBody bool     // Логировать тело запроса (опасно для production)
}

// RequestLogging создает middleware для structured логирования HTTP запросов
func RequestLogging(config LoggingConfig) gin.HandlerFunc {
	logger := config.Logger
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Пропускаем логирование для определенных путей
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Записываем время начала
		start := time.Now()
		path := c.Request.URL.Path
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

		// Создаем structured log entry
		entry := logger.WithFields(logrus.Fields{
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

		// Добавляем дополнительные поля если есть ошибки
		if len(c.Errors) > 0 {
			entry = entry.WithField("errors", c.Errors.String())
		}

		// Определяем уровень логирования по статус коду
		switch {
		case statusCode >= 500:
			entry.Error("HTTP request processed with server error")
		case statusCode >= 400:
			entry.Warn("HTTP request processed with client error")
		case statusCode >= 300:
			entry.Info("HTTP request processed with redirect")
		default:
			entry.Info("HTTP request processed successfully")
		}
	}
}

// Recovery создает middleware для обработки panic с логированием
func Recovery(logger *logrus.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = logrus.StandardLogger()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.WithFields(logrus.Fields{
			"panic":     recovered,
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"client_ip": c.ClientIP(),
		}).Error("Panic recovered in HTTP handler")

		c.AbortWithStatus(500)
	})
}
