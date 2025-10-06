// Package handlers provides HTTP handlers for authentication endpoints
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/middleware"
	"ollama-openai-proxy/internal/auth/password"
	"ollama-openai-proxy/internal/auth/service"
)

// AuthHandler обрабатывает authentication запросы
type AuthHandler struct {
	authService *service.AuthService
	logger      *logrus.Logger
}

// NewAuthHandler создает новый Auth Handler
func NewAuthHandler(authService *service.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Register регистрирует нового пользователя
// POST /api/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	var req service.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Debug("Invalid register request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Register user
	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		// Check if validation error
		if valErr, ok := err.(password.ValidationError); ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": valErr.Message,
				"field": valErr.Field,
			})
			return
		}

		h.logger.WithError(err).Error("Failed to register user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register user",
		})
		return
	}

	h.logger.WithField("user_id", resp.User.ID).Info("User registered successfully")

	c.JSON(http.StatusCreated, resp)
}

// Login выполняет вход пользователя
// POST /api/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req service.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Debug("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	// Login user
	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		// Check if validation error
		if valErr, ok := err.(password.ValidationError); ok {
			h.logger.WithFields(map[string]interface{}{
				"field":   valErr.Field,
				"message": valErr.Message,
				"request": req.Username,
			}).Warn("Login validation failed")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": valErr.Message,
				"field": valErr.Field,
			})
			return
		}

		h.logger.WithError(err).Error("Failed to login user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to login",
		})
		return
	}

	h.logger.WithField("user_id", resp.User.ID).Info("User logged in successfully")

	c.JSON(http.StatusOK, resp)
}

// Logout выполняет выход пользователя
// POST /api/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get JWT claims from context (set by middleware)
	claims, exists := middleware.GetClaims(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	// Get token from header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing authorization header",
		})
		return
	}

	// Extract token (remove "Bearer " prefix)
	token := authHeader[7:]

	// Logout (revoke token)
	if err := h.authService.Logout(c.Request.Context(), token, claims.ExpiresAt.Time); err != nil {
		h.logger.WithError(err).Error("Failed to logout user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to logout",
		})
		return
	}

	h.logger.WithField("user_id", claims.UserID).Info("User logged out successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// RefreshToken обновляет access token
// POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refresh_token is required",
		})
		return
	}

	// Refresh token
	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Debug("Failed to refresh token")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, tokenPair)
}

// Me возвращает информацию о текущем пользователе
// GET /api/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	// Get user info from context (set by middleware)
	userID, exists := middleware.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	username, _ := middleware.GetUsername(c)
	email, _ := middleware.GetEmail(c)
	tenantIDs, _ := middleware.GetTenantIDs(c)

	c.JSON(http.StatusOK, gin.H{
		"user_id":    userID,
		"username":   username,
		"email":      email,
		"tenant_ids": tenantIDs,
	})
}

// ChangePassword изменяет пароль пользователя
// POST /api/auth/password/change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	// TODO: Implement password change
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": "Not implemented yet",
	})
}
