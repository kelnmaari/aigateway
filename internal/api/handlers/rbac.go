// Package handlers provides RBAC API handlers
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/services/rbac"
	"ollama-openai-proxy/internal/storage"
)

// RBACHandler handles RBAC-related HTTP requests
type RBACHandler struct {
	db          storage.Database
	rbacService *rbac.Service
	logger      *logrus.Logger
}

// NewRBACHandler создает новый RBAC handler
func NewRBACHandler(db storage.Database, rbacService *rbac.Service, logger *logrus.Logger) *RBACHandler {
	return &RBACHandler{
		db:          db,
		rbacService: rbacService,
		logger:      logger,
	}
}

// ========================================
// Permissions Endpoints
// ========================================

// ListPermissions возвращает список всех системных разрешений
// GET /api/admin/rbac/permissions
func (h *RBACHandler) ListPermissions(c *gin.Context) {
	permissions, err := h.db.ListPermissions(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list permissions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve permissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// ========================================
// Roles Endpoints
// ========================================

// ListRoles возвращает список ролей (опционально для tenant)
// GET /api/admin/rbac/roles?tenant_id=xxx
func (h *RBACHandler) ListRoles(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	var tenantIDPtr *string
	if tenantID != "" {
		tenantIDPtr = &tenantID
	}

	roles, err := h.db.ListRoles(c.Request.Context(), tenantIDPtr)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list roles")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve roles",
		})
		return
	}

	// Для каждой роли получаем permissions (optional, для полного вывода)
	if c.Query("include_permissions") == "true" {
		for i, role := range roles {
			permissions, err := h.db.GetRolePermissions(c.Request.Context(), role.ID)
			if err != nil {
				h.logger.WithError(err).WithField("role_id", role.ID).Warn("Failed to get role permissions")
				continue
			}
			roles[i].Permissions = make([]models.RBACPermission, len(permissions))
			for j, perm := range permissions {
				roles[i].Permissions[j] = *perm
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
		"count": len(roles),
	})
}

// CreateRole создает новую кастомную роль
// POST /api/admin/rbac/roles
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	DisplayName string   `json:"display_name" binding:"required"`
	Description string   `json:"description"`
	Scope       string   `json:"scope" binding:"required,oneof=global tenant"` // "global" or "tenant"
	TenantID    *string  `json:"tenant_id,omitempty"`                          // required if scope=tenant
	Permissions []string `json:"permissions,omitempty"`                        // permission IDs
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Validate scope + tenant_id
	if req.Scope == string(models.RoleScopeTenant) && req.TenantID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tenant_id required for tenant-scoped role",
		})
		return
	}

	// Create role
	role := &models.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Type:        string(models.RoleTypeCustom),
		Scope:       req.Scope,
		TenantID:    req.TenantID,
	}

	if err := h.db.CreateRole(c.Request.Context(), role); err != nil {
		h.logger.WithError(err).Error("Failed to create role")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create role",
		})
		return
	}

	// Assign permissions if provided
	for _, permID := range req.Permissions {
		if err := h.db.AssignPermissionToRole(c.Request.Context(), role.ID, permID); err != nil {
			h.logger.WithError(err).WithField("permission_id", permID).Warn("Failed to assign permission to role")
		}
	}

	// Return created role with permissions
	permissions, err := h.db.GetRolePermissions(c.Request.Context(), role.ID)
	if err == nil {
		role.Permissions = make([]models.RBACPermission, len(permissions))
		for i, perm := range permissions {
			role.Permissions[i] = *perm
		}
	}

	h.logger.WithField("role_id", role.ID).Info("Role created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"role": role,
	})
}

// GetRole возвращает информацию о роли
// GET /api/admin/rbac/roles/:id
func (h *RBACHandler) GetRole(c *gin.Context) {
	roleID := c.Param("id")

	role, err := h.db.GetRole(c.Request.Context(), roleID)
	if err != nil {
		h.logger.WithError(err).WithField("role_id", roleID).Error("Failed to get role")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "role not found",
		})
		return
	}

	// Get permissions
	permissions, err := h.db.GetRolePermissions(c.Request.Context(), roleID)
	if err == nil {
		role.Permissions = make([]models.RBACPermission, len(permissions))
		for i, perm := range permissions {
			role.Permissions[i] = *perm
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// UpdateRole обновляет роль (только display_name и description)
// PUT /api/admin/rbac/roles/:id
type UpdateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	roleID := c.Param("id")

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Get existing role
	role, err := h.db.GetRole(c.Request.Context(), roleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "role not found",
		})
		return
	}

	// Update fields
	role.DisplayName = req.DisplayName
	role.Description = req.Description

	if err := h.db.UpdateRole(c.Request.Context(), role); err != nil {
		h.logger.WithError(err).WithField("role_id", roleID).Error("Failed to update role")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update role",
		})
		return
	}

	h.logger.WithField("role_id", roleID).Info("Role updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// DeleteRole удаляет роль (только custom roles)
// DELETE /api/admin/rbac/roles/:id
func (h *RBACHandler) DeleteRole(c *gin.Context) {
	roleID := c.Param("id")

	if err := h.db.DeleteRole(c.Request.Context(), roleID); err != nil {
		h.logger.WithError(err).WithField("role_id", roleID).Error("Failed to delete role")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete role",
		})
		return
	}

	h.logger.WithField("role_id", roleID).Info("Role deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "role deleted successfully",
	})
}

// ========================================
// Role-Permission Mapping
// ========================================

// GetRolePermissions возвращает permissions роли
// GET /api/admin/rbac/roles/:id/permissions
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	roleID := c.Param("id")

	permissions, err := h.db.GetRolePermissions(c.Request.Context(), roleID)
	if err != nil {
		h.logger.WithError(err).WithField("role_id", roleID).Error("Failed to get role permissions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve role permissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role_id":     roleID,
		"permissions": permissions,
		"count":       len(permissions),
	})
}

// AssignPermissionToRole добавляет permission роли
// POST /api/admin/rbac/roles/:id/permissions
type AssignPermissionRequest struct {
	PermissionID string `json:"permission_id" binding:"required"`
}

func (h *RBACHandler) AssignPermissionToRole(c *gin.Context) {
	roleID := c.Param("id")

	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	if err := h.db.AssignPermissionToRole(c.Request.Context(), roleID, req.PermissionID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": req.PermissionID,
		}).Error("Failed to assign permission to role")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to assign permission",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"role_id":       roleID,
		"permission_id": req.PermissionID,
	}).Info("Permission assigned to role")

	c.JSON(http.StatusOK, gin.H{
		"message": "permission assigned successfully",
	})
}

// RemovePermissionFromRole удаляет permission у роли
// DELETE /api/admin/rbac/roles/:id/permissions/:permission_id
func (h *RBACHandler) RemovePermissionFromRole(c *gin.Context) {
	roleID := c.Param("id")
	permissionID := c.Param("permission_id")

	if err := h.db.RemovePermissionFromRole(c.Request.Context(), roleID, permissionID); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"role_id":       roleID,
			"permission_id": permissionID,
		}).Error("Failed to remove permission from role")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to remove permission",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"role_id":       roleID,
		"permission_id": permissionID,
	}).Info("Permission removed from role")

	c.JSON(http.StatusOK, gin.H{
		"message": "permission removed successfully",
	})
}

// ========================================
// User-Role Assignments
// ========================================

// GetUserRoles возвращает роли пользователя
// GET /api/admin/rbac/users/:id/roles
func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userID := c.Param("id")

	userRoles, err := h.db.GetUserRoles(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user roles")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve user roles",
		})
		return
	}

	// Опционально получаем полную информацию о ролях
	if c.Query("include_details") == "true" {
		type UserRoleDetails struct {
			UserRole *models.UserRole `json:"user_role"`
			Role     *models.Role     `json:"role,omitempty"`
		}

		details := make([]UserRoleDetails, 0, len(userRoles))
		for _, ur := range userRoles {
			detail := UserRoleDetails{UserRole: ur}

			role, err := h.db.GetRole(c.Request.Context(), ur.RoleID)
			if err == nil {
				detail.Role = role
			}

			details = append(details, detail)
		}

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"roles":   details,
			"count":   len(details),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"roles":   userRoles,
		"count":   len(userRoles),
	})
}

// AssignRoleToUser назначает роль пользователю
// POST /api/admin/rbac/users/:id/roles
type AssignRoleRequest struct {
	RoleID   string  `json:"role_id" binding:"required"`
	TenantID *string `json:"tenant_id,omitempty"` // For tenant-scoped roles
}

func (h *RBACHandler) AssignRoleToUser(c *gin.Context) {
	userID := c.Param("id")

	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	// Validate that role exists
	role, err := h.db.GetRole(c.Request.Context(), req.RoleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "role not found",
		})
		return
	}

	// Validate tenant_id for tenant-scoped roles
	if role.Scope == string(models.RoleScopeTenant) && req.TenantID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tenant_id required for tenant-scoped role",
		})
		return
	}

	userRole := &models.UserRole{
		UserID:   userID,
		RoleID:   req.RoleID,
		TenantID: req.TenantID,
	}

	if err := h.db.AssignRoleToUser(c.Request.Context(), userRole); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"role_id": req.RoleID,
		}).Error("Failed to assign role to user")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to assign role",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"role_id": req.RoleID,
	}).Info("Role assigned to user")

	c.JSON(http.StatusOK, gin.H{
		"message":   "role assigned successfully",
		"user_role": userRole,
	})
}

// RemoveRoleFromUser удаляет роль у пользователя
// DELETE /api/admin/rbac/users/:id/roles/:role_id?tenant_id=xxx
func (h *RBACHandler) RemoveRoleFromUser(c *gin.Context) {
	userID := c.Param("id")
	roleID := c.Param("role_id")

	tenantID := c.Query("tenant_id")
	var tenantIDPtr *string
	if tenantID != "" {
		tenantIDPtr = &tenantID
	}

	if err := h.db.RemoveRoleFromUser(c.Request.Context(), userID, roleID, tenantIDPtr); err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"user_id": userID,
			"role_id": roleID,
		}).Error("Failed to remove role from user")

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to remove role",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"role_id": roleID,
	}).Info("Role removed from user")

	c.JSON(http.StatusOK, gin.H{
		"message": "role removed successfully",
	})
}

// ========================================
// User Permissions (convenience endpoint)
// ========================================

// GetUserPermissions возвращает все permissions пользователя (computed from roles)
// GET /api/admin/rbac/users/:id/permissions
func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	userID := c.Param("id")

	permissions, err := h.rbacService.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user permissions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve user permissions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":     userID,
		"permissions": permissions,
		"count":       len(permissions),
	})
}

