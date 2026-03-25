// Package handlers provides HTTP handlers for tenant management endpoints
package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/middleware"
	"aigateway/internal/models"
	auditService "aigateway/internal/services/audit"
	"aigateway/internal/storage"
)

// TenantHandler обрабатывает tenant management запросы
type TenantHandler struct {
	db          storage.Database
	logger      *logrus.Logger
	auditLogger *auditService.AuditLogger
}

// NewTenantHandler создает новый Tenant Handler
func NewTenantHandler(db storage.Database, logger *logrus.Logger, auditLogger *auditService.AuditLogger) *TenantHandler {
	return &TenantHandler{
		db:          db,
		logger:      logger,
		auditLogger: auditLogger,
	}
}

// ListAllTenants возвращает список всех tenants в системе (только для админов)
// GET /api/admin/tenants
func (h *TenantHandler) ListAllTenants(c *gin.Context) {
	// List all tenants from database
	tenants, err := h.db.ListAllTenants(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to list all tenants")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list tenants",
		})
		return
	}

	h.logger.WithField("count", len(tenants)).Debug("Listed all tenants for admin")

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"count":   len(tenants),
	})
}

// CreateTenant создает новый organization tenant
// POST /api/tenants
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Name is required",
		})
		return
	}

	// Generate slug from name
	slug := generateSlug(req.Name)

	// Start transaction
	tx, err := h.db.BeginTx(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to start transaction")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create tenant",
		})
		return
	}
	defer tx.Rollback()

	// Create tenant
	tenant := &models.Tenant{
		ID:          fmt.Sprintf("tenant_%s", uuid.New().String()),
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		OwnerID:     userID,
		Type:        models.TenantTypeOrganization,
		Status:      models.TenantStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Settings: models.TenantSettings{
			MaxAPIKeys:       100,
			MaxConversations: 1000,
			ChatEnabled:      true,
			APIAccessEnabled: true,
		},
		Metadata: make(map[string]any), // Empty map for JSONB
	}

	if err := tx.CreateTenant(c.Request.Context(), tenant); err != nil {
		h.logger.WithError(err).Error("Failed to create tenant")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create tenant",
		})
		return
	}

	// Add creator as owner
	member := &models.TenantMember{
		TenantID:  tenant.ID,
		UserID:    userID,
		Role:      models.TenantRoleOwner,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]any), // Empty map for JSONB
	}

	if err := tx.AddTenantMember(c.Request.Context(), member); err != nil {
		h.logger.WithError(err).Error("Failed to add tenant owner")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create tenant",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		h.logger.WithError(err).Error("Failed to commit transaction")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create tenant",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenant.ID,
		"user_id":   userID,
	}).Info("Tenant created successfully")

	// Audit log: Tenant created
	if h.auditLogger != nil {
		_ = h.auditLogger.LogTenantCreated(c.Request.Context(), userID, tenant.ID, tenant.Name, c.ClientIP())
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenant возвращает информацию о tenant
// GET /api/tenants/:id
func (h *TenantHandler) GetTenant(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if user is member of this tenant
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this tenant",
		})
		return
	}

	// Get tenant
	tenant, err := h.db.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant": tenant,
		"role":   member.Role,
	})
}

// UpdateTenant обновляет tenant
// PUT /api/tenants/:id
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if user is owner or admin
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if member.Role != models.TenantRoleOwner && member.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can update tenant",
		})
		return
	}

	var req struct {
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get tenant
	tenant, err := h.db.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	// Update fields
	updated := false

	if req.Name != nil && *req.Name != tenant.Name {
		tenant.Name = *req.Name
		tenant.Slug = generateSlug(*req.Name)
		updated = true
	}

	if req.Description != nil && *req.Description != tenant.Description {
		tenant.Description = *req.Description
		updated = true
	}

	if !updated {
		c.JSON(http.StatusOK, gin.H{
			"message": "No changes made",
		})
		return
	}

	tenant.UpdatedAt = time.Now()

	// Save changes
	if err := h.db.UpdateTenant(c.Request.Context(), tenant); err != nil {
		h.logger.WithError(err).Error("Failed to update tenant")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update tenant",
		})
		return
	}

	h.logger.WithField("tenant_id", tenantID).Info("Tenant updated")

	// Audit log: Tenant updated
	if h.auditLogger != nil {
		_ = h.auditLogger.LogTenantUpdated(c.Request.Context(), userID, tenantID, "tenant information updated", c.ClientIP())
	}

	c.JSON(http.StatusOK, tenant)
}

// DeleteTenant удаляет tenant (только owner)
// DELETE /api/tenants/:id
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Get tenant
	tenant, err := h.db.GetTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tenant not found",
		})
		return
	}

	// Check if user is owner
	if tenant.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only tenant owner can delete it",
		})
		return
	}

	// Cannot delete personal tenant
	if tenant.Type == models.TenantTypePersonal {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete personal workspace",
		})
		return
	}

	// Delete tenant
	if err := h.db.DeleteTenant(c.Request.Context(), tenantID); err != nil {
		h.logger.WithError(err).Error("Failed to delete tenant")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete tenant",
		})
		return
	}

	h.logger.WithField("tenant_id", tenantID).Info("Tenant deleted")

	// Audit log: Tenant deleted (CRITICAL)
	if h.auditLogger != nil {
		_ = h.auditLogger.LogTenantDeleted(c.Request.Context(), userID, tenantID, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant deleted successfully",
	})
}

// ListMembers возвращает список участников tenant
// GET /api/tenants/:id/members
func (h *TenantHandler) ListMembers(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if user is member
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Get all members
	members, err := h.db.ListTenantMembers(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant members")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get members",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
		"count":   len(members),
	})
}

// SearchUsers ищет пользователей по username или email для добавления в tenant
// GET /api/tenants/:id/search-users?query=...
func (h *TenantHandler) SearchUsers(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if requester is owner or admin
	requesterMember, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || requesterMember == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if requesterMember.Role != models.TenantRoleOwner && requesterMember.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can search users",
		})
		return
	}

	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query parameter is required",
		})
		return
	}

	// Try to find user by username
	user, err := h.db.GetUserByUsername(c.Request.Context(), query)
	if err == nil && user != nil {
		// Check if user is already a member
		member, _ := h.db.GetTenantMember(c.Request.Context(), tenantID, user.ID)
		c.JSON(http.StatusOK, gin.H{
			"user":           user,
			"already_member": member != nil,
		})
		return
	}

	// Try to find user by email
	user, err = h.db.GetUserByEmail(c.Request.Context(), query)
	if err == nil && user != nil {
		// Check if user is already a member
		member, _ := h.db.GetTenantMember(c.Request.Context(), tenantID, user.ID)
		c.JSON(http.StatusOK, gin.H{
			"user":           user,
			"already_member": member != nil,
		})
		return
	}

	// User not found
	c.JSON(http.StatusNotFound, gin.H{
		"error": "User not found",
	})
}

// AddMember добавляет участника в tenant
// POST /api/tenants/:id/members
func (h *TenantHandler) AddMember(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if requester is owner or admin
	requesterMember, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || requesterMember == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if requesterMember.Role != models.TenantRoleOwner && requesterMember.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can add members",
		})
		return
	}

	var req struct {
		UserID   string            `json:"user_id,omitempty"`
		Username string            `json:"username,omitempty"`
		Email    string            `json:"email,omitempty"`
		Role     models.TenantRole `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "role is required",
		})
		return
	}

	// Find user by ID, username, or email
	var targetUser *models.User
	if req.UserID != "" {
		targetUser, err = h.db.GetUser(c.Request.Context(), req.UserID)
	} else if req.Username != "" {
		targetUser, err = h.db.GetUserByUsername(c.Request.Context(), req.Username)
	} else if req.Email != "" {
		targetUser, err = h.db.GetUserByEmail(c.Request.Context(), req.Email)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id, username, or email is required",
		})
		return
	}

	// Check if user was found
	if err != nil || targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Validate role
	if req.Role != models.TenantRoleMember && req.Role != models.TenantRoleAdmin && req.Role != models.TenantRoleViewer {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role. Must be member, admin, or viewer",
		})
		return
	}

	// Check if already a member
	existingMember, _ := h.db.GetTenantMember(c.Request.Context(), tenantID, targetUser.ID)
	if existingMember != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User is already a member of this tenant",
		})
		return
	}

	// Add member
	member := &models.TenantMember{
		TenantID:  tenantID,
		UserID:    targetUser.ID,
		Role:      req.Role,
		InvitedBy: userID,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]any), // Empty map for JSONB
	}

	if err := h.db.AddTenantMember(c.Request.Context(), member); err != nil {
		h.logger.WithError(err).Error("Failed to add tenant member")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add member",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"user_id":   targetUser.ID,
		"username":  targetUser.Username,
		"role":      req.Role,
	}).Info("Member added to tenant")

	// Return member with user info
	c.JSON(http.StatusCreated, gin.H{
		"member": member,
		"user":   targetUser,
	})
}

// UpdateMemberRole обновляет роль участника
// PUT /api/tenants/:id/members/:user_id
func (h *TenantHandler) UpdateMemberRole(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")
	targetUserID := c.Param("user_id")

	// Check if requester is owner or admin
	requesterMember, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || requesterMember == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if requesterMember.Role != models.TenantRoleOwner && requesterMember.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can update member roles",
		})
		return
	}

	var req struct {
		Role models.TenantRole `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "role is required",
		})
		return
	}

	// Get target member
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, targetUserID)
	if err != nil || member == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Member not found",
		})
		return
	}

	// Cannot change owner role
	if member.Role == models.TenantRoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot change owner role",
		})
		return
	}

	// Update role
	member.Role = req.Role
	member.UpdatedAt = time.Now()

	if err := h.db.UpdateTenantMember(c.Request.Context(), member); err != nil {
		h.logger.WithError(err).Error("Failed to update member role")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update member role",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"user_id":   targetUserID,
		"new_role":  req.Role,
	}).Info("Member role updated")

	c.JSON(http.StatusOK, member)
}

// RemoveMember удаляет участника из tenant
// DELETE /api/tenants/:id/members/:user_id
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")
	targetUserID := c.Param("user_id")

	// Check if requester is owner or admin
	requesterMember, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || requesterMember == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if requesterMember.Role != models.TenantRoleOwner && requesterMember.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can remove members",
		})
		return
	}

	// Get target member
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, targetUserID)
	if err != nil || member == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Member not found",
		})
		return
	}

	// Cannot remove owner
	if member.Role == models.TenantRoleOwner {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot remove tenant owner",
		})
		return
	}

	// Remove member
	if err := h.db.RemoveTenantMember(c.Request.Context(), tenantID, targetUserID); err != nil {
		h.logger.WithError(err).Error("Failed to remove member")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove member",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"user_id":   targetUserID,
	}).Info("Member removed from tenant")

	c.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}

// ListTenantAPIKeys возвращает список API ключей организации
// GET /api/tenants/:id/api-keys
func (h *TenantHandler) ListTenantAPIKeys(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if user is member of this tenant
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied to this tenant",
		})
		return
	}

	// Get tenant API keys
	keys, err := h.db.ListTenantAPIKeys(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant API keys")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get API keys",
		})
		return
	}

	// Remove sensitive key data
	for _, key := range keys {
		key.KeyHash = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": keys,
		"count":    len(keys),
	})
}

// CreateTenantAPIKey создает API ключ для организации
// POST /api/tenants/:id/api-keys
func (h *TenantHandler) CreateTenantAPIKey(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")

	// Check if user is owner or admin
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if member.Role != models.TenantRoleOwner && member.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can create API keys",
		})
		return
	}

	var req struct {
		Name        string             `json:"name" binding:"required"`
		Description string             `json:"description,omitempty"`
		Models      []string           `json:"models,omitempty"`
		Permissions []string           `json:"permissions,omitempty"`
		RateLimits  *models.RateLimits `json:"rate_limits,omitempty"`
		ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Generate key ID first
	keyID := models.GenerateAPIKeyID()

	// Generate API key with embedded ID
	plainKey, err := models.GenerateAPIKeyWithID(keyID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate API key",
		})
		return
	}

	// Hash the key
	keyHash, err := models.HashAPIKey(plainKey)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create API key",
		})
		return
	}

	// Set defaults
	if len(req.Models) == 0 {
		req.Models = []string{"*"} // All models
	}
	if len(req.Permissions) == 0 {
		req.Permissions = []string{"chat", "models"} // Basic permissions
	}

	// Create API key
	apiKey := &models.APIKey{
		ID:          keyID,
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		KeyPrefix:   models.ExtractKeyPrefix(plainKey),
		UserID:      nil,
		TenantID:    &tenantID,
		Scope:       models.APIKeyScopeTenant,
		Models:      req.Models,
		Permissions: req.Permissions,
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
	}

	if req.RateLimits != nil {
		apiKey.RateLimits = *req.RateLimits
	} else {
		// Default rate limits for tenant keys
		apiKey.RateLimits = models.RateLimits{
			RequestsPerMinute: 100,
			RequestsPerHour:   5000,
			RequestsPerDay:    50000,
		}
	}

	// Save to database
	if err := h.db.CreateAPIKey(c.Request.Context(), apiKey); err != nil {
		h.logger.WithError(err).Error("Failed to save API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create API key",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"key_id":    apiKey.ID,
		"user_id":   userID,
	}).Info("Tenant API key created")

	// Return API key with plaintext (only shown once!)
	apiKey.KeyHash = "" // Don't return hash
	c.JSON(http.StatusCreated, gin.H{
		"api_key": apiKey,
		"key":     plainKey, // Plaintext key shown only once
	})
}

// DeleteTenantAPIKey удаляет API ключ организации
// DELETE /api/tenants/:id/api-keys/:key_id
func (h *TenantHandler) DeleteTenantAPIKey(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	tenantID := c.Param("id")
	keyID := c.Param("key_id")

	// Check if user is owner or admin
	member, err := h.db.GetTenantMember(c.Request.Context(), tenantID, userID)
	if err != nil || member == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	if member.Role != models.TenantRoleOwner && member.Role != models.TenantRoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only owners and admins can delete API keys",
		})
		return
	}

	// Get the API key to verify it belongs to this tenant
	apiKey, err := h.db.GetAPIKey(c.Request.Context(), keyID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API key")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "API key not found",
		})
		return
	}

	// Verify key belongs to this tenant
	if apiKey.TenantID == nil || *apiKey.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "API key does not belong to this tenant",
		})
		return
	}

	// Delete API key
	if err := h.db.DeleteAPIKey(c.Request.Context(), keyID); err != nil {
		h.logger.WithError(err).Error("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete API key",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"tenant_id": tenantID,
		"key_id":    keyID,
		"user_id":   userID,
	}).Info("Tenant API key deleted")

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// Helper function
func generateSlug(name string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)

	// Add random suffix to ensure uniqueness
	slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])

	return slug
}
