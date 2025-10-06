// Package middleware provides HTTP middleware for admin access control
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/storage"
)

// RequireAdmin middleware ensures the authenticated user has admin privileges
// Must be used AFTER JWTAuth middleware
func RequireAdmin(db storage.Database, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user_id from context (set by JWTAuth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			logger.Warn("RequireAdmin: user_id not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			c.Abort()
			return
		}

		// Get user from database
		user, err := db.GetUser(c.Request.Context(), userID.(string))
		if err != nil {
			logger.WithError(err).Error("RequireAdmin: failed to get user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to verify admin status",
			})
			c.Abort()
			return
		}

		// Check if user is admin
		if !user.IsAdmin {
			logger.WithFields(logrus.Fields{
				"user_id":  user.ID,
				"username": user.Username,
			}).Warn("RequireAdmin: user is not admin")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "admin privileges required",
			})
			c.Abort()
			return
		}

		// User is admin - allow request
		logger.WithFields(logrus.Fields{
			"user_id":  user.ID,
			"username": user.Username,
		}).Debug("RequireAdmin: admin access granted")

		c.Next()
	}
}
