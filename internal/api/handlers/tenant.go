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

	"ollama-openai-proxy/internal/auth/middleware"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// TenantHandler обрабатывает tenant management запросы
type TenantHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewTenantHandler создает новый Tenant Handler
func NewTenantHandler(db storage.Database, logger *logrus.Logger) *TenantHandler {
	return &TenantHandler{
		db:     db,
		logger: logger,
	}
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
		UserID string            `json:"user_id" binding:"required"`
		Role   models.TenantRole `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id and role are required",
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

	// Check if user exists
	_, err = h.db.GetUser(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Check if already a member
	existingMember, _ := h.db.GetTenantMember(c.Request.Context(), tenantID, req.UserID)
	if existingMember != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User is already a member of this tenant",
		})
		return
	}

	// Add member
	member := &models.TenantMember{
		TenantID:  tenantID,
		UserID:    req.UserID,
		Role:      req.Role,
		InvitedBy: userID,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
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
		"user_id":   req.UserID,
		"role":      req.Role,
	}).Info("Member added to tenant")

	c.JSON(http.StatusCreated, member)
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
