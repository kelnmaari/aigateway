// Package handlers provides quota management API handlers
// Version: 1.11.7+ (Enterprise Suite - Usage Quotas System)
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/quota"
	"aigateway/internal/storage"
)

// QuotaHandler handles quota management requests
type QuotaHandler struct {
	db           storage.Database
	quotaService *quota.Service
	logger       *logrus.Logger
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler(db storage.Database, quotaService *quota.Service, logger *logrus.Logger) *QuotaHandler {
	return &QuotaHandler{
		db:           db,
		quotaService: quotaService,
		logger:       logger,
	}
}

// CreateQuota создает новую квоту
func (h *QuotaHandler) CreateQuota(c *gin.Context) {
	var req models.CreateQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quota := &models.Quota{
		ID:                uuid.New().String(),
		Name:              req.Name,
		Scope:             req.Scope,
		TargetID:          req.TargetID,
		TokensPerDay:      req.TokensPerDay,
		TokensPerMonth:    req.TokensPerMonth,
		RequestsPerDay:    req.RequestsPerDay,
		RequestsPerMonth:  req.RequestsPerMonth,
		MaxConcurrent:     req.MaxConcurrent,
		MaxStorageBytes:   req.MaxStorageBytes,
		MaxConversations:  req.MaxConversations,
		MaxFileSize:       req.MaxFileSize,
		AllowedModels:     req.AllowedModels,
		Enabled:           true,
	}

	if err := h.db.CreateQuota(c.Request.Context(), quota); err != nil {
		h.logger.WithError(err).Error("Failed to create quota")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create quota"})
		return
	}

	c.JSON(http.StatusCreated, quota)
}

// ListQuotas возвращает список квот
func (h *QuotaHandler) ListQuotas(c *gin.Context) {
	scopeStr := c.Query("scope")
	
	var scope *models.QuotaScope
	if scopeStr != "" {
		s := models.QuotaScope(scopeStr)
		scope = &s
	}

	quotas, err := h.db.ListQuotas(c.Request.Context(), scope)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list quotas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list quotas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"quotas": quotas})
}

// GetQuota возвращает квоту по ID
func (h *QuotaHandler) GetQuota(c *gin.Context) {
	id := c.Param("id")

	quota, err := h.db.GetQuota(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quota not found"})
		return
	}

	c.JSON(http.StatusOK, quota)
}

// UpdateQuota обновляет квоту
func (h *QuotaHandler) UpdateQuota(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateQuotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quota, err := h.db.GetQuota(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quota not found"})
		return
	}

	// Update fields
	if req.Name != nil {
		quota.Name = *req.Name
	}
	if req.TokensPerDay != nil {
		quota.TokensPerDay = req.TokensPerDay
	}
	if req.TokensPerMonth != nil {
		quota.TokensPerMonth = req.TokensPerMonth
	}
	if req.RequestsPerDay != nil {
		quota.RequestsPerDay = req.RequestsPerDay
	}
	if req.RequestsPerMonth != nil {
		quota.RequestsPerMonth = req.RequestsPerMonth
	}
	if req.MaxConcurrent != nil {
		quota.MaxConcurrent = req.MaxConcurrent
	}
	if req.MaxStorageBytes != nil {
		quota.MaxStorageBytes = req.MaxStorageBytes
	}
	if req.MaxConversations != nil {
		quota.MaxConversations = req.MaxConversations
	}
	if req.MaxFileSize != nil {
		quota.MaxFileSize = req.MaxFileSize
	}
	if req.AllowedModels != nil {
		quota.AllowedModels = *req.AllowedModels
	}
	if req.Enabled != nil {
		quota.Enabled = *req.Enabled
	}

	if err := h.db.UpdateQuota(c.Request.Context(), quota); err != nil {
		h.logger.WithError(err).Error("Failed to update quota")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update quota"})
		return
	}

	c.JSON(http.StatusOK, quota)
}

// DeleteQuota удаляет квоту
func (h *QuotaHandler) DeleteQuota(c *gin.Context) {
	id := c.Param("id")

	if err := h.db.DeleteQuota(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to delete quota")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete quota"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "quota deleted"})
}

// GetQuotaUsage возвращает usage для квоты
func (h *QuotaHandler) GetQuotaUsage(c *gin.Context) {
	id := c.Param("id")

	usage, err := h.db.GetQuotaUsage(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quota usage not found"})
		return
	}

	c.JSON(http.StatusOK, usage)
}

// GetMyQuota возвращает квоту текущего пользователя
func (h *QuotaHandler) GetMyQuota(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var tenantID *string
	if tid, exists := c.Get("tenant_id"); exists {
		if tidStr, ok := tid.(string); ok && tidStr != "" {
			tenantID = &tidStr
		}
	}

	stats, err := h.quotaService.GetQuotaStats(c.Request.Context(), userID.(string), tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no quota assigned"})
		return
	}

	c.JSON(http.StatusOK, stats)
}


