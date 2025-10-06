// Package converter provides simplified conversion without model mapping
package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/optimizer"
)

// SimpleConverter обеспечивает прямую конвертацию без маппинга моделей
type SimpleConverter struct {
	logger    *logrus.Logger
	config    *config.Config
	optimizer *optimizer.PromptOptimizer
}

// NewSimpleConverter создает новый упрощенный конвертер
func NewSimpleConverter(cfg *config.Config, logger *logrus.Logger) *SimpleConverter {
	// Создаем оптимизатор если включен
	var opt *optimizer.PromptOptimizer
	if cfg.Tools.Optimizer.Enabled {
		optimizerCfg := optimizer.OptimizerConfig{
			EnableSystemMessageSimplification: cfg.Tools.Optimizer.SimplifySystemMessage,
			EnableSmartToolFiltering:          cfg.Tools.Optimizer.SmartToolFiltering,
			MaxToolsPerRequest:                cfg.Tools.Optimizer.MaxToolsPerRequest,
			PreserveKeyInstructions:           cfg.Tools.Optimizer.PreserveInstructions,
		}
		opt = optimizer.NewPromptOptimizer(logger, optimizerCfg)
		logger.Info("Smart Prompt Optimizer enabled")
	}

	return &SimpleConverter{
		logger:    logger,
		config:    cfg,
		optimizer: opt,
	}
}

// ConvertChatRequest конвертирует OpenAI запрос в Ollama формат (без маппинга)
func (c *SimpleConverter) ConvertChatRequest(req *models.ChatCompletionRequest) (*ollama.ChatRequest, error) {
	c.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"messages_count": len(req.Messages),
		"stream":         req.Stream,
	}).Debug("Converting OpenAI chat request to Ollama format (direct)")

	// Валидация базовых полей
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// 🎯 SMART PROMPT OPTIMIZER: Оптимизируем промпт для локальных моделей
	if c.optimizer != nil && len(req.Tools) > 0 {
		req = c.optimizer.OptimizeRequest(req)
	}

	// 💬 ДОПОЛНИТЕЛЬНЫЙ СИСТЕМНЫЙ ПРОМПТ: Добавляем дополнительное системное сообщение
	// ⚠️ КРИТИЧНО: НЕ добавляем если есть tools - это сбивает модель с толку!
	if c.config.Prompts.AdditionalSystemMessage != "" && len(req.Tools) == 0 {
		req.Messages = c.injectAdditionalSystemMessage(req.Messages)
		c.logger.Debug("💬 Additional system message added (no tools in request)")
	} else if c.config.Prompts.AdditionalSystemMessage != "" && len(req.Tools) > 0 {
		c.logger.Warn("⚠️ Skipping additional system message due to tools in request (prevents interference)")
	}

	// Конвертируем сообщения напрямую
	messages := make([]ollama.ChatMessage, len(req.Messages))
	for i, msg := range req.Messages {
		content := c.extractMessageContent(msg)
		ollamaMsg := ollama.ChatMessage{
			Role:    msg.Role,
			Content: content,
		}

		// Конвертируем tool_calls если есть
		if len(msg.ToolCalls) > 0 {
			ollamaMsg.ToolCalls = c.convertToolCalls(msg.ToolCalls)
		}

		// Добавляем tool_name для сообщений с role="tool"
		if msg.Role == "tool" && msg.ToolCallID != "" {
			ollamaMsg.ToolName = msg.ToolCallID
		}

		messages[i] = ollamaMsg
	}

	// Конвертируем параметры
	var options *ollama.ChatOptions
	if c.hasOptions(req) {
		temperature := req.Temperature

		// 🎯 AUTO-TEMPERATURE FIX: Снижаем температуру для запросов с tools
		// Высокая температура (1.0) делает модель креативной и она возвращает текст вместо tool_calls
		if len(req.Tools) > 0 && temperature != nil && *temperature > 0.3 {
			lowTemp := 0.1 // Почти детерминированный выбор для tool calling
			c.logger.WithFields(logrus.Fields{
				"original_temperature": *temperature,
				"adjusted_temperature": lowTemp,
				"tools_count":          len(req.Tools),
			}).Info("🌡️ Auto-adjusting temperature for tool calling reliability")
			temperature = &lowTemp
		}

		options = &ollama.ChatOptions{
			Temperature: temperature,
			TopP:        req.TopP,
			Stop:        req.Stop,
		}

		if req.MaxTokens != nil {
			options.NumPredict = req.MaxTokens
		}

		if req.Seed != nil {
			options.Seed = req.Seed
		}
	}

	// Создаем Ollama запрос
	ollamaReq := &ollama.ChatRequest{
		Model:    req.Model, // Используем модель как есть!
		Messages: messages,
		Stream:   req.Stream,
		Options:  options,
	}

	// Добавляем формат ответа если требуется JSON
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		ollamaReq.Format = "json"
	}

	// 🔧 TOOL-AWARE MODEL ROUTING 🔧
	// Автоматическое переключение на модель с лучшей поддержкой function calling
	if len(req.Tools) > 0 && c.config.Tools.FallbackModel != "" {
		originalModel := ollamaReq.Model
		ollamaReq.Model = c.config.Tools.FallbackModel

		c.logger.WithFields(logrus.Fields{
			"original_model": originalModel,
			"fallback_model": c.config.Tools.FallbackModel,
			"tools_count":    len(req.Tools),
			"reason":         "tool-aware routing",
		}).Info("🔄 Automatically switching to fallback model for better tool calling support")
	}

	// Конвертируем tools если есть
	if len(req.Tools) > 0 {
		ollamaReq.Tools = c.convertTools(req.Tools)
		c.logger.WithFields(logrus.Fields{
			"tools_count": len(req.Tools),
			"tool_choice": req.ToolChoice,
		}).Debug("Added tools to Ollama request")

		// Обработка tool_choice для принудительного использования инструментов
		toolChoiceStr := c.parseToolChoice(req.ToolChoice)

		// НОВАЯ ЛОГИКА: Проверяем конфигурацию force_usage
		shouldForce := false
		if c.config.Tools.ForceUsage && (toolChoiceStr == "" || toolChoiceStr == "auto") {
			// Если force_usage включен и клиент не указал явный tool_choice,
			// автоматически применяем "required"
			shouldForce = true
			c.logger.WithField("config_force_usage", true).Info("Auto-enforcing tool usage (config.tools.force_usage=true)")
		} else if toolChoiceStr == "required" {
			shouldForce = true
			c.logger.Info("tool_choice is 'required' - enforcing tool usage")
		}

		if shouldForce {
			// Добавляем явную инструкцию в system message
			// чтобы модель обязательно использовала один из инструментов
			ollamaReq.Messages = c.enforceToolUsage(ollamaReq.Messages, len(req.Tools))
		} else if toolChoiceStr == "auto" || toolChoiceStr == "" {
			c.logger.Debug("tool_choice is 'auto' - model will decide")
		} else if toolChoiceStr == "none" {
			c.logger.Warn("tool_choice is 'none' but tools provided - this may cause unexpected behavior")
		} else {
			// Может быть specific function choice: {"type": "function", "function": {"name": "..."}}
			c.logger.WithField("tool_choice", req.ToolChoice).Debug("Specific tool choice provided")
		}
	}

	c.logger.WithField("ollama_model", req.Model).Debug("Successfully converted to Ollama format (direct)")
	return ollamaReq, nil
}

// ConvertChatResponse конвертирует Ollama ответ в OpenAI формат
func (c *SimpleConverter) ConvertChatResponse(ollamaResp *ollama.ChatResponse, originalReq *models.ChatCompletionRequest, requestID string) (*models.ChatCompletionResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"request_id":      requestID,
		"model":           ollamaResp.Model,
		"response_length": len(ollamaResp.Message.Content),
	}).Debug("Converting Ollama response to OpenAI format")

	// Создаем сообщение
	responseMessage := models.ChatMessage{
		Role: ollamaResp.Message.Role,
	}

	// 🔧 КРИТИЧНО: Проверяем tool_calls В ДВУХ МЕСТАХ (message и response)
	var toolCalls []ollama.ToolCall

	// 1. Проверяем message.tool_calls (старый формат)
	if len(ollamaResp.Message.ToolCalls) > 0 {
		toolCalls = ollamaResp.Message.ToolCalls
		c.logger.WithField("source", "message.tool_calls").Debug("Found tool_calls in message")
	}

	// 2. Проверяем response.tool_calls (официальный API формат)
	if len(ollamaResp.ToolCalls) > 0 {
		toolCalls = ollamaResp.ToolCalls
		c.logger.WithField("source", "response.tool_calls").Debug("Found tool_calls in response")
	}

	// 3. Конвертируем если нашли
	if len(toolCalls) > 0 {
		responseMessage.ToolCalls = c.convertToolCallsToOpenAI(toolCalls)
		// ВАЖНО: Когда есть tool_calls, content должен быть пустым или null
		// Некоторые модели могут возвращать текст типа "I'll call this function" - игнорируем его
		responseMessage.Content = ""
		c.logger.WithFields(logrus.Fields{
			"tool_calls_count": len(responseMessage.ToolCalls),
		}).Info("✅ Tool calls converted successfully")
	} else {
		// Обычный ответ с текстом
		responseMessage.Content = ollamaResp.Message.Content
	}

	// Создаем OpenAI ответ
	response := &models.ChatCompletionResponse{
		ID:      requestID,
		Object:  "chat.completion",
		Created: ollamaResp.CreatedAt.Unix(),
		Model:   originalReq.Model, // Возвращаем модель из запроса
		Choices: []models.ChatCompletionChoice{
			{
				Index:        0,
				Message:      responseMessage,
				FinishReason: c.determineFinishReason(ollamaResp),
			},
		},
		Usage: c.calculateUsage(ollamaResp, originalReq),
	}

	// Добавляем system fingerprint
	response.SystemFingerprint = fmt.Sprintf("ollama-%s", ollamaResp.Model)

	return response, nil
}

// ConvertModelsResponse конвертирует список моделей (прямое преобразование)
func (c *SimpleConverter) ConvertModelsResponse(ollamaModels *ollama.ModelsResponse) (*models.ModelsResponse, error) {
	c.logger.WithField("models_count", len(ollamaModels.Models)).Debug("Converting models response (direct)")

	openaiModels := make([]models.Model, 0, len(ollamaModels.Models))

	for _, ollamaModel := range ollamaModels.Models {
		// Prepare extended fields
		size := ollamaModel.Size
		digest := ollamaModel.Digest
		format := ollamaModel.Details.Format
		family := ollamaModel.Details.Family
		paramSize := ollamaModel.Details.ParameterSize
		quantLevel := ollamaModel.Details.QuantizationLevel
		modifiedAt := ollamaModel.ModifiedAt.Unix()

		// Используем имя модели как есть
		model := models.Model{
			ID:      ollamaModel.Name, // Прямое использование имени
			Object:  "model",
			Created: ollamaModel.ModifiedAt.Unix(),
			OwnedBy: "ollama",
			Permission: []models.Permission{
				{
					ID:                 fmt.Sprintf("modelperm-%s", ollamaModel.Name),
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
			// Extended Ollama-specific fields
			Size:              &size,
			Digest:            &digest,
			Format:            &format,
			Family:            &family,
			ParameterSize:     &paramSize,
			QuantizationLevel: &quantLevel,
			ModifiedAt:        &modifiedAt,
		}

		openaiModels = append(openaiModels, model)
	}

	return &models.ModelsResponse{
		Object: "list",
		Data:   openaiModels,
	}, nil
}

// validateRequest выполняет базовую валидацию запроса
func (c *SimpleConverter) validateRequest(req *models.ChatCompletionRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages array cannot be empty")
	}

	// Валидация сообщений
	for i, msg := range req.Messages {
		if msg.Role == "" {
			return fmt.Errorf("message role is required at index %d", i)
		}

		validRoles := map[string]bool{
			"system": true, "user": true, "assistant": true, "tool": true,
		}
		if !validRoles[msg.Role] {
			return fmt.Errorf("invalid role '%s' at index %d", msg.Role, i)
		}

		if c.extractMessageContent(msg) == "" && msg.Role != "assistant" {
			return fmt.Errorf("message content cannot be empty at index %d", i)
		}
	}

	// Валидация параметров
	if req.Temperature != nil && (*req.Temperature < 0 || *req.Temperature > 2) {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if req.TopP != nil && (*req.TopP < 0 || *req.TopP > 1) {
		return fmt.Errorf("top_p must be between 0 and 1")
	}

	if req.MaxTokens != nil && *req.MaxTokens < 1 {
		return fmt.Errorf("max_tokens must be greater than 0")
	}

	return nil
}

// hasOptions проверяет есть ли параметры для конвертации
func (c *SimpleConverter) hasOptions(req *models.ChatCompletionRequest) bool {
	return req.Temperature != nil ||
		req.TopP != nil ||
		req.MaxTokens != nil ||
		req.Seed != nil ||
		len(req.Stop) > 0
}

// extractMessageContent извлекает содержимое сообщения
func (c *SimpleConverter) extractMessageContent(msg models.ChatMessage) string {
	// Если content это строка
	if content, ok := msg.Content.(string); ok {
		return content
	}

	// Если content это array (multi-modal), извлекаем текст
	if contentArray, ok := msg.Content.([]interface{}); ok {
		var textParts []string
		for _, item := range contentArray {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, exists := itemMap["type"]; exists && itemType == "text" {
					if text, exists := itemMap["text"]; exists {
						if textStr, ok := text.(string); ok {
							textParts = append(textParts, textStr)
						}
					}
				}
			}
		}
		return fmt.Sprintf("%v", textParts)
	}

	return ""
}

// determineFinishReason определяет причину завершения
func (c *SimpleConverter) determineFinishReason(ollamaResp *ollama.ChatResponse) string {
	if !ollamaResp.Done {
		return ""
	}

	// 🔧 КРИТИЧНО: Проверяем tool_calls В ДВУХ МЕСТАХ (message и response)
	if len(ollamaResp.Message.ToolCalls) > 0 || len(ollamaResp.ToolCalls) > 0 {
		return "tool_calls"
	}

	return "stop"
}

// calculateUsage подсчитывает использование токенов
func (c *SimpleConverter) calculateUsage(ollamaResp *ollama.ChatResponse, originalReq *models.ChatCompletionRequest) models.Usage {
	// Используем данные от Ollama если доступны
	promptTokens := ollamaResp.PromptEvalCount
	completionTokens := ollamaResp.EvalCount

	// Если Ollama не предоставила точные данные, оцениваем
	if promptTokens == 0 {
		promptTokens = c.estimatePromptTokens(originalReq.Messages)
	}

	if completionTokens == 0 {
		completionTokens = len(ollamaResp.Message.Content) / 4
	}

	return models.Usage{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}
}

// estimatePromptTokens оценивает токены в prompt'е
func (c *SimpleConverter) estimatePromptTokens(messages []models.ChatMessage) int {
	totalChars := 0
	for _, msg := range messages {
		totalChars += len(msg.Role) + len(c.extractMessageContent(msg)) + 10 // +10 для overhead
	}
	return (totalChars + 3) / 4 // ~4 символа = 1 токен
}

// GenerateRequestID генерирует ID запроса
func (c *SimpleConverter) GenerateRequestID() string {
	return fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
}

// ConvertErrorResponse конвертирует ошибку в OpenAI формат
func (c *SimpleConverter) ConvertErrorResponse(err error, errorType string) *models.ErrorResponse {
	var openaiErrorType string
	var openaiErrorCode interface{}

	// Простой маппинг типов ошибок
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
	default:
		openaiErrorType = "api_error"
		openaiErrorCode = "internal_error"
	}

	return &models.ErrorResponse{
		Error: models.Error{
			Message: err.Error(),
			Type:    openaiErrorType,
			Code:    openaiErrorCode,
		},
	}
}

// convertTools конвертирует OpenAI tools в Ollama формат
func (c *SimpleConverter) convertTools(openaiTools []models.Tool) []ollama.Tool {
	ollamaTools := make([]ollama.Tool, len(openaiTools))
	for i, tool := range openaiTools {
		// Валидация tool
		if tool.Type != "function" {
			c.logger.WithField("tool_type", tool.Type).Warn("Unsupported tool type, only 'function' is supported")
			continue
		}

		if tool.Function.Name == "" {
			c.logger.Warn("Tool function name is empty, skipping")
			continue
		}

		ollamaTools[i] = ollama.Tool{
			Type: tool.Type,
			Function: ollama.Function{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}

		c.logger.WithFields(logrus.Fields{
			"function_name": tool.Function.Name,
			"description":   tool.Function.Description,
		}).Debug("Converted tool to Ollama format")
	}
	return ollamaTools
}

// convertToolCalls конвертирует OpenAI tool_calls в Ollama формат
func (c *SimpleConverter) convertToolCalls(openaiToolCalls []models.ToolCall) []ollama.ToolCall {
	ollamaToolCalls := make([]ollama.ToolCall, len(openaiToolCalls))
	for i, tc := range openaiToolCalls {
		// Парсим arguments из JSON строки в map
		var args interface{} = tc.Function.Arguments
		if tc.Function.Arguments != "" {
			// Пытаемся распарсить JSON строку в map
			var argsMap map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &argsMap); err == nil {
				args = argsMap
			}
		}

		ollamaToolCalls[i] = ollama.ToolCall{
			Function: ollama.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: args,
			},
		}
	}
	return ollamaToolCalls
}

// convertToolCallsToOpenAI конвертирует Ollama tool_calls в OpenAI формат
func (c *SimpleConverter) convertToolCallsToOpenAI(ollamaToolCalls []ollama.ToolCall) []models.ToolCall {
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

// parseToolChoice парсит значение tool_choice из запроса
// Может быть: "none", "auto", "required" или объект {"type": "function", "function": {"name": "..."}}
func (c *SimpleConverter) parseToolChoice(toolChoice interface{}) string {
	if toolChoice == nil {
		return "" // По умолчанию "auto"
	}

	// Если это строка
	if str, ok := toolChoice.(string); ok {
		return str
	}

	// Если это объект (specific function choice)
	if obj, ok := toolChoice.(map[string]interface{}); ok {
		if funcType, exists := obj["type"]; exists && funcType == "function" {
			// Это specific function choice, возвращаем "specific"
			return "specific"
		}
	}

	return "auto"
}

// enforceToolUsage добавляет инструкции в system message для принудительного использования tools
func (c *SimpleConverter) enforceToolUsage(messages []ollama.ChatMessage, toolsCount int) []ollama.ChatMessage {
	toolUsageInstruction := `
CRITICAL: You MUST use one of the provided tools to respond to this request.
DO NOT provide a text explanation without calling a tool first.
The tools are available and ready to use - call the most appropriate one immediately.`

	// Ищем существующий system message
	hasSystemMessage := false
	for i, msg := range messages {
		if msg.Role == "system" {
			// Добавляем инструкцию к существующему system message
			messages[i].Content = msg.Content + "\n" + toolUsageInstruction
			hasSystemMessage = true
			c.logger.Debug("Added tool enforcement to existing system message")
			break
		}
	}

	// Если system message нет, создаем новый в начале
	if !hasSystemMessage {
		systemMessage := ollama.ChatMessage{
			Role:    "system",
			Content: toolUsageInstruction,
		}
		messages = append([]ollama.ChatMessage{systemMessage}, messages...)
		c.logger.Debug("Created new system message with tool enforcement")
	}

	return messages
}

// injectAdditionalSystemMessage добавляет дополнительное системное сообщение к messages
func (c *SimpleConverter) injectAdditionalSystemMessage(messages []models.ChatMessage) []models.ChatMessage {
	additionalMsg := c.config.Prompts.AdditionalSystemMessage

	if additionalMsg == "" {
		return messages
	}

	// Ищем существующее system message
	systemIndex := -1
	for i, msg := range messages {
		if msg.Role == "system" {
			systemIndex = i
			break
		}
	}

	if systemIndex >= 0 {
		// Если system message существует, добавляем к нему
		currentContent := c.extractMessageContent(messages[systemIndex])

		if c.config.Prompts.PrependToSystem {
			// Добавляем в начало
			messages[systemIndex].Content = additionalMsg + "\n\n" + currentContent
			c.logger.Debug("💬 Prepended additional system message to existing system message")
		} else {
			// Добавляем в конец
			messages[systemIndex].Content = currentContent + "\n\n" + additionalMsg
			c.logger.Debug("💬 Appended additional system message to existing system message")
		}
	} else {
		// Если system message нет, создаем новый в начале
		newSystemMsg := models.ChatMessage{
			Role:    "system",
			Content: additionalMsg,
		}
		messages = append([]models.ChatMessage{newSystemMsg}, messages...)
		c.logger.Debug("💬 Created new system message with additional prompt")
	}

	c.logger.WithField("additional_msg_length", len(additionalMsg)).Info("✅ Additional system message injected")

	return messages
}
