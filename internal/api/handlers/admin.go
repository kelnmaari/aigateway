// Package handlers provides admin API handlers
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/apikey"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// AdminHandler обрабатывает административные эндпоинты
type AdminHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	keyManager   *apikey.Manager  // Legacy JSON storage (deprecated)
	db           storage.Database // Database for API keys (Version 1.3.0+)
	ollamaClient OllamaClientInterface
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
// Но с доступом к БД для API ключей (Version 1.3.0+)
func NewAdminHandlerWithoutKeys(cfg *config.Config, logger *logrus.Logger, db storage.Database) *AdminHandler {
	return &AdminHandler{
		config:     cfg,
		logger:     logger,
		keyManager: nil, // Будет возвращать "not implemented" ошибки (legacy)
		db:         db,  // Database для API ключей (Version 1.3.0+)
	}
}

// SetOllamaClient устанавливает Ollama client для AdminHandler
func (h *AdminHandler) SetOllamaClient(client OllamaClientInterface) {
	h.ollamaClient = client
}

// ListAPIKeys обрабатывает GET /admin/api-keys
func (h *AdminHandler) ListAPIKeys(c *gin.Context) {
	h.logger.Info("Admin API Keys list requested")

	// Use database if available (Version 1.3.0+), fallback to JSON storage
	if h.db != nil {
		h.logger.Info("Using DATABASE for API keys listing (Version 1.3.0+)")
		h.listAPIKeysFromDB(c)
		return
	}

	// Legacy: Use JSON storage
	h.logger.Warn("Database not available, falling back to JSON storage (legacy)")
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
	h.logger.Info("🔍 listAPIKeysFromDB called - reading from DATABASE")

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

	h.logger.WithField("db_keys_count", len(keys)).Info("📊 Keys retrieved from DATABASE")

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

	// Enrich keys with owner and tenant information
	for i := range pagedKeys {
		// Get owner username if available
		if pagedKeys[i].UserID != nil && *pagedKeys[i].UserID != "" {
			if user, err := h.db.GetUser(ctx, *pagedKeys[i].UserID); err == nil {
				pagedKeys[i].Metadata = map[string]interface{}{
					"owner_username": user.Username,
				}
			}
		}

		// Get tenant name if available
		if pagedKeys[i].TenantID != nil && *pagedKeys[i].TenantID != "" {
			if tenant, err := h.db.GetTenant(ctx, *pagedKeys[i].TenantID); err == nil {
				if pagedKeys[i].Metadata == nil {
					pagedKeys[i].Metadata = make(map[string]interface{})
				}
				pagedKeys[i].Metadata["tenant_name"] = tenant.Name
			}
		}
	}

	// Convert to public format
	publicKeys := make([]models.APIKeyPublic, len(pagedKeys))
	for i, key := range pagedKeys {
		ownerUsername := ""
		tenantName := ""

		// Extract enriched data from metadata
		if key.Metadata != nil {
			if username, ok := key.Metadata["owner_username"].(string); ok {
				ownerUsername = username
			}
			if tname, ok := key.Metadata["tenant_name"].(string); ok {
				tenantName = tname
			}
		}

		publicKeys[i] = models.APIKeyPublic{
			ID:            key.ID,
			Name:          key.Name,
			Description:   key.Description,
			UserID:        key.UserID,
			TenantID:      key.TenantID,
			Scope:         key.Scope,
			Models:        key.Models,
			Permissions:   key.Permissions,
			RateLimits:    key.RateLimits,
			Status:        key.Status,
			CreatedAt:     key.CreatedAt,
			OwnerUsername: ownerUsername,
			TenantName:    tenantName,
			UpdatedAt:     key.UpdatedAt,
			ExpiresAt:     key.ExpiresAt,
			LastUsedAt:    key.LastUsedAt,
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

// GetModelDetails обрабатывает GET /api/admin/models/:name/details
// Возвращает расширенную информацию о модели из Ollama
func (h *AdminHandler) GetModelDetails(c *gin.Context) {
	modelName := c.Param("name")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Model name is required",
				"type":    "invalid_request_error",
				"code":    "model_name_required",
			},
		})
		return
	}

	if h.ollamaClient == nil {
		h.logger.Error("Ollama client not available")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"message": "Ollama client not configured",
				"type":    "service_unavailable",
				"code":    "ollama_unavailable",
			},
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Получаем детали модели из Ollama
	ollamaResp, err := h.ollamaClient.ShowModel(ctx, modelName)
	if err != nil {
		h.logger.WithError(err).WithField("model", modelName).Error("Failed to get model details from Ollama")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to retrieve model details",
				"type":    "api_error",
				"code":    "ollama_error",
				"details": err.Error(),
			},
		})
		return
	}

	// Преобразуем Ollama response в наш формат
	response := models.ModelDetailsResponse{
		Name:       modelName,
		License:    ollamaResp.License,
		Template:   ollamaResp.Template,
		Modelfile:  ollamaResp.Modelfile,
		Parameters: parseModelParameters(ollamaResp.Parameters),
	}

	// Добавляем детали если есть
	if ollamaResp.Details != nil {
		response.Format = ollamaResp.Details.Format
		response.Family = ollamaResp.Details.Family
		response.ParameterSize = ollamaResp.Details.ParameterSize
		response.Quantization = ollamaResp.Details.QuantizationLevel
	}

	// Добавляем информацию о модели если есть
	if ollamaResp.ModelInfo != nil {
		info := ollamaResp.ModelInfo
		response.Architecture = getStringValue(info, "general.architecture")
		response.ContextLength = getIntValue(info, "llama.context_length")
		response.EmbeddingSize = getIntValue(info, "llama.embedding_length")
		response.Layers = getIntValue(info, "llama.block_count")
		response.Heads = getIntValue(info, "llama.attention.head_count")
		response.VocabSize = getIntValue(info, "tokenizer.ggml.vocab_size")
	}

	h.logger.WithField("model", modelName).Info("Model details retrieved successfully")
	c.JSON(http.StatusOK, response)
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
	var storageErr *storage.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Type == storage.StorageErrorTypeNotFound
	}
	return false
}

// isAlreadyExistsError проверяет является ли ошибка "уже существует"
func isAlreadyExistsError(err error) bool {
	var storageErr *storage.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Type == storage.StorageErrorTypeAlreadyExists
	}
	return false
}

// isInvalidDataError проверяет является ли ошибка "неверные данные"
func isInvalidDataError(err error) bool {
	var storageErr *storage.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Type == storage.StorageErrorTypeInvalidData
	}
	return false
}

// isPermissionError проверяет является ли ошибка "доступ запрещен"
func isPermissionError(err error) bool {
	var storageErr *storage.StorageError
	if errors.As(err, &storageErr) {
		return storageErr.Type == storage.StorageErrorTypePermission
	}
	return false
}

// parseModelParameters парсит строку параметров модели в map
func parseModelParameters(paramsStr string) map[string]interface{} {
	params := make(map[string]interface{})
	if paramsStr == "" {
		return params
	}

	// Простой парсинг строки параметров (формат: key value)
	lines := splitLines(paramsStr)
	for _, line := range lines {
		line = trimSpace(line)
		if line == "" {
			continue
		}

		// Разбиваем на key и value по первому пробелу
		parts := splitFirst(line, " ")
		if len(parts) == 2 {
			key := trimSpace(parts[0])
			value := trimSpace(parts[1])
			params[key] = value
		}
	}

	return params
}

// getStringValue извлекает строковое значение из map
func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// getIntValue извлекает числовое значение из map
func getIntValue(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// splitLines разбивает строку на линии
func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			result = append(result, current)
			current = ""
		} else if ch != '\r' {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// splitFirst разбивает строку на две части по первому вхождению разделителя
func splitFirst(s, sep string) []string {
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

// trimSpace удаляет пробелы с начала и конца строки
func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

