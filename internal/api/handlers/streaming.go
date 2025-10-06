// Package handlers provides streaming support for chat completions
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/converter"
	"ollama-openai-proxy/internal/models"
)

// StreamingChatHandler обрабатывает streaming chat completions
type StreamingChatHandler struct {
	config          *config.Config
	logger          *logrus.Logger
	ollamaClient    OllamaClientInterface
	converter       *converter.SimpleConverter
	streamConverter *converter.StreamConverter
}

// NewStreamingChatHandler создает новый streaming handler
func NewStreamingChatHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *StreamingChatHandler {
	return &StreamingChatHandler{
		config:          cfg,
		logger:          logger,
		ollamaClient:    ollamaClient,
		converter:       converter.NewSimpleConverter(cfg, logger),
		streamConverter: converter.NewStreamConverter(cfg, logger),
	}
}

// HandleStreamingCompletion обрабатывает streaming chat completion
func (h *StreamingChatHandler) HandleStreamingCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	h.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"messages_count": len(req.Messages),
		"stream":         req.Stream,
		"tools_count":    len(req.Tools),
		"tool_choice":    req.ToolChoice,
	}).Info("Starting streaming chat completion")

	// Детальное логирование tools если они есть
	if len(req.Tools) > 0 {
		h.logger.WithField("tools_count", len(req.Tools)).Warn("IMPORTANT: Tools received in streaming request")
		for i, tool := range req.Tools {
			h.logger.WithFields(logrus.Fields{
				"tool_index": i,
				"tool_type":  tool.Type,
				"func_name":  tool.Function.Name,
			}).Info("Streaming tool definition")
		}
	}

	// Генерируем ID запроса
	requestID := h.converter.GenerateRequestID()

	// 🔧 КРИТИЧНО: Сбрасываем состояние конвертера для нового запроса
	h.streamConverter.Reset()

	// Создаем контекст с увеличенным timeout для больших моделей
	timeout := 120 * time.Second

	// Для больших моделей (30B+) увеличиваем timeout
	if h.isLargeModel(req.Model) {
		timeout = 300 * time.Second // 5 минут для больших моделей
		h.logger.WithField("model", req.Model).Debug("Using extended timeout for large model")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	// Настраиваем SSE headers ПОСЛЕ успешной конвертации
	defer h.setupSSEHeaders(c)

	// Метрики для streaming
	metrics := converter.NewStreamingMetrics(requestID, req.Model)
	defer h.logStreamingMetrics(metrics)

	// 1. Убеждаемся что модель готова (особенно важно для больших моделей)
	if h.isLargeModel(req.Model) {
		h.logger.WithField("model", req.Model).Info("Large model detected, ensuring model readiness")

		// Для streaming НЕ используем warmup - он может конфликтовать
		h.logger.WithField("model", req.Model).Debug("Skipping warmup for streaming request")
	}

	// 2. Конвертируем запрос в Ollama формат
	ollamaReq, err := h.converter.ConvertChatRequest(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Request conversion failed: " + err.Error(),
				"type":    "invalid_request_error",
				"code":    "conversion_error",
			},
		})
		metrics.Finalize(false, err)
		return
	}

	// 3. Настраиваем SSE headers ДО начала streaming
	h.setupSSEHeaders(c)

	// 4. Создаем streaming клиент с увеличенным timeout
	streamingClient := ollama.NewStreamingClient(h.getOllamaClientWithTimeout(req.Model))
	if streamingClient == nil {
		h.sendStreamingError(c, fmt.Errorf("failed to create streaming client"))
		metrics.Finalize(false, fmt.Errorf("no streaming client"))
		return
	}

	// 5. Запускаем streaming запрос
	responseChan, errorChan := streamingClient.ChatCompletionStream(ctx, ollamaReq)

	// 4. Обрабатываем streaming ответ
	h.processStreamingResponse(c, responseChan, errorChan, req, requestID, metrics)
}

// setupSSEHeaders настраивает заголовки для Server-Sent Events
func (h *StreamingChatHandler) setupSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Отключаем buffering в nginx
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Headers", "Cache-Control")

	// Устанавливаем статус OK сразу
	c.Status(http.StatusOK)

	// Принудительно flush headers
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// processStreamingResponse обрабатывает streaming ответ от Ollama
func (h *StreamingChatHandler) processStreamingResponse(
	c *gin.Context,
	responseChan <-chan *ollama.ChatResponse,
	errorChan <-chan error,
	originalReq *models.ChatCompletionRequest,
	requestID string,
	metrics *converter.StreamingMetrics,
) {
	// Получаем writer для потоковой записи
	writer := c.Writer
	flusher, ok := writer.(http.Flusher)
	if !ok {
		h.sendStreamingError(c, fmt.Errorf("streaming not supported by client"))
		metrics.Finalize(false, fmt.Errorf("no flusher available"))
		return
	}

	h.logger.WithField("request_id", requestID).Debug("Starting to process streaming chunks")

	chunksProcessed := 0
	totalTokens := 0

	for {
		select {
		case ollamaChunk, ok := <-responseChan:
			if !ok {
				// Канал закрыт, отправляем финальный [DONE] event
				h.logger.WithFields(logrus.Fields{
					"request_id":       requestID,
					"chunks_processed": chunksProcessed,
					"total_tokens":     totalTokens,
				}).Debug("Streaming completed, sending [DONE] event")

				doneEvent := h.streamConverter.CreateDoneEvent()
				writer.WriteString(doneEvent)
				flusher.Flush()

				metrics.Finalize(true, nil)
				return
			}

			// Записываем метрики первого chunk'а
			if chunksProcessed == 0 {
				metrics.RecordFirstChunk()
			}

			// Конвертируем Ollama chunk в OpenAI формат
			openaiChunk, err := h.streamConverter.ConvertOllamaChunkToOpenAI(ollamaChunk, originalReq, requestID)
			if err != nil {
				h.logger.WithError(err).Error("Failed to convert streaming chunk")
				continue // Пропускаем этот chunk, но продолжаем streaming
			}

			// Конвертируем в SSE формат
			sseEvent, err := h.streamConverter.ConvertToServerSentEvent(openaiChunk)
			if err != nil {
				h.logger.WithError(err).Error("Failed to convert chunk to SSE")
				continue
			}

			// Отправляем chunk клиенту
			writer.WriteString(sseEvent)
			flusher.Flush()

			// Обновляем метрики
			chunkTokens := len(ollamaChunk.Message.Content) / 4 // Приблизительно
			metrics.RecordChunk(chunkTokens)
			chunksProcessed++
			totalTokens += chunkTokens

			h.logger.WithFields(logrus.Fields{
				"request_id":   requestID,
				"chunk_index":  chunksProcessed,
				"chunk_tokens": chunkTokens,
				"done":         ollamaChunk.Done,
			}).Debug("Sent streaming chunk to client")

		case err, ok := <-errorChan:
			if !ok {
				// Канал ошибок закрыт без ошибки
				return
			}

			h.logger.WithError(err).WithField("request_id", requestID).Error("Streaming error from Ollama")

			// Отправляем ошибку как SSE event
			errorEvent := h.streamConverter.CreateErrorEvent(err)
			writer.WriteString(errorEvent)
			flusher.Flush()

			metrics.Finalize(false, err)
			return

		case <-c.Request.Context().Done():
			h.logger.WithField("request_id", requestID).Info("Client disconnected during streaming")
			metrics.Finalize(false, fmt.Errorf("client disconnected"))
			return
		}
	}
}

// sendStreamingError отправляет ошибку в streaming формате
func (h *StreamingChatHandler) sendStreamingError(c *gin.Context, err error) {
	h.logger.WithError(err).Error("Streaming error")

	// Если headers еще не отправлены, используем JSON error
	if !c.Writer.Written() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "api_error",
				"code":    "streaming_error",
			},
		})
		return
	}

	// Если streaming уже начат, отправляем ошибку как SSE event
	errorEvent := h.streamConverter.CreateErrorEvent(err)
	c.Writer.WriteString(errorEvent)

	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// getOllamaClient получает базовый Ollama клиент для streaming
func (h *StreamingChatHandler) getOllamaClient() *ollama.Client {
	// Создаем новый базовый клиент специально для streaming
	client, err := ollama.NewClient(h.config, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create streaming client")
		return nil
	}
	return client
}

// getOllamaClientWithTimeout создает клиент с настроенным timeout для модели
func (h *StreamingChatHandler) getOllamaClientWithTimeout(modelName string) *ollama.Client {
	// Создаем копию конфигурации с увеличенным timeout
	configCopy := *h.config

	// Для больших моделей увеличиваем timeout
	if h.isLargeModel(modelName) {
		configCopy.Ollama.Timeout = 300 * time.Second // 5 минут для больших моделей
		h.logger.WithFields(logrus.Fields{
			"model":   modelName,
			"timeout": configCopy.Ollama.Timeout,
		}).Debug("Using extended timeout for large model")
	}

	client, err := ollama.NewClient(&configCopy, h.logger)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create timeout-adjusted streaming client")
		return nil
	}

	return client
}

// logStreamingMetrics логирует метрики streaming запроса
func (h *StreamingChatHandler) logStreamingMetrics(metrics *converter.StreamingMetrics) {
	h.logger.WithFields(logrus.Fields{
		"request_id":             metrics.RequestID,
		"model":                  metrics.Model,
		"success":                metrics.Success,
		"total_duration_ms":      metrics.TotalDuration.Milliseconds(),
		"time_to_first_chunk_ms": metrics.GetTimeToFirstChunk().Milliseconds(),
		"chunks_count":           metrics.ChunksCount,
		"completion_tokens":      metrics.CompletionTokens,
		"streaming_rate":         fmt.Sprintf("%.2f chunks/sec", metrics.GetStreamingRate()),
		"tokens_per_second":      fmt.Sprintf("%.2f tokens/sec", metrics.GetTokensPerSecond()),
		"error_message":          metrics.ErrorMessage,
	}).Info("Streaming request completed")
}

// isLargeModel проверяет является ли модель большой (требует больше времени)
func (h *StreamingChatHandler) isLargeModel(modelName string) bool {
	// Модели 30B+ параметров считаем большими
	largeModelPatterns := []string{
		"30b", "70b", "405b", "gpt-oss-c32k", "qwen3-coder:30b", "qwen3-coder-30b",
	}

	modelLower := strings.ToLower(modelName)
	for _, pattern := range largeModelPatterns {
		if strings.Contains(modelLower, pattern) {
			return true
		}
	}

	return false
}
