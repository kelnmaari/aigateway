// Package ui - Settings UI Handler
// Version: v3.0.9 - Phase 3
package ui

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/settings"
)

// SettingsHandler handles settings UI requests
type SettingsHandler struct {
	manager *settings.Manager
	logger  *logrus.Logger
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(manager *settings.Manager, logger *logrus.Logger) *SettingsHandler {
	return &SettingsHandler{
		manager: manager,
		logger:  logger,
	}
}

// GetSettings returns all settings grouped by category (admin panel)
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all settings grouped by category
	grouped, err := h.manager.GetAllSettings(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get settings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve settings",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": grouped,
	})
}

// UpdateSettingRequest represents request to update a setting
type UpdateSettingRequest struct {
	Value string `json:"value" binding:"required"`
}

// UpdateSetting updates a setting value (Phase 3)
func (h *SettingsHandler) UpdateSetting(c *gin.Context) {
	ctx := c.Request.Context()
	settingID := c.Param("id")

	if settingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Setting ID is required",
		})
		return
	}

	var req UpdateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Get current setting to validate editability
	setting, err := h.manager.GetSetting(ctx, settingID)
	if err != nil {
		h.logger.WithError(err).WithField("id", settingID).Error("Failed to get setting")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Setting not found",
		})
		return
	}

	// Check if setting is editable
	if !setting.IsEditable {
		c.JSON(http.StatusForbidden, gin.H{
			"error": fmt.Sprintf("Setting '%s' is not editable via UI (requires server restart)", settingID),
		})
		return
	}

	// Validate new value
	if err := h.validateSettingValue(setting, req.Value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Validation failed: %v", err),
		})
		return
	}

	// Get user ID from JWT claims (for audit log)
	userID := "admin" // Default
	if claims, exists := c.Get("user_id"); exists {
		if uid, ok := claims.(string); ok {
			userID = uid
		}
	}

	// Update setting
	if err := h.manager.SetString(ctx, settingID, req.Value, userID); err != nil {
		h.logger.WithError(err).
			WithField("id", settingID).
			WithField("user", userID).
			Error("Failed to update setting")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update setting",
		})
		return
	}

	h.logger.WithFields(map[string]interface{}{
		"id":        settingID,
		"old_value": setting.Value,
		"new_value": req.Value,
		"user":      userID,
	}).Info("Setting updated via UI")

	// Return updated setting
	updatedSetting, _ := h.manager.GetSetting(ctx, settingID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Setting updated successfully",
		"setting": updatedSetting,
	})
}

// validateSettingValue validates the new value based on type and validation rules
func (h *SettingsHandler) validateSettingValue(setting *settings.Setting, newValue string) error {
	// Type validation
	switch setting.Type {
	case settings.TypeBool:
		if newValue != "true" && newValue != "false" {
			return fmt.Errorf("boolean value must be 'true' or 'false'")
		}
	case settings.TypeInt:
		if _, err := strconv.Atoi(newValue); err != nil {
			return fmt.Errorf("invalid integer value: %v", err)
		}
	case settings.TypeFloat:
		if _, err := strconv.ParseFloat(newValue, 64); err != nil {
			return fmt.Errorf("invalid float value: %v", err)
		}
	case settings.TypeDuration:
		if _, err := time.ParseDuration(newValue); err != nil {
			return fmt.Errorf("invalid duration format (use '30s', '5m', '1h'): %v", err)
		}
	}

	// Custom validation rules (regex)
	if setting.ValidationRule != "" {
		matched, err := regexp.MatchString(setting.ValidationRule, newValue)
		if err != nil {
			return fmt.Errorf("invalid validation rule: %v", err)
		}
		if !matched {
			return fmt.Errorf("value does not match validation pattern: %s", setting.ValidationRule)
		}
	}

	return nil
}

// GetSettingsByCategory returns settings for a specific category
func (h *SettingsHandler) GetSettingsByCategory(c *gin.Context) {
	ctx := c.Request.Context()
	category := settings.SettingCategory(c.Param("category"))

	settingsList, err := h.manager.GetAllSettings(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get settings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve settings",
		})
		return
	}

	categorySettings, ok := settingsList[category]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Category not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"settings": categorySettings,
	})
}
