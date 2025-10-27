// Package handlers provides model preloading API endpoints
// Version 1.12.1+: Model Preloading & Warming
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/services/model"
)

// ModelPreloadHandler обрабатывает endpoints для управления preloading
type ModelPreloadHandler struct {
	logger         *logrus.Logger
	modelPreloader *model.ModelPreloader
}

// NewModelPreloadHandler создает новый preload handler
func NewModelPreloadHandler(logger *logrus.Logger, preloader *model.ModelPreloader) *ModelPreloadHandler {
	return &ModelPreloadHandler{
		logger:         logger,
		modelPreloader: preloader,
	}
}

// GetLoadedModels возвращает список загруженных моделей
// GET /api/admin/models/loaded
func (h *ModelPreloadHandler) GetLoadedModels(c *gin.Context) {
	if h.modelPreloader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Model preloader not configured",
		})
		return
	}

	models := h.modelPreloader.GetLoadedModels()

	c.JSON(http.StatusOK, gin.H{
		"loaded_models": models,
		"count":         len(models),
	})
}

// PreloadModel вручную загружает модель
// POST /api/admin/models/:name/preload
func (h *ModelPreloadHandler) PreloadModel(c *gin.Context) {
	if h.modelPreloader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Model preloader not configured",
		})
		return
	}

	modelName := c.Param("name")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "model name is required",
		})
		return
	}

	h.logger.WithField("model", modelName).Info("Manual preload requested")

	if err := h.modelPreloader.PreloadModel(c.Request.Context(), modelName); err != nil {
		h.logger.WithError(err).Errorf("Failed to preload model: %s", modelName)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to preload model",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Model preloaded successfully",
		"model":   modelName,
	})
}


