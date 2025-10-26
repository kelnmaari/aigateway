// Package handlers provides HTTP handlers for user management endpoints
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/middleware"
	"ollama-openai-proxy/internal/auth/password"
	"ollama-openai-proxy/internal/models"
	auditService "ollama-openai-proxy/internal/services/audit"
	"ollama-openai-proxy/internal/storage"
)

// UserHandler обрабатывает user management запросы
type UserHandler struct {
	db          storage.Database
	logger      *logrus.Logger
	auditLogger *auditService.AuditLogger
}

// NewUserHandler создает новый User Handler
func NewUserHandler(db storage.Database, logger *logrus.Logger, auditLogger *auditService.AuditLogger) *UserHandler {
	return &UserHandler{
		db:          db,
		logger:      logger,
		auditLogger: auditLogger,
	}
}

// GetProfile возвращает профиль текущего пользователя
// GET /api/users/me
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	// Get user from database
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user profile",
		})
		return
	}

	// Remove sensitive data
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// UpdateProfile обновляет профиль текущего пользователя
// PUT /api/users/me
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	var req struct {
		FullName    *string                 `json:"full_name,omitempty"`
		Email       *string                 `json:"email,omitempty"`
		Preferences *models.UserPreferences `json:"preferences,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Get current user
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user",
		})
		return
	}

	// Update fields
	updated := false

	if req.FullName != nil && *req.FullName != user.FullName {
		user.FullName = *req.FullName
		updated = true
	}

	if req.Email != nil && *req.Email != user.Email {
		// Validate email
		if err := password.ValidateEmail(*req.Email); err != nil {
			if valErr, ok := err.(password.ValidationError); ok {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": valErr.Message,
					"field": valErr.Field,
				})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid email",
			})
			return
		}

		// Check if email already exists
		existingUser, err := h.db.GetUserByEmail(c.Request.Context(), *req.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email already in use",
				"field": "email",
			})
			return
		}

		user.Email = *req.Email
		user.Verified = false // Need to re-verify new email
		updated = true
	}

	if req.Preferences != nil {
		user.Preferences = *req.Preferences
		updated = true
	}

	if !updated {
		c.JSON(http.StatusOK, gin.H{
			"message": "No changes made",
		})
		return
	}

	// Save changes
	if err := h.db.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update profile",
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("User profile updated")

	// Remove sensitive data
	user.PasswordHash = ""

	c.JSON(http.StatusOK, user)
}

// DeleteAccount удаляет аккаунт текущего пользователя
// DELETE /api/users/me
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	var req struct {
		Password string `json:"password" binding:"required"`
		Confirm  bool   `json:"confirm" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Password and confirmation required",
		})
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Account deletion must be confirmed",
		})
		return
	}

	// Get user
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user",
		})
		return
	}

	// Verify password
	if err := password.Verify(user.PasswordHash, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Incorrect password",
		})
		return
	}

	// Delete user (cascade will delete related data)
	if err := h.db.DeleteUser(c.Request.Context(), userID); err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete account",
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("User account deleted")

	c.JSON(http.StatusOK, gin.H{
		"message": "Account deleted successfully",
	})
}

// ListTenants возвращает список tenants пользователя
// GET /api/users/me/tenants
func (h *UserHandler) ListTenants(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	// Get user's tenants
	tenants, err := h.db.ListUserTenants(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user tenants")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get tenants",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenants": tenants,
		"count":   len(tenants),
	})
}

// ListPersonalAPIKeys возвращает личные API ключи пользователя
// GET /api/users/me/api-keys
func (h *UserHandler) ListPersonalAPIKeys(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	// Get user's personal API keys
	keys, err := h.db.ListPersonalAPIKeys(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get personal API keys")
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

// CreatePersonalAPIKey создает личный API ключ
// POST /api/users/me/api-keys
func (h *UserHandler) CreatePersonalAPIKey(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
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
		ID:          keyID, // Use the same ID that's embedded in the key
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		UserID:      &userID,
		TenantID:    nil,
		Scope:       models.APIKeyScopePersonal,
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
		// Default rate limits for personal keys
		apiKey.RateLimits = models.RateLimits{
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    10000,
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
		"user_id": userID,
		"key_id":  apiKey.ID,
	}).Info("Personal API key created")

	// Audit log: API Key created
	if h.auditLogger != nil {
		_ = h.auditLogger.LogAPIKeyCreated(c.Request.Context(), userID, apiKey.ID, apiKey.Name, c.ClientIP())
	}

	// Return API key with plaintext (only shown once!)
	apiKey.KeyHash = "" // Don't return hash
	c.JSON(http.StatusCreated, gin.H{
		"api_key": apiKey,
		"key":     plainKey, // ⚠️ Plaintext key - only shown once!
		"warning": "Save this key securely. It will not be shown again.",
	})
}

// DeletePersonalAPIKey удаляет персональный API ключ пользователя
// DELETE /api/users/me/api-keys/:id
func (h *UserHandler) DeletePersonalAPIKey(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "API key ID is required",
		})
		return
	}

	// Получаем ключ из БД для проверки владельца
	apiKey, err := h.db.GetAPIKey(c.Request.Context(), keyID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API key")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "API key not found",
		})
		return
	}

	// Проверяем что ключ принадлежит пользователю и это персональный ключ
	if apiKey.UserID == nil || *apiKey.UserID != userID || apiKey.TenantID != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"key_id":  keyID,
		}).Warn("Attempt to delete API key that doesn't belong to user")
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You don't have permission to delete this API key",
		})
		return
	}

	// Удаляем ключ
	if err := h.db.DeleteAPIKey(c.Request.Context(), keyID); err != nil {
		h.logger.WithError(err).Error("Failed to delete API key")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete API key",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"key_id":  keyID,
	}).Info("Personal API key deleted")

	// Audit log: API Key deleted
	if h.auditLogger != nil {
		_ = h.auditLogger.LogAPIKeyDeleted(c.Request.Context(), userID, keyID, c.ClientIP())
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API key deleted successfully",
	})
}

// ChangePassword изменяет пароль пользователя
// POST /api/users/me/password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := middleware.RequireJWTAuth(c)
	if !exists {
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Current password and new password required",
		})
		return
	}

	// Get user
	user, err := h.db.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get user",
		})
		return
	}

	// Verify current password
	if err := password.Verify(user.PasswordHash, req.CurrentPassword); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Incorrect current password",
		})
		return
	}

	// Validate new password
	if err := password.Validate(req.NewPassword); err != nil {
		if valErr, ok := err.(password.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": valErr.Message,
				"field": valErr.Field,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid password",
		})
		return
	}

	// Hash new password
	hashedPassword, err := password.Hash(req.NewPassword)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to change password",
		})
		return
	}

	// Update password (using dedicated method for better performance)
	if err := h.db.UpdateUserPassword(c.Request.Context(), userID, hashedPassword); err != nil {
		h.logger.WithError(err).Error("Failed to update user password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to change password",
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("User password changed")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}
