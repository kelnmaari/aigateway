// Package handlers provides HTTP handlers for admin user management
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"aigateway/internal/auth/password"
	"aigateway/internal/models"
	auditService "aigateway/internal/services/audit"
	"aigateway/internal/storage"
)

// AdminUserHandler handles admin-level user management
type AdminUserHandler struct {
	db          storage.Database
	logger      *logrus.Logger
	auditLogger *auditService.AuditLogger
}

// NewAdminUserHandler creates a new admin user handler
func NewAdminUserHandler(db storage.Database, logger *logrus.Logger, auditLogger *auditService.AuditLogger) *AdminUserHandler {
	return &AdminUserHandler{
		db:          db,
		logger:      logger,
		auditLogger: auditLogger,
	}
}

// CreateUserRequest represents admin request to create a user
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name,omitempty"`
	IsAdmin  bool   `json:"is_admin"`
}

// UpdateUserRequest represents admin request to update a user
type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
	Status   *string `json:"status,omitempty"`
}

// ListUsers godoc
// @Summary List all users (admin only)
// @Description Gets a list of all users with enriched data (roles, tenants, auth_provider)
// @Tags Admin
// @Accept json
// @Produce json
// @Success 200 {array} models.UserWithDetails
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/admin/users [get]
// @Security BearerAuth
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	// Get users with enriched data (roles, tenants) (v2.2.2+)
	usersWithDetails, err := h.db.GetUsersWithDetails(c.Request.Context(), models.UserFilters{})
	if err != nil {
		h.logger.WithError(err).Error("Failed to list users with details")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}

	// Remove password hashes from response
	for _, userDetails := range usersWithDetails {
		userDetails.PasswordHash = ""
	}

	c.JSON(http.StatusOK, gin.H{"users": usersWithDetails, "total": len(usersWithDetails)})
}

// GetUser godoc
// @Summary Get user by ID (admin only)
// @Description Gets detailed information about a specific user
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.User
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/admin/users/{id} [get]
// @Security BearerAuth
func (h *AdminUserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Remove password hash
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// CreateUser godoc
// @Summary Create a new user (admin only)
// @Description Creates a new user account
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User details"
// @Success 201 {object} models.User
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/admin/users [post]
// @Security BearerAuth
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Validate username
	if err := password.ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate email
	if err := password.ValidateEmail(req.Email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password
	if err := password.Validate(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Create user
	now := time.Now()
	user := &models.User{
		ID:           fmt.Sprintf("user_%s", uuid.New().String()),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		IsAdmin:      req.IsAdmin,
		Status:       models.UserStatusActive, // Status is UserStatus (not pointer)
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save to database
	if err := h.db.CreateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Create personal tenant for user
	tenantSlug := fmt.Sprintf("%s-workspace", user.Username)
	tenant := &models.Tenant{
		ID:        fmt.Sprintf("tenant_%s", uuid.New().String()),
		Name:      fmt.Sprintf("%s's Workspace", user.Username),
		Slug:      tenantSlug,
		Type:      models.TenantTypePersonal,
		OwnerID:   user.ID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.db.CreateTenant(c.Request.Context(), tenant); err != nil {
		h.logger.WithError(err).Warn("Failed to create personal tenant for user")
		// Continue anyway - user is created
	} else {
		// Add user as owner
		member := &models.TenantMember{
			TenantID: tenant.ID,
			UserID:   user.ID,
			Role:     models.TenantRoleOwner, // Role is TenantRole (not pointer)
			JoinedAt: now,
		}
		if err := h.db.AddTenantMember(c.Request.Context(), member); err != nil {
			h.logger.WithError(err).Warn("Failed to add user to personal tenant")
		}
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"is_admin": user.IsAdmin,
	}).Info("User created by admin")

	// Audit log: User created
	adminUserID, _ := c.Get("user_id")
	if h.auditLogger != nil {
		_ = h.auditLogger.LogUserCreated(c.Request.Context(), adminUserID.(string), user.ID, user.Username, c.ClientIP())
	}

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary Update user (admin only)
// @Description Updates user information
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "Update fields"
// @Success 200 {object} models.User
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/admin/users/{id} [put]
// @Security BearerAuth
func (h *AdminUserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	// Get existing user
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Parse request
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Apply updates
	if req.Email != nil {
		if err := password.ValidateEmail(*req.Email); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		user.Email = *req.Email
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}

	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if req.Status != nil {
		user.Status = models.UserStatus(*req.Status) // Status is UserStatus (not pointer)
	}

	user.UpdatedAt = time.Now()

	// Save to database
	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User updated by admin")

	// Remove password hash from response
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user (admin only)
// @Description Permanently deletes a user account
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} gin.H
// @Failure 401 {object} gin.H
// @Failure 403 {object} gin.H
// @Failure 404 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /api/admin/users/{id} [delete]
// @Security BearerAuth
func (h *AdminUserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Get admin user ID from context
	adminUserID, _ := c.Get("user_id")

	// Prevent self-deletion
	if userID == adminUserID.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete your own account"})
		return
	}

	// Check if user exists
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Delete user
	if err := h.db.DeleteUser(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete user"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"deleted_by": adminUserID,
	}).Info("User deleted by admin")

	// Audit log: User deleted (CRITICAL)
	if h.auditLogger != nil {
		_ = h.auditLogger.LogUserDeleted(c.Request.Context(), adminUserID.(string), userID, c.ClientIP())
	}

	c.Status(http.StatusNoContent)
}

// ResetUserPassword - reset user password (admin only)
func (h *AdminUserHandler) ResetUserPassword(c *gin.Context) {
	userID := c.Param("id")

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Get admin user ID from context
	adminUserID, _ := c.Get("user_id")

	// Check if user exists
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()

	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to update user password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reset password"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"reset_by": adminUserID,
	}).Info("User password reset by admin")

	c.JSON(http.StatusOK, gin.H{
		"message": "password reset successfully",
		"user_id": user.ID,
	})
}

// DisableUser - disable user account (admin only)
func (h *AdminUserHandler) DisableUser(c *gin.Context) {
	userID := c.Param("id")

	// Get admin user ID from context
	adminUserID, _ := c.Get("user_id")

	// Prevent self-disable
	if userID == adminUserID.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot disable your own account"})
		return
	}

	// Check if user exists
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Update status to inactive
	user.Status = models.UserStatusInactive
	user.UpdatedAt = time.Now()

	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to disable user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable user"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"username":    user.Username,
		"disabled_by": adminUserID,
	}).Info("User disabled by admin")

	// Audit log: User status changed to inactive
	if h.auditLogger != nil {
		_ = h.auditLogger.LogUserUpdated(c.Request.Context(), adminUserID.(string), userID, "status changed to inactive", c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user disabled successfully",
		"user":    toPublicUser(user),
	})
}

// EnableUser - enable user account (admin only)
func (h *AdminUserHandler) EnableUser(c *gin.Context) {
	userID := c.Param("id")

	// Get admin user ID from context
	adminUserID, _ := c.Get("user_id")

	// Check if user exists
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Update status to active
	user.Status = models.UserStatusActive
	user.UpdatedAt = time.Now()

	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to enable user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable user"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"enabled_by": adminUserID,
	}).Info("User enabled by admin")

	// Audit log: User status changed to active
	if h.auditLogger != nil {
		_ = h.auditLogger.LogUserUpdated(c.Request.Context(), adminUserID.(string), userID, "status changed to active", c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "user enabled successfully",
		"user":    toPublicUser(user),
	})
}

// toPublicUser removes sensitive fields from user model
func toPublicUser(user *models.User) gin.H {
	return gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"full_name":  user.FullName,
		"is_admin":   user.IsAdmin,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}
}

