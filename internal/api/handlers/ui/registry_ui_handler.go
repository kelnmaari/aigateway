package ui

import (
	"net/http"

	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/web/templates"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RegistryUIHandler handles UI rendering for Model Registry
type RegistryUIHandler struct {
	db       storage.Database
	renderer *templates.Renderer
	logger   *logrus.Logger
}

// NewRegistryUIHandler creates a new registry UI handler
func NewRegistryUIHandler(db storage.Database, renderer *templates.Renderer, log *logrus.Logger) *RegistryUIHandler {
	return &RegistryUIHandler{
		db:       db,
		renderer: renderer,
		logger:   log,
	}
}

// GetModelsTable returns HTML table rows for models
// GET /api/ui/registry/models
func (h *RegistryUIHandler) GetModelsTable(c *gin.Context) {
	models, err := h.db.ListModelRegistry(c.Request.Context(), nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list models")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to load models",
		})
		return
	}

	data := gin.H{
		"Models": models,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderModelsTable(c.Writer, data); err != nil {
		h.logger.WithError(err).Error("Failed to render models table")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// SearchModels returns filtered HTML table rows
// GET /api/ui/registry/models/search?q=llama
func (h *RegistryUIHandler) SearchModels(c *gin.Context) {
	query := c.Query("q")

	// Use filter to search models by model_id or name
	filter := &models.ModelRegistryFilter{}
	// Note: Need to implement search logic in DB layer
	// For now, just list all models (TODO: add search support)
	
	modelsList, err := h.db.ListModelRegistry(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to search models")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Search failed",
		})
		return
	}

	data := gin.H{
		"Models": modelsList,
		"Query":  query,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderModelsTable(c.Writer, data); err != nil {
		h.logger.WithError(err).Error("Failed to render search results")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render template",
		})
	}
}

// GetNewModelForm returns HTML form for creating a new model
// GET /api/ui/registry/models/new-form
func (h *RegistryUIHandler) GetNewModelForm(c *gin.Context) {
	// Get available providers
	providers, err := h.db.ListModelProviders(c.Request.Context(), false)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list providers")
		providers = []*models.ModelProvider{} // fallback to empty list
	}

	data := gin.H{
		"Providers": providers,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/registry/model_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render model form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

// GetEditModelForm returns HTML form for editing a model
// GET /api/ui/registry/models/:id/edit
func (h *RegistryUIHandler) GetEditModelForm(c *gin.Context) {
	modelID := c.Param("id")

	model, err := h.db.GetModelRegistry(c.Request.Context(), modelID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model")
		c.HTML(http.StatusNotFound, "", gin.H{
			"error": "Model not found",
		})
		return
	}

	// Get available providers
	providers, err := h.db.ListModelProviders(c.Request.Context(), false)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list providers")
		providers = []*models.ModelProvider{}
	}

	data := gin.H{
		"Model":     model,
		"Providers": providers,
		"IsEdit":    true,
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/registry/model_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render edit form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

// GetNewProviderForm returns HTML form for adding a new provider
// POST /api/ui/registry/providers/new
func (h *RegistryUIHandler) GetNewProviderForm(c *gin.Context) {
	data := gin.H{}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.RenderPartial(c.Writer, "partials/registry/provider_form.html", data); err != nil {
		h.logger.WithError(err).Error("Failed to render provider form")
		c.HTML(http.StatusInternalServerError, "", gin.H{
			"error": "Failed to render form",
		})
	}
}

