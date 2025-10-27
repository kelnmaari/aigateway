package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/jwt"
)

// SSEAuthMiddleware аутентификация для SSE endpoints
// Принимает JWT токен через query параметр ?token=xxx
func SSEAuthMiddleware(jwtManager *jwt.Manager, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пытаемся получить токен из query параметра
		token := c.Query("token")

		// Если нет в query, пробуем из header (для обратной совместимости)
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					token = parts[1]
				}
			}
		}

		if token == "" {
			logger.Warn("SSE: Missing token in query or header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication token"})
			c.Abort()
			return
		}

		// Валидация токена
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			logger.WithError(err).Warn("SSE: Invalid token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Сохраняем данные пользователя в контекст
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("tenant_ids", claims.TenantIDs)

		logger.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
		}).Debug("SSE: User authenticated")

		c.Next()
	}
}

