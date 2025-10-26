// Package middleware provides RBAC middleware for Gin
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/services/rbac"
	"ollama-openai-proxy/internal/storage"
)

// RBACMiddleware provides RBAC (Role-Based Access Control) middleware for Gin
type RBACMiddleware struct {
	rbacService *rbac.Service
	db          storage.Database
	logger      *logrus.Logger
}

// NewRBACMiddleware создает новый RBAC middleware
func NewRBACMiddleware(rbacService *rbac.Service, db storage.Database, logger *logrus.Logger) *RBACMiddleware {
	return &RBACMiddleware{
		rbacService: rbacService,
		db:          db,
		logger:      logger,
	}
}

// RequirePermission проверяет наличие конкретного разрешения у пользователя
//
// Использование:
//
//	r.POST("/api/admin/api-keys", rbacMiddleware.RequirePermission("api_keys:create"), handler.CreateAPIKey)
//
// Поддерживает wildcards:
//   - "*:*" - любое разрешение
//   - "api_keys:*" - любое действие с api_keys
//   - "*:create" - create для любого ресурса
func (m *RBACMiddleware) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by JWTAuth middleware)
		userAny, exists := c.Get("user")
		if !exists {
			m.logger.Warn("RBAC: user not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		user, ok := userAny.(*models.User)
		if !ok {
			m.logger.Error("RBAC: invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}

		// Super admins bypass all checks
		if user.IsAdmin {
			c.Next()
			return
		}

		// Get tenant_id from query if present (for tenant-scoped permissions)
		tenantID := c.Query("tenant_id")
		var tenantIDPtr *string
		if tenantID != "" {
			tenantIDPtr = &tenantID
		}

		// Check permission
		hasPermission, err := m.rbacService.CheckPermission(c.Request.Context(), user.ID, permission, tenantIDPtr)
		if err != nil {
			m.logger.WithError(err).WithFields(logrus.Fields{
				"user_id":    user.ID,
				"permission": permission,
			}).Error("RBAC: failed to check permission")

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check permission",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			m.logger.WithFields(logrus.Fields{
				"user_id":    user.ID,
				"username":   user.Username,
				"permission": permission,
			}).Warn("RBAC: permission denied")

			c.JSON(http.StatusForbidden, gin.H{
				"error":      "permission denied",
				"permission": permission,
			})
			c.Abort()
			return
		}

		// Permission granted
		c.Next()
	}
}

// RequireAnyPermission проверяет наличие хотя бы одного из указанных разрешений
//
// Использование:
//
//	r.GET("/api/admin/dashboard", rbacMiddleware.RequireAnyPermission(
//	    "dashboard:read",
//	    "system:admin",
//	))
func (m *RBACMiddleware) RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userAny, exists := c.Get("user")
		if !exists {
			m.logger.Warn("RBAC: user not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		user, ok := userAny.(*models.User)
		if !ok {
			m.logger.Error("RBAC: invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}

		// Super admins bypass all checks
		if user.IsAdmin {
			c.Next()
			return
		}

		// Get tenant_id from query
		tenantID := c.Query("tenant_id")
		var tenantIDPtr *string
		if tenantID != "" {
			tenantIDPtr = &tenantID
		}

		// Check if user has ANY of the required permissions
		for _, permission := range permissions {
			hasPermission, err := m.rbacService.CheckPermission(c.Request.Context(), user.ID, permission, tenantIDPtr)
			if err != nil {
				m.logger.WithError(err).WithFields(logrus.Fields{
					"user_id":    user.ID,
					"permission": permission,
				}).Error("RBAC: failed to check permission")
				continue
			}

			if hasPermission {
				// User has at least one required permission
				c.Next()
				return
			}
		}

		// No permissions matched
		m.logger.WithFields(logrus.Fields{
			"user_id":     user.ID,
			"username":    user.Username,
			"permissions": permissions,
		}).Warn("RBAC: permission denied (requires any)")

		c.JSON(http.StatusForbidden, gin.H{
			"error":       "permission denied",
			"permissions": permissions,
			"message":     "requires at least one of the specified permissions",
		})
		c.Abort()
	}
}

// RequireAllPermissions проверяет наличие всех указанных разрешений
//
// Использование:
//
//	r.DELETE("/api/admin/tenants/:id", rbacMiddleware.RequireAllPermissions(
//	    "tenants:delete",
//	    "tenants:admin",
//	))
func (m *RBACMiddleware) RequireAllPermissions(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userAny, exists := c.Get("user")
		if !exists {
			m.logger.Warn("RBAC: user not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		user, ok := userAny.(*models.User)
		if !ok {
			m.logger.Error("RBAC: invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}

		// Super admins bypass all checks
		if user.IsAdmin {
			c.Next()
			return
		}

		// Get tenant_id from query
		tenantID := c.Query("tenant_id")
		var tenantIDPtr *string
		if tenantID != "" {
			tenantIDPtr = &tenantID
		}

		// Check ALL required permissions
		missingPermissions := []string{}
		for _, permission := range permissions {
			hasPermission, err := m.rbacService.CheckPermission(c.Request.Context(), user.ID, permission, tenantIDPtr)
			if err != nil {
				m.logger.WithError(err).WithFields(logrus.Fields{
					"user_id":    user.ID,
					"permission": permission,
				}).Error("RBAC: failed to check permission")

				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to check permissions",
				})
				c.Abort()
				return
			}

			if !hasPermission {
				missingPermissions = append(missingPermissions, permission)
			}
		}

		if len(missingPermissions) > 0 {
			// User missing some permissions
			m.logger.WithFields(logrus.Fields{
				"user_id":             user.ID,
				"username":            user.Username,
				"missing_permissions": missingPermissions,
			}).Warn("RBAC: permission denied (requires all)")

			c.JSON(http.StatusForbidden, gin.H{
				"error":               "permission denied",
				"missing_permissions": missingPermissions,
				"message":             "requires all specified permissions",
			})
			c.Abort()
			return
		}

		// User has all required permissions
		c.Next()
	}
}

// RequireRole проверяет наличие определённой роли у пользователя (by name)
//
// Использование:
//
//	r.GET("/api/admin/audit", rbacMiddleware.RequireRole("super_admin"))
//
// Note: Этот метод проверяет роль по имени. Для точной проверки прав используй RequirePermission.
func (m *RBACMiddleware) RequireRole(roleName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context
		userAny, exists := c.Get("user")
		if !exists {
			m.logger.Warn("RBAC: user not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})
			c.Abort()
			return
		}

		user, ok := userAny.(*models.User)
		if !ok {
			m.logger.Error("RBAC: invalid user type in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			c.Abort()
			return
		}

		// Get user roles
		userRoles, err := m.db.GetUserRoles(c.Request.Context(), user.ID)
		if err != nil {
			m.logger.WithError(err).WithField("user_id", user.ID).Error("RBAC: failed to get user roles")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check role",
			})
			c.Abort()
			return
		}

		// Check if user has the required role
		for _, userRole := range userRoles {
			// Get role details
			role, err := m.db.GetRole(c.Request.Context(), userRole.RoleID)
			if err != nil {
				m.logger.WithError(err).WithField("role_id", userRole.RoleID).Warn("RBAC: failed to get role")
				continue
			}

			if role.Name == roleName {
				// User has the required role
				c.Next()
				return
			}
		}

		// User doesn't have the required role
		m.logger.WithFields(logrus.Fields{
			"user_id":  user.ID,
			"username": user.Username,
			"role":     roleName,
		}).Warn("RBAC: role not found")

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "role required",
			"role":    roleName,
			"message": "requires specific role",
		})
		c.Abort()
	}
}

