// Package handlers provides HTTP handlers for the API
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/converter"
	"aigateway/internal/models"
)

// CompletionsHandler обрабатывает запросы legacy completions API
type CompletionsHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
	converter    *converter.SimpleConverter
}

// NewCompletionsHandler создает новый handler для completions
func NewCompletionsHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *CompletionsHandler {
	return &CompletionsHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
	}
}

// HandleCompletions обрабатывает POST /v1/completions (legacy API)
func (h *CompletionsHandler) HandleCompletions(c *gin.Context) {
	var req models.CompletionRequest

	// Парсим запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Warn("Invalid completion request")
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
		"model":  req.Model,
		"stream": req.Stream,
	}).Debug("Processing legacy completion request")

	// Конвертируем completion request в chat request
	chatReq, err := converter.ConvertCompletionToChatRequest(&req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert completion to chat request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Code:    "invalid_request",
			},
		})
		return
	}

	// Обрабатываем как chat completion (streaming или non-streaming)
	if req.Stream {
		h.handleStreamingCompletion(c, chatReq, req.Model)
	} else {
		h.handleNonStreamingCompletion(c, chatReq, req.Model)
	}
}

// handleNonStreamingCompletion обрабатывает non-streaming completion
func (h *CompletionsHandler) handleNonStreamingCompletion(c *gin.Context, chatReq *models.ChatCompletionRequest, originalModel string) {
	// Генерируем ID запроса
	requestID := h.converter.GenerateRequestID()

	// Конвертируем в Ollama format
	ollamaReq, err := h.converter.ConvertChatRequest(chatReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert chat request to ollama format")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: err.Error(),
				Type:    "invalid_request_error",
				Code:    "conversion_error",
			},
		})
		return
	}

	// Вызываем Ollama
	ollamaResp, err := h.ollamaClient.ChatCompletion(c.Request.Context(), ollamaReq)
	if err != nil {
		h.logger.WithError(err).Error("Ollama chat completion failed")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to generate completion",
				Type:    "api_error",
				Code:    "completion_error",
			},
		})
		return
	}

	// Конвертируем ответ в OpenAI chat format
	chatResp, err := h.converter.ConvertChatResponse(ollamaResp, chatReq, requestID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert ollama response")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to format completion response",
				Type:    "api_error",
				Code:    "response_error",
			},
		})
		return
	}

	// Конвертируем chat response в completion response
	completionResp, err := converter.ConvertChatToCompletionResponse(chatResp, originalModel)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert chat response to completion format")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: "Failed to format completion response",
				Type:    "api_error",
				Code:    "response_error",
			},
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"model":        originalModel,
		"choices":      len(completionResp.Choices),
		"total_tokens": completionResp.Usage.TotalTokens,
	}).Info("Legacy completion generated successfully")

	c.JSON(http.StatusOK, completionResp)
}

// handleStreamingCompletion обрабатывает streaming completion
func (h *CompletionsHandler) handleStreamingCompletion(c *gin.Context, chatReq *models.ChatCompletionRequest, originalModel string) {
	// Устанавливаем заголовки для SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")

	// Генерируем ID запроса
	requestID := fmt.Sprintf("cmpl-%d", time.Now().Unix())
	created := time.Now().Unix()

	// Конвертируем в Ollama format
	ollamaReq, err := h.converter.ConvertChatRequest(chatReq)
	if err != nil {
		h.logger.WithError(err).Error("Failed to convert chat request to ollama format")
		sendCompletionErrorChunk(c.Writer, "Failed to convert request")
		return
	}

	// Создаем streaming клиент
	responseChan, errorChan := h.ollamaClient.ChatCompletionStream(c.Request.Context(), ollamaReq)

	// Получаем writer для потоковой записи
	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		h.logger.Error("Streaming not supported by client")
		sendCompletionErrorChunk(writer, "Streaming not supported")
		return
	}

	h.logger.WithField("request_id", requestID).Debug("Starting completion streaming")

	// Обрабатываем streaming ответ
	for {
		select {
		case ollamaChunk, ok := <-responseChan:
			if !ok {
				// Канал закрыт, отправляем финальный [DONE] event
				h.logger.WithField("request_id", requestID).Debug("Streaming completed")
				fmt.Fprintf(writer, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			// Извлекаем content из Ollama chunk
			content := ""
			if ollamaChunk.Message.Content != "" {
				content = ollamaChunk.Message.Content
			}

			// Создаем completion chunk
			finishReason := ""
			if ollamaChunk.Done {
				finishReason = "stop"
			}

			chunk := models.CompletionStreamChunk{
				ID:      requestID,
				Object:  "text_completion",
				Created: created,
				Model:   originalModel,
				Choices: []models.CompletionStreamChoice{
					{
						Text:         content,
						Index:        0,
						FinishReason: finishReason,
					},
				},
			}

			// Отправляем chunk
			chunkData, err := json.Marshal(chunk)
			if err != nil {
				h.logger.WithError(err).Warn("Failed to marshal completion chunk")
				continue
			}

			fmt.Fprintf(writer, "data: %s\n\n", chunkData)
			flusher.Flush()

		case err := <-errorChan:
			if err != nil {
				h.logger.WithError(err).Error("Streaming error occurred")
				sendCompletionErrorChunk(writer, err.Error())
				return
			}
		}
	}
}

// sendCompletionErrorChunk отправляет error chunk для streaming completion
func sendCompletionErrorChunk(w gin.ResponseWriter, message string) {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    "api_error",
			"code":    "completion_error",
		},
	}
	data, _ := json.Marshal(errorData)
	fmt.Fprintf(w, "data: %s\n\n", data)
	w.Flush()
}

