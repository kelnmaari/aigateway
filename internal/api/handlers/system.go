// Package handlers provides HTTP request handlers
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/service"
	"aigateway/internal/config"
)

// SystemHandler handles system-level operations (initialization, health, etc)
type SystemHandler struct {
	config           *config.Config
	logger           *logrus.Logger
	bootstrapService *service.BootstrapService
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(cfg *config.Config, logger *logrus.Logger, bootstrapService *service.BootstrapService) *SystemHandler {
	return &SystemHandler{
		config:           cfg,
		logger:           logger,
		bootstrapService: bootstrapService,
	}
}

// GetInitStatus returns the system initialization status
// GET /api/system/init-status
func (h *SystemHandler) GetInitStatus(c *gin.Context) {
	initialized, err := h.bootstrapService.IsInitialized(c.Request.Context())
	if err != nil {
		h.logger.WithError(err).Error("Failed to check initialization status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check system status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"initialized":        initialized,
		"requires_bootstrap": !initialized,
	})
}

// Bootstrap performs first-time system setup
// POST /api/system/bootstrap
func (h *SystemHandler) Bootstrap(c *gin.Context) {
	var req service.BootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.bootstrapService.Bootstrap(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Bootstrap failed")

		// Provide specific error messages
		switch err.Error() {
		case "system is already initialized":
			c.JSON(http.StatusConflict, gin.H{"error": "System is already initialized"})
		case "invalid bootstrap token":
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid bootstrap token"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

