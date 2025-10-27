// Package handlers provides chat completions handler with proper converter integration
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/converter"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/rag/orchestrator"
	"ollama-openai-proxy/internal/storage"
	"ollama-openai-proxy/internal/webfetch"
)

// ModelPreloader interface для tracking model usage (v1.12.1+)
type ModelPreloader interface {
	MarkUsed(modelName string)
}

// ChatHandler обрабатывает chat completions эндпоинт с упрощенными конвертерами
type ChatHandler struct {
	config              *config.Config
	logger              *logrus.Logger
	ollamaClient        OllamaClientInterface
	converter           *converter.SimpleConverter
	db                  storage.Database              // для model configs (v1.9.1+)
	webfetchIntegration *webfetch.ChatIntegration     // для автоматического fetch web content (v1.10.4+)
	modelPreloader      ModelPreloader                // для tracking model usage (v1.12.1+)
	ragOrchestrator     *orchestrator.RAGOrchestrator // для RAG system (v1.13.0+)
}

// NewChatHandler создает новый chat handler с упрощенными конвертерами
func NewChatHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *ChatHandler {
	return &ChatHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		converter:    converter.NewSimpleConverter(cfg, logger),
		db:           nil, // backward compatibility
	}
}

// NewChatHandlerWithDB создает новый chat handler с database для model configs
func NewChatHandlerWithDB(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface, db storage.Database) *ChatHandler {
	// Create web fetch integration if enabled
	var webfetchIntegration *webfetch.ChatIntegration
	if cfg.WebFetch.Enabled && db != nil {
		webfetchConfig := convertWebFetchConfig(cfg.WebFetch)
		webfetchService := webfetch.NewService(webfetchConfig, db, logger)
		webfetchIntegration = webfetch.NewChatIntegration(webfetchService, logger)
	}

	return &ChatHandler{
		config:              cfg,
		logger:              logger,
		ollamaClient:        ollamaClient,
		converter:           converter.NewSimpleConverter(cfg, logger),
		db:                  db,
		webfetchIntegration: webfetchIntegration,
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
		db:           nil, // backward compatibility
	}
}

// SetModelPreloader устанавливает model preloader для tracking (v1.12.1+)
func (h *ChatHandler) SetModelPreloader(preloader ModelPreloader) {
	h.modelPreloader = preloader
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

	// Track model usage для preloading (v1.12.1+)
	if h.modelPreloader != nil {
		h.modelPreloader.MarkUsed(req.Model)
	}

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

	// Enrich messages with file content if file_ids present (FILE-STORAGE-01: Phase 4)
	if err := h.enrichMessagesWithFiles(c.Request.Context(), &req); err != nil {
		h.logger.WithError(err).Warn("Failed to enrich messages with files")
		// Don't fail the request, just log the warning
	}

	// Enrich messages with web content if URLs detected (WEB-FETCH-01: v1.10.4)
	if err := h.enrichMessagesWithWebContent(c.Request.Context(), &req); err != nil {
		h.logger.WithError(err).Warn("Failed to enrich messages with web content")
		// Don't fail the request, just log the warning
	}

	// RAG System integration (v1.13.0+)
	if req.RAGEnabled {
		if err := h.enrichMessagesWithRAG(c.Request.Context(), &req); err != nil {
			h.logger.WithError(err).Error("Failed to enrich messages with RAG context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "RAG processing failed: " + err.Error(),
					"type":    "api_error",
					"code":    "rag_error",
				},
			})
			return
		}
		h.logger.Info("Successfully enriched message with RAG context")
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

	// 0. Применяем effective model config (v1.9.1+)
	if h.db != nil {
		// Получаем user_id и tenant_id из контекста (устанавливаются auth middleware)
		userID, _ := c.Get("user_id")
		tenantID, _ := c.Get("tenant_id")

		userIDStr := ""
		tenantIDStr := ""
		if userID != nil {
			userIDStr, _ = userID.(string)
		}
		if tenantID != nil {
			tenantIDStr, _ = tenantID.(string)
		}

		// Получаем effective config для модели
		effectiveParams, err := h.db.GetEffectiveModelConfig(ctx, req.Model, userIDStr, tenantIDStr)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to get effective model config, using defaults")
		} else if effectiveParams != nil {
			// Применяем effective params если они не были явно указаны в запросе
			if req.Temperature == nil && effectiveParams.Temperature != nil {
				temp := float64(*effectiveParams.Temperature)
				req.Temperature = &temp
			}
			if req.TopP == nil && effectiveParams.TopP != nil {
				topP := float64(*effectiveParams.TopP)
				req.TopP = &topP
			}
			if req.MaxTokens == nil && effectiveParams.NumPredict != nil {
				maxTokens := *effectiveParams.NumPredict
				req.MaxTokens = &maxTokens
			}

			h.logger.WithFields(logrus.Fields{
				"model":       req.Model,
				"user_id":     userIDStr,
				"tenant_id":   tenantIDStr,
				"temperature": req.Temperature,
				"top_p":       req.TopP,
				"max_tokens":  req.MaxTokens,
			}).Debug("Applied effective model config")
		}
	}

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

// enrichMessagesWithFiles обогащает сообщения содержимым прикрепленных файлов (FILE-STORAGE-01: Phase 4)
func (h *ChatHandler) enrichMessagesWithFiles(ctx context.Context, req *models.ChatCompletionRequest) error {
	if h.db == nil {
		return nil // No database available
	}

	for i := range req.Messages {
		msg := &req.Messages[i]

		if len(msg.FileIDs) == 0 {
			continue
		}

		// Get original content as string
		originalContent := ""
		switch v := msg.Content.(type) {
		case string:
			originalContent = v
		default:
			h.logger.Warn("Skipping file enrichment for non-string content")
			continue
		}

		// Build enriched content with file texts
		parts := []string{"📎 Прикрепленные файлы:"}

		for _, fileID := range msg.FileIDs {
			file, err := h.db.GetFileByID(ctx, fileID)
			if err != nil {
				h.logger.WithError(err).Warnf("Failed to load file %s", fileID)
				parts = append(parts, fmt.Sprintf("\n\n--- Файл ID: %s ---", fileID))
				parts = append(parts, "[Файл не найден]")
				continue
			}

			parts = append(parts, fmt.Sprintf("\n\n--- Файл: %s (%s) ---", file.Filename, file.MimeType))

			if file.ExtractedText != nil && *file.ExtractedText != "" {
				// Limit content to avoid token overflow (max 10K chars per file)
				text := *file.ExtractedText
				maxLength := 10000
				if len(text) > maxLength {
					text = text[:maxLength] + "\n... (содержимое обрезано)"
				}
				parts = append(parts, text)
			} else if file.ExtractionStatus == "failed" {
				errMsg := "неизвестная ошибка"
				if file.ExtractionError != nil {
					errMsg = *file.ExtractionError
				}
				parts = append(parts, fmt.Sprintf("[Не удалось извлечь текст: %s]", errMsg))
			} else if file.ExtractionStatus == "pending" {
				parts = append(parts, "[Извлечение текста еще не завершено]")
			} else {
				parts = append(parts, "[Текст не извлечен]")
			}
		}

		parts = append(parts, "\n\n--- Сообщение пользователя ---")
		parts = append(parts, originalContent)

		// Update message content
		msg.Content = strings.Join(parts, "\n")

		h.logger.WithFields(logrus.Fields{
			"message_index": i,
			"files_count":   len(msg.FileIDs),
		}).Debug("Enriched message with file content")
	}

	return nil
}

// enrichMessagesWithWebContent обогащает сообщения содержимым веб-страниц (WEB-FETCH-01: v1.10.4)
func (h *ChatHandler) enrichMessagesWithWebContent(ctx context.Context, req *models.ChatCompletionRequest) error {
	// Check if webfetch integration is available
	if h.webfetchIntegration == nil {
		return nil // WebFetch not enabled
	}

	for i := range req.Messages {
		msg := &req.Messages[i]

		// Only process user messages
		if msg.Role != "user" {
			continue
		}

		// Get original content as string
		originalContent := ""
		switch v := msg.Content.(type) {
		case string:
			originalContent = v
		default:
			h.logger.Warn("Skipping web content enrichment for non-string content")
			continue
		}

		// Process message to detect and fetch URLs
		processed, err := h.webfetchIntegration.ProcessMessage(ctx, originalContent, webfetch.ProcessMessageOptions{
			AutoFetch:      true,
			MaxURLs:        2,                // Limit to 2 URLs per message to avoid context overflow
			IncludeHTML:    false,
			Summarize:      false,
			Timeout:        15 * time.Second, // Quick fetch timeout
			TruncateLength: 0,                // 0 = без обрезания контента, отдаем полное содержимое страницы
		})

		if err != nil {
			h.logger.WithError(err).Warn("Failed to process message for web content")
			continue
		}

		// If web content was fetched, update message
		if processed.HasWebContent {
			msg.Content = processed.EnhancedMessage

			h.logger.WithFields(logrus.Fields{
				"message_index": i,
				"urls_detected": len(processed.DetectedURLs),
				"urls_fetched":  len(processed.FetchedPages),
				"urls_failed":   len(processed.FetchErrors),
			}).Info("Enriched message with web content")
		}
	}

	return nil
}

// handleStreamingCompletion обрабатывает streaming запрос
func (h *ChatHandler) handleStreamingCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	// Enrich messages with file content for streaming too (FILE-STORAGE-01: Phase 4)
	if err := h.enrichMessagesWithFiles(c.Request.Context(), req); err != nil {
		h.logger.WithError(err).Warn("Failed to enrich streaming messages with files")
		// Don't fail the request, just log the warning
	}

	// Enrich messages with web content for streaming too (WEB-FETCH-01: v1.10.4)
	if err := h.enrichMessagesWithWebContent(c.Request.Context(), req); err != nil {
		h.logger.WithError(err).Warn("Failed to enrich streaming messages with web content")
		// Don't fail the request, just log the warning
	}

	// Создаем streaming handler
	streamingHandler := NewStreamingChatHandler(h.config, h.logger, h.ollamaClient)

	// Передаем обработку streaming handler'у
	streamingHandler.HandleStreamingCompletion(c, req)
}

// SetRAGOrchestrator устанавливает RAG orchestrator (v1.13.0+)
func (h *ChatHandler) SetRAGOrchestrator(orchestrator *orchestrator.RAGOrchestrator) {
	h.ragOrchestrator = orchestrator
}

// enrichMessagesWithRAG обогащает сообщения контекстом из RAG (v1.13.0+)
func (h *ChatHandler) enrichMessagesWithRAG(ctx context.Context, req *models.ChatCompletionRequest) error {
	if h.ragOrchestrator == nil {
		return fmt.Errorf("RAG orchestrator not initialized")
	}

	// Получаем последнее user message как query
	var userQuery string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			// Extract text from content
			if contentStr, ok := req.Messages[i].Content.(string); ok {
				userQuery = contentStr
				break
			}
		}
	}

	if userQuery == "" {
		return fmt.Errorf("no user query found in messages")
	}

	// Получаем user_id из контекста
	userID := ""
	if val, exists := ctx.Value("user_id").(string); exists {
		userID = val
	}

	// Получаем conversation_id если есть
	convID := ""
	if val, exists := ctx.Value("conversation_id").(string); exists {
		convID = val
	}

	h.logger.WithFields(logrus.Fields{
		"query":      userQuery,
		"source_ids": req.RAGSourceIDs,
		"top_k":      req.RAGTopK,
		"rerank":     req.RAGRerank,
	}).Info("Processing RAG query")

	// Выполняем RAG query
	ragResp, err := h.ragOrchestrator.Query(ctx, orchestrator.RAGRequest{
		Query:      userQuery,
		SourceIDs:  req.RAGSourceIDs,
		TopK:       req.RAGTopK,
		MinScore:   req.RAGMinScore,
		UserID:     userID,
		ConvID:     convID,
		Rerank:     req.RAGRerank,
	})

	if err != nil {
		return fmt.Errorf("RAG query failed: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"chunks":         ragResp.TotalChunks,
		"context_tokens": ragResp.ContextTokens,
		"search_time_ms": ragResp.SearchTime.Milliseconds(),
	}).Info("RAG query completed")

	// Если нет результатов, не добавляем context
	if ragResp.TotalChunks == 0 {
		h.logger.Warn("No RAG results found, proceeding without RAG context")
		return nil
	}

	// Добавляем RAG context как system message в начало
	systemMessage := models.ChatMessage{
		Role:    "system",
		Content: ragResp.Context,
	}

	// Вставляем system message в начало (перед первым user message)
	req.Messages = append([]models.ChatMessage{systemMessage}, req.Messages...)

	h.logger.WithFields(logrus.Fields{
		"chunks_used":    ragResp.TotalChunks,
		"context_tokens": ragResp.ContextTokens,
		"sources":        len(ragResp.SourceChunks),
	}).Info("RAG context added to messages")

	return nil
}
