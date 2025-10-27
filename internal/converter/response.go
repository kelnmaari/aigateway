// Package converter provides response conversion from Ollama to OpenAI format
package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/models"
)

// ResponseConverter обрабатывает конвертацию ответов Ollama в OpenAI формат
type ResponseConverter struct {
	logger       *logrus.Logger
	modelManager ModelManager
}

// NewResponseConverter создает новый конвертер ответов
func NewResponseConverter(logger *logrus.Logger, modelManager ModelManager) *ResponseConverter {
	return &ResponseConverter{
		logger:       logger,
		modelManager: modelManager,
	}
}

// ConvertChatResponse конвертирует Ollama chat ответ в OpenAI формат
func (r *ResponseConverter) ConvertChatResponse(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
	requestID string,
) (*models.ChatCompletionResponse, error) {
	startTime := time.Now()

	r.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"ollama_model":    ollamaResp.Model,
		"original_model":  originalReq.Model,
		"response_length": len(ollamaResp.Message.Content),
		"done":            ollamaResp.Done,
	}).Debug("Converting Ollama chat response to OpenAI format")

	// Создаем базовый ответ
	response := models.NewChatCompletionResponse(requestID, originalReq.Model)

	// Конвертируем сообщение
	message := models.ChatMessage{
		Role:    ollamaResp.Message.Role,
		Content: ollamaResp.Message.Content,
	}

	// Определяем причину завершения
	finishReason := r.determineFinishReason(ollamaResp, originalReq)

	// Создаем choice
	choice := models.ChatCompletionChoice{
		Index:        0,
		Message:      message,
		FinishReason: finishReason,
	}

	response.Choices = []models.ChatCompletionChoice{choice}

	// Устанавливаем время создания из Ollama ответа
	response.Created = ollamaResp.CreatedAt.Unix()

	// Подсчитываем usage
	usage, err := r.calculateUsage(ollamaResp, originalReq)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to calculate usage, using estimates")
		usage = r.estimateUsage(ollamaResp, originalReq)
	}
	response.Usage = *usage

	// Добавляем system fingerprint если доступен
	if ollamaResp.Model != "" {
		response.SystemFingerprint = fmt.Sprintf("ollama-%s", ollamaResp.Model)
	}

	duration := time.Since(startTime)
	r.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"conversion_time":   duration,
		"prompt_tokens":     response.Usage.PromptTokens,
		"completion_tokens": response.Usage.CompletionTokens,
		"total_tokens":      response.Usage.TotalTokens,
		"finish_reason":     finishReason,
	}).Debug("Successfully converted Ollama response to OpenAI format")

	return response, nil
}

// ConvertStreamChunk конвертирует Ollama streaming chunk в OpenAI формат
func (r *ResponseConverter) ConvertStreamChunk(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
	requestID string,
	isLast bool,
) (*models.ChatCompletionChunk, error) {
	r.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"is_last":    isLast,
		"done":       ollamaResp.Done,
	}).Debug("Converting Ollama stream chunk to OpenAI format")

	// Создаем базовый chunk
	chunk := models.NewChatCompletionChunk(requestID, originalReq.Model)

	// Создаем delta сообщение с поддержкой thinking для gpt-oss моделей
	content := ollamaResp.Message.Content

	// Для gpt-oss моделей: отправляем thinking как content
	// Это позволяет видеть процесс рассуждения модели
	if content == "" && ollamaResp.Message.Thinking != "" {
		content = ollamaResp.Message.Thinking
	}

	delta := models.ChatMessage{
		Role:    ollamaResp.Message.Role,
		Content: content,
	}

	// 🔧 КРИТИЧНО: Проверяем tool_calls В ДВУХ МЕСТАХ (message и response)
	var toolCalls []ollama.ToolCall

	// 1. Проверяем message.tool_calls (старый формат)
	if len(ollamaResp.Message.ToolCalls) > 0 {
		toolCalls = ollamaResp.Message.ToolCalls
		r.logger.WithField("source", "message.tool_calls").Debug("Found tool_calls in streaming message")
	}

	// 2. Проверяем response.tool_calls (официальный API формат)
	if len(ollamaResp.ToolCalls) > 0 {
		toolCalls = ollamaResp.ToolCalls
		r.logger.WithField("source", "response.tool_calls").Debug("Found tool_calls in streaming response")
	}

	// 3. Конвертируем если нашли
	if len(toolCalls) > 0 {
		delta.ToolCalls = r.convertToolCallsToOpenAI(toolCalls)
		// ВАЖНО: Очищаем content если есть tool_calls (как в non-streaming)
		delta.Content = ""
		r.logger.WithFields(logrus.Fields{
			"tool_calls_count": len(delta.ToolCalls),
		}).Info("✅ Tool calls converted in STREAMING chunk!")
	}

	// Определяем finish reason для последнего chunk'а
	var finishReason *string
	if isLast || ollamaResp.Done {
		reason := r.determineFinishReason(ollamaResp, originalReq)
		finishReason = &reason
	}

	// Создаем choice для chunk'а
	choice := models.ChatCompletionChunkChoice{
		Index:        0,
		Delta:        delta,
		FinishReason: finishReason,
	}

	chunk.Choices = []models.ChatCompletionChunkChoice{choice}

	// Устанавливаем время создания
	chunk.Created = ollamaResp.CreatedAt.Unix()

	// Добавляем system fingerprint
	if ollamaResp.Model != "" {
		chunk.SystemFingerprint = fmt.Sprintf("ollama-%s", ollamaResp.Model)
	}

	return chunk, nil
}

// ConvertModelsResponse конвертирует Ollama models ответ в OpenAI формат
func (r *ResponseConverter) ConvertModelsResponse(
	ollamaModels *ollama.ModelsResponse,
) (*models.ModelsResponse, error) {
	r.logger.WithField("models_count", len(ollamaModels.Models)).Debug("Converting Ollama models response to OpenAI format")

	openaiModels := make([]models.Model, 0, len(ollamaModels.Models))

	for _, ollamaModel := range ollamaModels.Models {
		// Попытка обратного маппинга модели
		openaiName := ollamaModel.Name
		if mappedName, err := r.modelManager.MapOllamaToOpenAI(ollamaModel.Name); err == nil {
			openaiName = mappedName
		}

		model := models.Model{
			ID:      openaiName,
			Object:  "model",
			Created: ollamaModel.ModifiedAt.Unix(),
			OwnedBy: "ollama",
			Permission: []models.Permission{
				{
					ID:                 fmt.Sprintf("modelperm-%s", openaiName),
					Object:             "model_permission",
					Created:            ollamaModel.ModifiedAt.Unix(),
					AllowCreateEngine:  false,
					AllowSampling:      true,
					AllowLogProbs:      false,
					AllowSearchIndices: false,
					AllowView:          true,
					AllowFineTuning:    false,
					Organization:       "ollama",
					Group:              nil,
					IsBlocking:         false,
				},
			},
		}

		openaiModels = append(openaiModels, model)
	}

	response := &models.ModelsResponse{
		Object: "list",
		Data:   openaiModels,
	}

	r.logger.WithField("converted_models", len(openaiModels)).Debug("Successfully converted models response")

	return response, nil
}

// determineFinishReason определяет причину завершения ответа
func (r *ResponseConverter) determineFinishReason(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
) string {
	// Если ответ не завершен, возвращаем null для streaming
	if !ollamaResp.Done {
		return ""
	}

	// 🔧 КРИТИЧНО: Если есть tool_calls В ЛЮБОМ МЕСТЕ, finish_reason должен быть "tool_calls"
	if len(ollamaResp.Message.ToolCalls) > 0 || len(ollamaResp.ToolCalls) > 0 {
		return "tool_calls"
	}

	// Проверяем лимит токенов
	if originalReq.MaxTokens != nil {
		estimatedTokens := len(ollamaResp.Message.Content) / 4 // Приблизительная оценка
		if estimatedTokens >= *originalReq.MaxTokens {
			return "length"
		}
	}

	// Проверяем stop sequences
	if len(originalReq.Stop) > 0 {
		content := ollamaResp.Message.Content
		for _, stopSeq := range originalReq.Stop {
			if len(content) >= len(stopSeq) && content[len(content)-len(stopSeq):] == stopSeq {
				return "stop"
			}
		}
	}

	// По умолчанию - естественная остановка
	return "stop"
}

// calculateUsage подсчитывает точное использование токенов
func (r *ResponseConverter) calculateUsage(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
) (*models.Usage, error) {
	// Используем данные из Ollama если доступны
	var promptTokens, completionTokens int

	if ollamaResp.PromptEvalCount > 0 {
		promptTokens = ollamaResp.PromptEvalCount
	} else {
		// Оцениваем токены запроса
		promptTokens = r.estimatePromptTokens(originalReq.Messages)
	}

	if ollamaResp.EvalCount > 0 {
		completionTokens = ollamaResp.EvalCount
	} else {
		// Оцениваем токены ответа
		completionTokens = r.estimateCompletionTokens(ollamaResp.Message.Content)
	}

	return &models.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}, nil
}

// estimateUsage приблизительно оценивает использование токенов
func (r *ResponseConverter) estimateUsage(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
) *models.Usage {
	promptTokens := r.estimatePromptTokens(originalReq.Messages)
	completionTokens := r.estimateCompletionTokens(ollamaResp.Message.Content)

	return &models.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// estimatePromptTokens оценивает количество токенов в prompt'е
func (r *ResponseConverter) estimatePromptTokens(messages []models.ChatMessage) int {
	totalChars := 0

	for _, msg := range messages {
		// Добавляем символы роли
		totalChars += len(msg.Role)

		// Добавляем символы содержимого
		content := ""
		if contentStr, ok := msg.Content.(string); ok {
			content = contentStr
		}
		totalChars += len(content)

		// Добавляем накладные расходы на форматирование
		totalChars += 10 // Примерная оценка overhead
	}

	// Конвертируем символы в токены (приблизительно 4 символа = 1 токен)
	return (totalChars + 3) / 4
}

// estimateCompletionTokens оценивает количество токенов в ответе
func (r *ResponseConverter) estimateCompletionTokens(content string) int {
	// Приблизительно 4 символа = 1 токен
	tokens := (len(content) + 3) / 4

	// КРИТИЧНО: Минимум 1 токен для OpenAI API совместимости
	// Даже пустой ответ требует inference и должен быть учтен
	if tokens == 0 {
		tokens = 1
	}

	return tokens
}

// ConvertErrorResponse конвертирует ошибку Ollama в OpenAI формат
func (r *ResponseConverter) ConvertErrorResponse(
	ollamaError error,
	errorType string,
	statusCode int,
) *models.ErrorResponse {
	r.logger.WithFields(logrus.Fields{
		"error":       ollamaError.Error(),
		"error_type":  errorType,
		"status_code": statusCode,
	}).Debug("Converting Ollama error to OpenAI format")

	var openaiErrorType string
	var openaiErrorCode interface{}

	// Маппинг типов ошибок
	switch errorType {
	case "model_not_found":
		openaiErrorType = "invalid_request_error"
		openaiErrorCode = "model_not_found"
	case "invalid_request":
		openaiErrorType = "invalid_request_error"
		openaiErrorCode = "invalid_request"
	case "service_unavailable":
		openaiErrorType = "api_error"
		openaiErrorCode = "service_unavailable"
	case "rate_limit_exceeded":
		openaiErrorType = "rate_limit_error"
		openaiErrorCode = "rate_limit_exceeded"
	default:
		openaiErrorType = "api_error"
		openaiErrorCode = "internal_error"
	}

	return &models.ErrorResponse{
		Error: models.Error{
			Message: ollamaError.Error(),
			Type:    openaiErrorType,
			Code:    openaiErrorCode,
		},
	}
}

// convertToolCallsToOpenAI конвертирует Ollama tool calls в OpenAI формат
func (r *ResponseConverter) convertToolCallsToOpenAI(ollamaToolCalls []ollama.ToolCall) []models.ToolCall {
	openaiToolCalls := make([]models.ToolCall, len(ollamaToolCalls))
	for i, tc := range ollamaToolCalls {
		// Генерируем ID для tool call
		toolCallID := fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), i)

		// Конвертируем arguments в JSON строку для OpenAI формата
		var argsStr string
		if tc.Function.Arguments != nil {
			if str, ok := tc.Function.Arguments.(string); ok {
				argsStr = str
			} else {
				// Конвертируем в JSON строку
				if bytes, err := json.Marshal(tc.Function.Arguments); err == nil {
					argsStr = string(bytes)
				}
			}
		}

		openaiToolCalls[i] = models.ToolCall{
			ID:   toolCallID,
			Type: "function",
			Function: models.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: argsStr,
			},
		}
	}
	return openaiToolCalls
}

