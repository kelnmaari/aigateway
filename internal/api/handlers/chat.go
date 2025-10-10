// Package handlers provides chat completions handler with proper converter integration
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/converter"
	"ollama-openai-proxy/internal/models"
)

// ChatHandler обрабатывает chat completions эндпоинт с упрощенными конвертерами
type ChatHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
	converter    *converter.SimpleConverter
}

// NewChatHandler создает новый chat handler с упрощенными конвертерами
func NewChatHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *ChatHandler {
	return &ChatHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
	}
}

// NewChatHandlerWithManager создает новый chat handler (совместимость, model manager не используется)
func NewChatHandlerWithManager(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface, _ interface{}) *ChatHandler {
	// Model manager больше не нужен - используем прямой подход
	return &ChatHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
	}
}

// Completion обрабатывает POST /v1/chat/completions
func (h *ChatHandler) Completion(c *gin.Context) {
	var req models.ChatCompletionRequest

	// Валидация JSON запроса
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid chat completion request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request format: " + err.Error(),
				"type":    "invalid_request_error",
				"code":    "invalid_request",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"messages_count": len(req.Messages),
		"temperature":    req.Temperature,
		"max_tokens":     req.MaxTokens,
		"stream":         req.Stream,
		"tools_count":    len(req.Tools),
		"tool_choice":    req.ToolChoice,
	}).Info("Chat completion request received")

	// Сохраняем модель в контекст для usage tracking
	c.Set("model", req.Model)

	// Детальное логирование tools если они есть
	if len(req.Tools) > 0 {
		h.logger.WithField("tools_count", len(req.Tools)).Info("Request includes tools")
		for i, tool := range req.Tools {
			h.logger.WithFields(logrus.Fields{
				"tool_index": i,
				"tool_type":  tool.Type,
				"func_name":  tool.Function.Name,
				"func_desc":  tool.Function.Description,
			}).Debug("Tool definition received")
		}
	}

	// Проверка поддержки streaming
	if req.Stream {
		h.logger.WithField("model", req.Model).Info("Handling streaming chat completion")
		h.handleStreamingCompletion(c, &req)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	// Генерируем ID запроса
	requestID := h.converter.GenerateRequestID()

	// 1. Преобразовать запрос OpenAI в формат Ollama (прямая конвертация)
	ollamaReq, err := h.converter.ConvertChatRequest(&req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert request to Ollama format")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Request conversion failed: " + err.Error(),
				"type":    "invalid_request_error",
				"code":    "conversion_error",
			},
		})
		return
	}

	// 2. Отправить запрос в Ollama
	ollamaResp, err := h.ollamaClient.ChatCompletion(ctx, ollamaReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get chat completion from Ollama")

		// Конвертируем ошибку в OpenAI формат
		errorResp := h.converter.ConvertErrorResponse(err, "service_unavailable")
		c.JSON(http.StatusServiceUnavailable, errorResp)
		return
	}

	// 3. Преобразовать ответ Ollama в формат OpenAI
	response, err := h.converter.ConvertChatResponse(ollamaResp, &req, requestID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert Ollama response to OpenAI format")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Response conversion failed: " + err.Error(),
				"type":    "api_error",
				"code":    "conversion_error",
			},
		})
		return
	}

	// 4. Устанавливаем токены в контекст для usage tracking
	c.Set("prompt_tokens", response.Usage.PromptTokens)
	c.Set("completion_tokens", response.Usage.CompletionTokens)
	c.Set("total_tokens", response.Usage.TotalTokens)

	h.logger.WithFields(logrus.Fields{
		"prompt_tokens":     response.Usage.PromptTokens,
		"completion_tokens": response.Usage.CompletionTokens,
		"total_tokens":      response.Usage.TotalTokens,
	}).Debug("Tokens set in context for usage tracking")

	// 5. Возвращаем ответ
	h.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"response_id":       response.ID,
		"prompt_tokens":     response.Usage.PromptTokens,
		"completion_tokens": response.Usage.CompletionTokens,
		"total_tokens":      response.Usage.TotalTokens,
		"model":             response.Model,
	}).Info("Chat completion response generated successfully")

	c.JSON(http.StatusOK, response)
}

// handleStreamingCompletion обрабатывает streaming запрос
func (h *ChatHandler) handleStreamingCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	// Создаем streaming handler
	streamingHandler := NewStreamingChatHandler(h.config, h.logger, h.ollamaClient)

	// Передаем обработку streaming handler'у
	streamingHandler.HandleStreamingCompletion(c, req)
}
