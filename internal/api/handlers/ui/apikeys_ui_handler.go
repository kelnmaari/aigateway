package ui

import (
	"net/http"

	"aigateway/internal/storage"
	"aigateway/internal/web/templates"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// APIKeysUIHandler handles UI rendering for API Keys
type APIKeysUIHandler struct {
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewAPIKeysUIHandler creates a new API keys UI handler
func NewAPIKeysUIHandler(db storage.Database, renderer *templates.Renderer, log *logrus.Logger) *APIKeysUIHandler {
	return &APIKeysUIHandler{
		db:       db,
		renderer: renderer,
		logger:   log,
	}
}

// GetAPIKeysList returns HTML list of API keys
// GET /api/ui/api-keys/personal
func (h *APIKeysUIHandler) GetAPIKeysList(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.HTML(http.StatusUnauthorized, "", gin.H{
			"error": "Unauthorized",
		})
		return
	}

	// Get all keys for user
	keys, err := h.db.ListAPIKeys(c.Request.Context(), userID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list API keys")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to load API keys",
		})
		return
	}

	data := gin.H{
		"Keys": keys,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderAPIKeysList(c.Writer, data); err != nil {
		h.logger.WithError(err).Error("Failed to render API keys list")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetAPIKeysForTenant returns HTML list of API keys for a tenant
// GET /api/ui/api-keys/tenant/:id
func (h *APIKeysUIHandler) GetAPIKeysForTenant(c *gin.Context) {
	tenantID := c.Param("id")

	keys, err := h.db.ListAPIKeysByTenant(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list tenant API keys")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to load API keys",
		})
		return
	}

	data := gin.H{
		"Keys":     keys,
		"TenantID": tenantID,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderAPIKeysList(c.Writer, data); err != nil {
		h.logger.WithError(err).Error("Failed to render API keys list")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetEditAPIKeyForm returns HTML form for editing an API key
// GET /api/ui/api-keys/:id/edit
func (h *APIKeysUIHandler) GetEditAPIKeyForm(c *gin.Context) {
	keyID := c.Param("id")

	key, err := h.db.GetAPIKey(c.Request.Context(), keyID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API key")
		c.HTML(http.StatusNotFound, "", gin.H{
			"error": "API key not found",
		})
		return
	}

	data := gin.H{
		"Key":    key,
		"IsEdit": true,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/apikeys/key_edit_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render edit form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

// GetCreateAPIKeyForm returns HTML form for creating a new API key
// GET /api/ui/api-keys/create-form
func (h *APIKeysUIHandler) GetCreateAPIKeyForm(c *gin.Context) {
	// Get available models for selection
	models, err := h.db.ListModelRegistry(c.Request.Context(), nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list models")
		models = nil // fallback to empty list
	}

	data := gin.H{
		"Models": models,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/apikeys/key_create_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render create form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

// GetAPIKeyCard returns HTML card for a single API key
// GET /api/ui/api-keys/:id
func (h *APIKeysUIHandler) GetAPIKeyCard(c *gin.Context) {
	keyID := c.Param("id")

	key, err := h.db.GetAPIKey(c.Request.Context(), keyID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get API key")
		c.HTML(http.StatusNotFound, "", gin.H{
			"error": "API key not found",
		})
		return
	}

	data := gin.H{
		"Key": key,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/apikeys/key_card.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render API key card")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

