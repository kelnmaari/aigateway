// Package handlers provides models endpoint handler
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/converter"
)

// ModelsHandler обрабатывает эндпоинты управления моделями
type ModelsHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
	converter    *converter.SimpleConverter
}

// NewModelsHandler создает новый models handler
func NewModelsHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *ModelsHandler {
	return &ModelsHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
	}
}

// NewModelsHandlerWithManager создает новый models handler (совместимость, model manager не используется)
func NewModelsHandlerWithManager(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface, _ interface{}) *ModelsHandler {
	// Model manager больше не нужен - используем прямой подход
	return &ModelsHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
	}
}

// List обрабатывает GET /v1/models
func (h *ModelsHandler) List(c *gin.Context) {
	h.logger.WithFields(logrus.Fields{
		"endpoint": "/v1/models",
		"method":   "GET",
	}).Info("Models list requested")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Получаем список моделей напрямую из Ollama
	h.logger.Debug("Getting models directly from Ollama")
	ollamaModels, err := h.ollamaClient.GetModels(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get models from Ollama")

		// Конвертируем ошибку в OpenAI формат
		errorResp := h.converter.ConvertErrorResponse(err, "service_unavailable")
		c.JSON(http.StatusServiceUnavailable, errorResp)
		return
	}

	// Конвертируем в OpenAI формат (прямое преобразование)
	response, err := h.converter.ConvertModelsResponse(ollamaModels)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert models response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Models conversion failed: " + err.Error(),
				"type":    "api_error",
				"code":    "conversion_error",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"models_count":       len(response.Data),
		"direct_from_ollama": true,
	}).Debug("Successfully retrieved models from Ollama")

	c.JSON(http.StatusOK, response)
}

