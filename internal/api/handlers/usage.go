// Package handlers provides HTTP handlers for usage statistics
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/storage"
)

// UsageHandler handles usage statistics endpoints
type UsageHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewUsageHandler creates a new usage handler
func NewUsageHandler(db storage.Database, logger *logrus.Logger) *UsageHandler {
	return &UsageHandler{
		db:     db,
		logger: logger,
	}
}

// GetUserUsage returns usage statistics for the current user
// GET /api/usage/personal
func (h *UsageHandler) GetUserUsage(c *gin.Context) {
	// Get user ID from JWT auth middleware context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}
	userIDStr := userID.(string)

	// Parse time period (default: 7 days)
	period := 7 * 24 * time.Hour
	if periodParam := c.Query("period"); periodParam != "" {
		switch periodParam {
		case "24h":
			period = 24 * time.Hour
		case "7days":
			period = 7 * 24 * time.Hour
		case "30days":
			period = 30 * 24 * time.Hour
		case "90days":
			period = 90 * 24 * time.Hour
		}
	}

	stats, err := h.db.GetUserUsageStats(c.Request.Context(), userIDStr, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user usage stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetTenantUsage returns usage statistics for a tenant
// GET /api/usage/tenant/:tenant_id
func (h *UsageHandler) GetTenantUsage(c *gin.Context) {
	// Get user ID from JWT auth middleware context (for access control)
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "tenant_id is required",
		})
		return
	}

	// Verify user has access to this tenant
	// (This should check tenant membership, but for simplicity we'll allow it for now)
	// TODO: Add proper tenant membership check

	// Parse time period (default: 7 days)
	period := 7 * 24 * time.Hour
	if periodParam := c.Query("period"); periodParam != "" {
		switch periodParam {
		case "24h":
			period = 24 * time.Hour
		case "7days":
			period = 7 * 24 * time.Hour
		case "30days":
			period = 30 * 24 * time.Hour
		case "90days":
			period = 90 * 24 * time.Hour
		}
	}

	stats, err := h.db.GetTenantUsageStats(c.Request.Context(), tenantID, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get tenant usage stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve tenant usage statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
