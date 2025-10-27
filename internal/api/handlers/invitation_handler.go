// Package handlers provides HTTP handlers for invitation management (AUTH-03, v2.2.0)
package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// InvitationHandler обрабатывает invitation запросы (AUTH-03, v2.2.0)
type InvitationHandler struct {
	db     storage.Database
	config *config.Config
	logger *logrus.Logger
}

// NewInvitationHandler создает новый Invitation Handler
func NewInvitationHandler(db storage.Database, config *config.Config, logger *logrus.Logger) *InvitationHandler {
	return &InvitationHandler{
		db:     db,
		config: config,
		logger: logger,
	}
}

// CreateInvitation создает новое приглашение
// POST /api/admin/invitations
func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	var req models.CreateInvitationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Debug("Invalid create invitation request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get current user ID from context (set by JWT middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Apply default values from config
	if req.MaxUses == 0 {
		req.MaxUses = h.config.Auth.Invitations.MaxUsesDefault
		if req.MaxUses == 0 {
			req.MaxUses = 1 // Fallback default
		}
	}

	// Apply default expiry if not specified
	if req.ExpiresAt == nil && h.config.Auth.Invitations.DefaultExpiryDays > 0 {
		expiresAt := time.Now().AddDate(0, 0, h.config.Auth.Invitations.DefaultExpiryDays)
		req.ExpiresAt = &expiresAt
	}

	// Create invitation model
	invitation := &models.Invitation{
		ID:              uuid.New().String(),
		Token:           uuid.New().String(),
		CreatedByUserID: userID.(string),
		Email:           req.Email,
		ExpiresAt:       req.ExpiresAt,
		MaxUses:         req.MaxUses,
	}

	// Save to database
	if err := h.db.CreateInvitation(c.Request.Context(), invitation); err != nil {
		h.logger.WithError(err).Error("Failed to create invitation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create invitation",
		})
		return
	}

	// Generate full invitation link
	baseURL := h.getBaseURL(c)
	invitationLink := invitation.InvitationLink(baseURL)

	h.logger.WithFields(logrus.Fields{
		"invitation_id": invitation.ID,
		"created_by":    userID,
	}).Info("Invitation created")

	c.JSON(http.StatusCreated, gin.H{
		"invitation":      invitation,
		"invitation_link": invitationLink,
	})
}

// ListInvitations возвращает список приглашений
// GET /api/admin/invitations
func (h *InvitationHandler) ListInvitations(c *gin.Context) {
	var filter models.InvitationListFilter

	// Parse query parameters
	if status := c.Query("status"); status != "" {
		invStatus := models.InvitationStatus(status)
		filter.Status = &invStatus
	}

	if email := c.Query("email"); email != "" {
		filter.Email = &email
	}

	if createdBy := c.Query("created_by"); createdBy != "" {
		filter.CreatedByUserID = &createdBy
	}

	// Pagination
	filter.Limit = 50 // Default limit
	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := fmt.Sscanf(limit, "%d", &l); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	filter.Offset = 0
	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := fmt.Sscanf(offset, "%d", &o); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	// Get invitations from database
	invitations, err := h.db.ListInvitations(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list invitations")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve invitations",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invitations": invitations,
		"count":       len(invitations),
	})
}

// GetInvitationStats возвращает статистику по приглашениям
// GET /api/admin/invitations/stats
func (h *InvitationHandler) GetInvitationStats(c *gin.Context) {
	stats, err := h.db.GetInvitationStats(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to get invitation stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// RevokeInvitation отзывает приглашение
// DELETE /api/admin/invitations/:id
func (h *InvitationHandler) RevokeInvitation(c *gin.Context) {
	invitationID := c.Param("id")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invitation ID is required",
		})
		return
	}

	// Get current user ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get optional reason from request body
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if req.Reason == "" {
		req.Reason = "Revoked by administrator"
	}

	// Revoke invitation
	err := h.db.RevokeInvitation(c.Request.Context(), invitationID, userID.(string), req.Reason)
	if err != nil {
		h.logger.WithError(err).WithField("invitation_id", invitationID).Error("Failed to revoke invitation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to revoke invitation",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"invitation_id": invitationID,
		"revoked_by":    userID,
	}).Info("Invitation revoked")

	c.JSON(http.StatusOK, gin.H{
		"message": "Invitation revoked successfully",
	})
}

// GetInvitationDetails возвращает детальную информацию о приглашении с данными пользователей
// GET /api/admin/invitations/:id
func (h *InvitationHandler) GetInvitationDetails(c *gin.Context) {
	invitationID := c.Param("id")
	if invitationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invitation ID is required",
		})
		return
	}

	// Get invitation with user details
	invitation, err := h.db.GetInvitationWithUsers(c.Request.Context(), invitationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get invitation with users")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Invitation not found",
		})
		return
	}

	h.logger.WithField("invitation_id", invitationID).Debug("Invitation details retrieved")

	c.JSON(http.StatusOK, invitation)
}

// ValidateInvitation проверяет валидность токена приглашения
// GET /api/invitations/:token/validate
func (h *InvitationHandler) ValidateInvitation(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invitation token is required",
		})
		return
	}

	// Get invitation by token
	invitation, err := h.db.GetInvitationByToken(c.Request.Context(), token)
	if err != nil {
		h.logger.WithError(err).Debug("Invitation not found")
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Invitation not found",
			"valid":  false,
			"status": "not_found",
		})
		return
	}

	// Check if valid
	status := invitation.GetStatus()
	isValid := invitation.IsValid()

	// Optional: check email restriction
	email := c.Query("email")
	var canUseEmail bool
	if email != "" {
		canUseEmail = invitation.CanBeUsedByEmail(email)
	} else {
		canUseEmail = true // No email check if not provided
	}

	response := gin.H{
		"valid":  isValid && canUseEmail,
		"status": status,
		"invitation": gin.H{
			"id":          invitation.ID,
			"email":       invitation.Email,
			"expires_at":  invitation.ExpiresAt,
			"max_uses":    invitation.MaxUses,
			"current_uses": invitation.CurrentUses,
		},
	}

	if !canUseEmail {
		response["error"] = "This invitation is restricted to a specific email address"
	}

	if !isValid {
		response["error"] = "Invitation is not valid: " + string(status)
	}

	statusCode := http.StatusOK
	if !isValid || !canUseEmail {
		statusCode = http.StatusForbidden
	}

	c.JSON(statusCode, response)
}

// Helper: get base URL from request
func (h *InvitationHandler) getBaseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	// Check X-Forwarded-Proto header
	if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := c.Request.Host
	return scheme + "://" + host
}


