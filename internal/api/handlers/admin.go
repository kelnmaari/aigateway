// Package handlers provides admin API handlers
package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/apikey"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// AdminHandler обрабатывает административные эндпоинты
type AdminHandler struct {
	config     *config.Config
	logger     *logrus.Logger
	keyManager *apikey.Manager  // Legacy JSON storage (deprecated)
	db         storage.Database // Database for API keys (Version 1.3.0+)
}

// NewAdminHandler создает новый admin handler
func NewAdminHandler(cfg *config.Config, logger *logrus.Logger, keyManager *apikey.Manager, db storage.Database) *AdminHandler {
	return &AdminHandler{
		config:     cfg,
		logger:     logger,
		keyManager: keyManager,
		db:         db,
	}
}

// NewAdminHandlerWithoutKeys создает admin handler без key manager (для обратной совместимости)
func NewAdminHandlerWithoutKeys(cfg *config.Config, logger *logrus.Logger) *AdminHandler {
	return &AdminHandler{
		config:     cfg,
		logger:     logger,
		keyManager: nil, // Будет возвращать "not implemented" ошибки
	}
}

// ListAPIKeys обрабатывает GET /admin/api-keys
func (h *AdminHandler) ListAPIKeys(c *gin.Context) {
	h.logger.Info("Admin API Keys list requested")

	// Use database if available (Version 1.3.0+), fallback to JSON storage
	if h.db != nil {
		h.listAPIKeysFromDB(c)
		return
	}

	// Legacy: Use JSON storage
	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	// Парсим query параметры для фильтрации
	req := models.ListAPIKeysRequest{
		Limit:     parseIntQuery(c, "limit", 50),
		Offset:    parseIntQuery(c, "offset", 0),
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	// Фильтр по статусу
	if statusStr := c.Query("status"); statusStr != "" {
		status := models.APIKeyStatus(statusStr)
		req.Status = &status
	}

	// Фильтр по модели
	if model := c.Query("model"); model != "" {
		req.Model = &model
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Получаем список ключей
	response, err := h.keyManager.ListAPIKeys(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list API keys")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve API keys",
				"type":    "api_error",
				"code":    "list_keys_failed",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"total_keys":    response.Total,
		"returned_keys": len(response.APIKeys),
		"limit":         req.Limit,
		"offset":        req.Offset,
	}).Debug("API keys list retrieved")

	c.JSON(http.StatusOK, response)
}

// listAPIKeysFromDB reads API keys from database (Version 1.3.0+)
func (h *AdminHandler) listAPIKeysFromDB(c *gin.Context) {
	// Parse query parameters
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get all API keys from database
	keys, err := h.db.ListAPIKeys(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list API keys from database")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve API keys",
				"type":    "api_error",
				"code":    "list_keys_failed",
			},
		})
		return
	}

	// Apply pagination
	total := len(keys)
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedKeys := keys[start:end]

	// Convert to public format
	publicKeys := make([]models.APIKeyPublic, len(pagedKeys))
	for i, key := range pagedKeys {
		publicKeys[i] = models.APIKeyPublic{
			ID:          key.ID,
			Name:        key.Name,
			Description: key.Description,
			Models:      key.Models,
			Permissions: key.Permissions,
			RateLimits:  key.RateLimits,
			Status:      key.Status,
			CreatedAt:   key.CreatedAt,
			UpdatedAt:   key.UpdatedAt,
			ExpiresAt:   key.ExpiresAt,
			LastUsedAt:  key.LastUsedAt,
		}
	}

	h.logger.WithFields(logrus.Fields{
		"total_keys":    total,
		"returned_keys": len(publicKeys),
		"limit":         limit,
		"offset":        offset,
	}).Debug("API keys list retrieved from database")

	c.JSON(http.StatusOK, gin.H{
		"api_keys": publicKeys,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// CreateAPIKey обрабатывает POST /admin/api-keys
func (h *AdminHandler) CreateAPIKey(c *gin.Context) {
	h.logger.Info("Admin create API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid create API key request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format: " + err.Error(),
				"type":    "invalid_request_error",
				"code":    "invalid_request",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Создаем API ключ
	response, err := h.keyManager.CreateAPIKey(ctx, req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create API key: " + err.Error(),
				"type":    "api_error",
				"code":    "key_creation_failed",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"key_id":   response.APIKey.ID,
		"key_name": response.APIKey.Name,
	}).Info("API key created successfully")

	c.JSON(http.StatusCreated, response)
}

// GetAPIKey обрабатывает GET /admin/api-keys/:id
func (h *AdminHandler) GetAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	h.logger.WithField("key_id", keyID).Info("Admin get API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Получаем API ключ
	apiKey, err := h.keyManager.GetAPIKey(ctx, keyID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to get API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve API key",
				"type":    "api_error",
				"code":    "key_retrieval_failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": apiKey,
	})
}

// UpdateAPIKey обрабатывает PUT /admin/api-keys/:id
func (h *AdminHandler) UpdateAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	h.logger.WithField("key_id", keyID).Info("Admin update API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	var req models.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid update API key request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format: " + err.Error(),
				"type":    "invalid_request_error",
				"code":    "invalid_request",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Обновляем API ключ
	updatedKey, err := h.keyManager.UpdateAPIKey(ctx, keyID, req)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to update API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to update API key: " + err.Error(),
				"type":    "api_error",
				"code":    "key_update_failed",
			},
		})
		return
	}

	h.logger.WithField("key_id", keyID).Info("API key updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"api_key": updatedKey,
	})
}

// DeleteAPIKey обрабатывает DELETE /admin/api-keys/:id
func (h *AdminHandler) DeleteAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	h.logger.WithField("key_id", keyID).Info("Admin delete API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Удаляем API ключ
	if err := h.keyManager.DeleteAPIKey(ctx, keyID); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to delete API key: " + err.Error(),
				"type":    "api_error",
				"code":    "key_deletion_failed",
			},
		})
		return
	}

	h.logger.WithField("key_id", keyID).Info("API key deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// GetAPIKeyUsage обрабатывает GET /admin/api-keys/:id/usage
func (h *AdminHandler) GetAPIKeyUsage(c *gin.Context) {
	keyID := c.Param("id")
	days := parseIntQuery(c, "days", 7)

	h.logger.WithFields(logrus.Fields{
		"key_id": keyID,
		"days":   days,
	}).Info("Admin get API Key usage requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Получаем отчет об использовании
	report, err := h.keyManager.GetUsageReport(ctx, keyID, days)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to get API key usage")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve usage report",
				"type":    "api_error",
				"code":    "usage_report_failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"usage_report": report,
	})
}

// RevokeAPIKey обрабатывает PATCH /admin/api-keys/:id/revoke (AUTH-04)
func (h *AdminHandler) RevokeAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	// Получаем причину отзыва из body
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Reason опционален, просто используем пустую строку
		req.Reason = "Revoked by administrator"
	}

	h.logger.WithFields(logrus.Fields{
		"key_id": keyID,
		"reason": req.Reason,
	}).Info("Admin revoke API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Отзываем ключ
	if err := h.keyManager.RevokeAPIKey(ctx, keyID, req.Reason); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to revoke API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to revoke API key",
				"type":    "api_error",
				"code":    "revoke_failed",
			},
		})
		return
	}

	// Получаем обновленный ключ
	updatedKey, err := h.keyManager.GetAPIKey(ctx, keyID)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get updated API key after revoke")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key revoked successfully",
		"api_key": updatedKey,
	})
}

// EnableAPIKey обрабатывает PATCH /admin/api-keys/:id/enable (AUTH-04)
func (h *AdminHandler) EnableAPIKey(c *gin.Context) {
	keyID := c.Param("id")

	h.logger.WithField("key_id", keyID).Info("Admin enable API Key requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Активируем ключ
	if err := h.keyManager.EnableAPIKey(ctx, keyID); err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to enable API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to enable API key",
				"type":    "api_error",
				"code":    "enable_failed",
			},
		})
		return
	}

	// Получаем обновленный ключ
	updatedKey, err := h.keyManager.GetAPIKey(ctx, keyID)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get updated API key after enable")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key enabled successfully",
		"api_key": updatedKey,
	})
}

// ExtendAPIKeyExpiration обрабатывает POST /admin/api-keys/:id/extend (AUTH-04)
func (h *AdminHandler) ExtendAPIKeyExpiration(c *gin.Context) {
	keyID := c.Param("id")

	// Получаем количество дней из body
	var req struct {
		Days int `json:"days" binding:"required,min=1,max=36500"` // Max 100 years
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request: days must be between 1 and 36500",
				"type":    "invalid_request_error",
				"code":    "invalid_days",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"key_id": keyID,
		"days":   req.Days,
	}).Info("Admin extend API Key expiration requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Продлеваем срок действия
	updatedKey, err := h.keyManager.ExtendExpiration(ctx, keyID, req.Days)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to extend API key expiration")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to extend API key expiration",
				"type":    "api_error",
				"code":    "extend_failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "API key expiration extended successfully",
		"api_key":       updatedKey,
		"extended_days": req.Days,
	})
}

// UpdateAPIKeyPermissions обрабатывает PATCH /admin/api-keys/:id/permissions (AUTH-04)
func (h *AdminHandler) UpdateAPIKeyPermissions(c *gin.Context) {
	keyID := c.Param("id")

	// Получаем обновления из body
	var req struct {
		Models      *[]string `json:"models,omitempty"`
		Permissions *[]string `json:"permissions,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"code":    "invalid_body",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"key_id":      keyID,
		"models":      req.Models,
		"permissions": req.Permissions,
	}).Info("Admin update API Key permissions requested")

	if h.keyManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "API Key management not available",
				"type":    "service_unavailable",
				"code":    "key_manager_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Обновляем permissions
	var models []string
	var permissions []string
	if req.Models != nil {
		models = *req.Models
	}
	if req.Permissions != nil {
		permissions = *req.Permissions
	}

	updatedKey, err := h.keyManager.UpdateAPIKeyPermissions(ctx, keyID, models, permissions)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"message": "API key not found",
					"type":    "not_found_error",
					"code":    "key_not_found",
				},
			})
			return
		}

		h.logger.WithError(err).Error("Failed to update API key permissions")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to update API key permissions",
				"type":    "api_error",
				"code":    "update_failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key permissions updated successfully",
		"api_key": updatedKey,
	})
}

// Helper functions

// parseIntQuery парсит integer query параметр с default значением
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if valueStr := c.Query(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil && value >= 0 {
			return value
		}
	}
	return defaultValue
}

// isNotFoundError проверяет является ли ошибка "не найдено"
func isNotFoundError(err error) bool {
	// TODO: Реализовать проверку типа ошибки
	return false
}
