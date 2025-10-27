// Package handlers provides HTTP handlers for the API
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/converter"
	"aigateway/internal/models"
)

// EmbeddingsHandler обрабатывает запросы на создание embeddings
type EmbeddingsHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
}

// NewEmbeddingsHandler создает новый handler для embeddings
func NewEmbeddingsHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *EmbeddingsHandler {
	return &EmbeddingsHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
	}
}

// HandleEmbeddings обрабатывает POST /v1/embeddings
func (h *EmbeddingsHandler) HandleEmbeddings(c *gin.Context) {
	var req models.EmbeddingRequest

	// Парсим запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid embeddings request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Invalid request format",
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":         req.Model,
		"user":          req.User,
		"default_model": h.config.Models.DefaultEmbeddingModel,
	}).Debug("Processing embeddings request")

	// Конвертируем в Ollama format с поддержкой дефолтной модели
	ollamaReq, err := converter.ConvertEmbeddingRequest(&req, h.config.Models.DefaultEmbeddingModel)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert embedding request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			},
		})
		return
	}

	// Вызываем Ollama API
	ollamaResp, err := h.ollamaClient.Embed(c.Request.Context(), ollamaReq)
	if err != nil {
		h.logger.WithError(err).Error("Ollama embed request failed")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to generate embeddings",
				Type:    "api_error",
				Code:    "embedding_error",
			},
		})
		return
	}

	// Конвертируем ответ в OpenAI format
	openaiResp, err := converter.ConvertEmbeddingResponse(ollamaResp, req.Model)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert embedding response")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to format embeddings response",
				Type:    "api_error",
				Code:    "response_error",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":      req.Model,
		"embeddings": len(openaiResp.Data),
		"tokens":     openaiResp.Usage.TotalTokens,
	}).Info("Embeddings generated successfully")

	c.JSON(http.StatusOK, openaiResp)
}

