// Package converter provides request/response conversion between OpenAI and Ollama APIs
package converter

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/models"
)

// ChatConverter обрабатывает конвертацию chat completion запросов и ответов
type ChatConverter struct {
	logger       *logrus.Logger
	modelManager ModelManager
}

// NewChatConverter создает новый конвертер для chat completion
func NewChatConverter(logger *logrus.Logger, modelManager ModelManager) *ChatConverter {
	return &ChatConverter{
		logger:       logger,
		modelManager: modelManager,
	}
}

// ConvertRequest конвертирует OpenAI chat completion запрос в Ollama формат
func (c *ChatConverter) ConvertRequest(req *models.ChatCompletionRequest) (*ollama.ChatRequest, error) {
	startTime := time.Now()

	c.logger.WithFields(logrus.Fields{
		"original_model": req.Model,
		"messages_count": len(req.Messages),
		"stream":         req.Stream,
		"temperature":    req.Temperature,
		"max_tokens":     req.MaxTokens,
	}).Debug("Converting OpenAI chat completion request to Ollama format")

	// 1. Маппинг модели
	ollamaModel, err := c.modelManager.MapOpenAIToOllama(req.Model)
	if err != nil {
		c.logger.WithError(err).WithField("model", req.Model).Warn("Failed to map model, using original name")
		ollamaModel = req.Model // Fallback к оригинальному имени
	}

	// 2. Конвертация сообщений
	messages, err := c.convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// 3. Конвертация параметров
	options, err := c.convertOptions(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert options: %w", err)
	}

	// 4. Создание Ollama запроса
	ollamaReq := &ollama.ChatRequest{
		Model:    ollamaModel,
		Messages: messages,
		Stream:   req.Stream,
		Options:  options,
	}

	// Добавляем формат ответа если требуется JSON
	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		ollamaReq.Format = "json"
	}

	// Отключаем thinking режим для gpt-oss моделей
	if c.isThinkingModel(ollamaModel) {
		thinkDisabled := false
		ollamaReq.Think = &thinkDisabled

		// Также добавляем системное сообщение для усиления эффекта
		if len(messages) == 0 || messages[0].Role != "system" {
			systemMsg := ollama.ChatMessage{
				Role:    "system",
				Content: "You must provide direct answers only. Do not show your thinking process. Respond with final answers immediately.",
			}
			messages = append([]ollama.ChatMessage{systemMsg}, messages...)
			ollamaReq.Messages = messages
		}

		c.logger.WithField("model", ollamaModel).Debug("Disabled thinking mode for gpt-oss model")
	}

	duration := time.Since(startTime)
	c.logger.WithFields(logrus.Fields{
		"mapped_model":    ollamaModel,
		"conversion_time": duration,
		"ollama_options":  options != nil,
	}).Debug("Successfully converted OpenAI request to Ollama format")

	return ollamaReq, nil
}

// convertMessages конвертирует OpenAI сообщения в Ollama формат
func (c *ChatConverter) convertMessages(openaiMessages []models.ChatMessage) ([]ollama.ChatMessage, error) {
	messages := make([]ollama.ChatMessage, 0, len(openaiMessages))

	for i, msg := range openaiMessages {
		// Проверяем поддерживаемые роли
		if !c.isValidRole(msg.Role) {
			return nil, fmt.Errorf("unsupported message role: %s at index %d", msg.Role, i)
		}

		// Конвертируем содержимое сообщения
		content := c.extractContent(msg)
		if content == "" && msg.Role != "assistant" {
			return nil, fmt.Errorf("empty message content at index %d", i)
		}

		// Обрабатываем tool calls (пока не поддерживается в Ollama)
		if len(msg.ToolCalls) > 0 {
			c.logger.WithField("message_index", i).Warn("Tool calls are not supported in Ollama, ignoring")
		}

		// Обрабатываем function calls (deprecated)
		if msg.FunctionCall != nil {
			c.logger.WithField("message_index", i).Warn("Function calls are not supported in Ollama, ignoring")
		}

		ollamaMsg := ollama.ChatMessage{
			Role:    msg.Role,
			Content: content,
		}

		messages = append(messages, ollamaMsg)
	}

	return messages, nil
}

// convertOptions конвертирует OpenAI параметры в Ollama опции
func (c *ChatConverter) convertOptions(req *models.ChatCompletionRequest) (*ollama.ChatOptions, error) {
	// Если нет параметров, возвращаем nil
	if !c.hasOptions(req) {
		return nil, nil
	}

	options := &ollama.ChatOptions{}

	// Temperature (прямое соответствие)
	if req.Temperature != nil {
		// Валидация диапазона
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return nil, fmt.Errorf("temperature must be between 0 and 2, got: %f", *req.Temperature)
		}
		options.Temperature = req.Temperature
	}

	// Top-P (прямое соответствие)
	if req.TopP != nil {
		// Валидация диапазона
		if *req.TopP < 0 || *req.TopP > 1 {
			return nil, fmt.Errorf("top_p must be between 0 and 1, got: %f", *req.TopP)
		}
		options.TopP = req.TopP
	}

	// Max tokens -> num_predict
	if req.MaxTokens != nil {
		if *req.MaxTokens < 1 {
			return nil, fmt.Errorf("max_tokens must be greater than 0, got: %d", *req.MaxTokens)
		}
		options.NumPredict = req.MaxTokens
	}

	// Stop sequences (прямое соответствие)
	if len(req.Stop) > 0 {
		options.Stop = req.Stop
	}

	// Seed (прямое соответствие)
	if req.Seed != nil {
		options.Seed = req.Seed
	}

	// Параметры которые не поддерживаются в Ollama
	if req.PresencePenalty != nil {
		c.logger.Warn("presence_penalty is not supported in Ollama, ignoring")
	}
	if req.FrequencyPenalty != nil {
		c.logger.Warn("frequency_penalty is not supported in Ollama, ignoring")
	}
	if len(req.LogitBias) > 0 {
		c.logger.Warn("logit_bias is not supported in Ollama, ignoring")
	}
	if req.N != nil && *req.N > 1 {
		c.logger.Warn("multiple completions (n > 1) are not supported in Ollama, ignoring")
	}

	return options, nil
}

// hasOptions проверяет есть ли опции для конвертации
func (c *ChatConverter) hasOptions(req *models.ChatCompletionRequest) bool {
	return req.Temperature != nil ||
		req.TopP != nil ||
		req.MaxTokens != nil ||
		req.Seed != nil ||
		len(req.Stop) > 0
}

// isValidRole проверяет поддерживаемые роли в Ollama
func (c *ChatConverter) isValidRole(role string) bool {
	validRoles := map[string]bool{
		"system":    true,
		"user":      true,
		"assistant": true,
	}
	return validRoles[role]
}

// isThinkingModel проверяет является ли модель thinking моделью
func (c *ChatConverter) isThinkingModel(modelName string) bool {
	thinkingModels := []string{
		"gpt-oss",
		"deepseek-r1",
		"qwq",
	}

	modelLower := strings.ToLower(modelName)
	for _, pattern := range thinkingModels {
		if strings.Contains(modelLower, pattern) {
			return true
		}
	}

	return false
}

// extractContent извлекает текстовое содержимое из сообщения
func (c *ChatConverter) extractContent(msg models.ChatMessage) string {
	// Если content это строка, возвращаем её
	if content, ok := msg.Content.(string); ok {
		return content
	}

	// Если content это array (multi-modal), извлекаем текст
	if contentArray, ok := msg.Content.([]interface{}); ok {
		return c.extractTextFromArray(contentArray)
	}

	// Если content не задан, но есть tool calls, формируем текст
	if msg.Content == nil && len(msg.ToolCalls) > 0 {
		return c.formatToolCallsAsText(msg.ToolCalls)
	}

	return ""
}

// extractTextFromArray извлекает текст из массива content (multi-modal)
func (c *ChatConverter) extractTextFromArray(contentArray []interface{}) string {
	var textParts []string

	for _, item := range contentArray {
		if itemMap, ok := item.(map[string]interface{}); ok {
			if itemType, exists := itemMap["type"]; exists && itemType == "text" {
				if text, exists := itemMap["text"]; exists {
					if textStr, ok := text.(string); ok {
						textParts = append(textParts, textStr)
					}
				}
			} else if itemType == "image_url" {
				// Для изображений добавляем placeholder
				textParts = append(textParts, "[IMAGE]")
				c.logger.Warn("Image content is not supported in Ollama, replacing with placeholder")
			}
		}
	}

	return fmt.Sprintf("%s", textParts)
}

// formatToolCallsAsText форматирует tool calls как текст
func (c *ChatConverter) formatToolCallsAsText(toolCalls []models.ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	var parts []string
	for _, call := range toolCalls {
		part := fmt.Sprintf("Called function %s with arguments: %s", call.Function.Name, call.Function.Arguments)
		parts = append(parts, part)
	}

	return fmt.Sprintf("Tool calls: %s", parts)
}

