// Package middleware provides CORS middleware for Ollama-OpenAI Proxy
package middleware

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ollama-openai-proxy/internal/config"
)

// CORS создает middleware для обработки CORS запросов
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := "*"
		methods := "GET, POST, PUT, DELETE, OPTIONS"
		headers := "Authorization, Content-Type, X-API-Key"
		maxAge := "86400"

		// Если CORS настроен в конфигурации
		if cfg.Server.CORS.Enabled {
			// Настройка разрешенных origins
			if len(cfg.Server.CORS.AllowedOrigins) > 0 {
				requestOrigin := c.GetHeader("Origin")
				if requestOrigin != "" {
					for _, allowedOrigin := range cfg.Server.CORS.AllowedOrigins {
						if allowedOrigin == "*" || allowedOrigin == requestOrigin {
							origin = allowedOrigin
							break
						}
					}
				} else {
					origin = cfg.Server.CORS.AllowedOrigins[0]
				}
			}

			// Настройка разрешенных методов
			if len(cfg.Server.CORS.AllowedMethods) > 0 {
				methods = strings.Join(cfg.Server.CORS.AllowedMethods, ", ")
			}

			// Настройка разрешенных заголовков
			if len(cfg.Server.CORS.AllowedHeaders) > 0 {
				headers = strings.Join(cfg.Server.CORS.AllowedHeaders, ", ")
			}

			// Настройка Max-Age
			if cfg.Server.CORS.MaxAge > 0 {
				maxAge = strconv.Itoa(cfg.Server.CORS.MaxAge)
			}
		}

		// Устанавливаем CORS заголовки
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", methods)
		c.Header("Access-Control-Allow-Headers", headers)
		c.Header("Access-Control-Max-Age", maxAge)

		// Для preflight запросов возвращаем 204
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
