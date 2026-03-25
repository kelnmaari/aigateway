// Package middleware provides HTTP middleware for Gin
package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

// TenantMembershipMiddleware проверяет, является ли пользователь членом tenant
// Использует параметры маршрута :id или :tenant_id
func TenantMembershipMiddleware(db storage.Database, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by JWTAuth middleware)
		userID, exists := c.Get("userID")
		if !exists {
			logger.Debug("User ID not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok {
			logger.Error("User ID is not a string")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			c.Abort()
			return
		}

		// Try to get tenant ID from different param names
		tenantID := c.Param("id")
		if tenantID == "" {
			tenantID = c.Param("tenant_id")
		}

		if tenantID == "" {
			logger.Debug("Tenant ID not found in route params")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tenant ID required",
			})
			c.Abort()
			return
		}

		// Check if user is a member of the tenant
		member, err := db.GetTenantMember(c.Request.Context(), tenantID, userIDStr)
		if err != nil {
			logger.WithError(err).Error("Failed to check tenant membership")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify access",
			})
			c.Abort()
			return
		}

		if member == nil {
			logger.WithFields(logrus.Fields{
				"user_id":   userIDStr,
				"tenant_id": tenantID,
			}).Warn("Access denied: user is not a member of tenant")

			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied: you are not a member of this tenant",
			})
			c.Abort()
			return
		}

		// Store tenant membership info in context for later use
		c.Set("tenant_id", tenantID)
		c.Set("tenant_role", member.Role)
		c.Set("tenant_member", member)

		logger.WithFields(logrus.Fields{
			"user_id":   userIDStr,
			"tenant_id": tenantID,
			"role":      member.Role,
		}).Debug("Tenant membership verified")

		c.Next()
	}
}

// RequireTenantRole middleware проверяет, что у пользователя есть определенная роль в tenant
func RequireTenantRole(db storage.Database, logger *logrus.Logger, allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant role from context (set by TenantMembershipMiddleware)
		tenantRole, exists := c.Get("tenant_role")
		if !exists {
			logger.Debug("Tenant role not found in context - membership check not performed")
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Access denied",
			})
			c.Abort()
			return
		}

		roleStr, ok := tenantRole.(string)
		if !ok {
			logger.Error("Tenant role is not a string")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			c.Abort()
			return
		}

		// Check if user's role is in allowed roles
		if slices.Contains(allowedRoles, roleStr) {
			c.Next()
			return
		}

		logger.WithFields(logrus.Fields{
			"user_role":     roleStr,
			"allowed_roles": allowedRoles,
		}).Warn("Access denied: insufficient permissions")

		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied: insufficient permissions",
		})
		c.Abort()
	}
}
