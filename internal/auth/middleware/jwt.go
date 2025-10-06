// Package middleware provides authentication middleware for HTTP requests
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/jwt"
)

// JWTAuth middleware для проверки JWT токенов
func JWTAuth(jwtManager *jwt.Manager, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Debug("Missing Authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Check Bearer format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debug("Invalid Authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			logger.WithError(err).Debug("Token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("tenant_ids", claims.TenantIDs)
		c.Set("jwt_claims", claims)

		logger.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
		}).Debug("JWT authentication successful")

		c.Next()
	}
}

// OptionalJWTAuth middleware для опциональной аутентификации
// Не возвращает ошибку если токен отсутствует, но валидирует если присутствует
func OptionalJWTAuth(jwtManager *jwt.Manager, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided - continue without auth
			c.Next()
			return
		}

		// Token provided - validate it
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debug("Invalid Authorization header format in optional auth")
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			logger.WithError(err).Debug("Token validation failed in optional auth")
			c.Next()
			return
		}

		// Set user info in context if validation successful
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("tenant_ids", claims.TenantIDs)
		c.Set("jwt_claims", claims)

		logger.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
		}).Debug("Optional JWT authentication successful")

		c.Next()
	}
}

// GetUserID извлекает user_id из контекста
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	id, ok := userID.(string)
	return id, ok
}

// GetUsername извлекает username из контекста
func GetUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		return "", false
	}

	name, ok := username.(string)
	return name, ok
}

// GetEmail извлекает email из контекста
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get("email")
	if !exists {
		return "", false
	}

	e, ok := email.(string)
	return e, ok
}

// GetTenantIDs извлекает tenant_ids из контекста
func GetTenantIDs(c *gin.Context) ([]string, bool) {
	tenantIDs, exists := c.Get("tenant_ids")
	if !exists {
		return nil, false
	}

	ids, ok := tenantIDs.([]string)
	return ids, ok
}

// GetClaims извлекает полные JWT claims из контекста
func GetClaims(c *gin.Context) (*jwt.Claims, bool) {
	claims, exists := c.Get("jwt_claims")
	if !exists {
		return nil, false
	}

	jwtClaims, ok := claims.(*jwt.Claims)
	return jwtClaims, ok
}

// RequireJWTAuth проверяет что пользователь аутентифицирован через JWT
// Используется в handlers чтобы не дублировать проверку
func RequireJWTAuth(c *gin.Context) (string, bool) {
	userID, exists := GetUserID(c)
	if !exists || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return "", false
	}
	return userID, true
}
