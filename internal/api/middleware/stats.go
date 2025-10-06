// Package middleware provides HTTP middleware for the API
package middleware

import (
	"time"

	"ollama-openai-proxy/internal/api/handlers"

	"github.com/gin-gonic/gin"
)

// StatsMiddleware подсчитывает статистику запросов
func StatsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем health checks и stats эндпоинт
		skipPaths := []string{"/health", "/healthz", "/ready", "/api/stats"}
		for _, path := range skipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Увеличиваем счетчики
		handlers.GlobalStats.IncrementTotalRequests()
		handlers.GlobalStats.IncrementActiveRequests()

		// Засекаем время
		start := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Уменьшаем счетчик активных запросов
		handlers.GlobalStats.DecrementActiveRequests()

		// Обновляем статистику на основе результата
		duration := time.Since(start)
		handlers.GlobalStats.TotalDuration += duration

		if c.Writer.Status() >= 400 {
			handlers.GlobalStats.IncrementErrorRequests()
		} else {
			handlers.GlobalStats.IncrementSuccessRequests()
		}
	}
}
