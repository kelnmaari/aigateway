// Package converter provides unified conversion interface between OpenAI and Ollama APIs
package converter

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/models"
)

// Converter объединяет все конвертеры в единый интерфейс
type Converter struct {
	logger            *logrus.Logger
	config            *config.Config
	modelManager      ModelManager
	chatConverter     *ChatConverter
	responseConverter *ResponseConverter
}

// NewConverter создает новый объединенный конвертер
func NewConverter(cfg *config.Config, logger *logrus.Logger) *Converter {
	// Создаем model manager
	modelManager := NewDefaultModelManager(cfg, logger)

	// Создаем специализированные конвертеры
	chatConverter := NewChatConverter(logger, modelManager)
	responseConverter := NewResponseConverter(logger, modelManager)

	return &Converter{
		logger:            logger,
		config:            cfg,
		modelManager:      modelManager,
		chatConverter:     chatConverter,
		responseConverter: responseConverter,
	}
}

// ConvertChatCompletionRequest конвертирует OpenAI chat completion запрос
func (c *Converter) ConvertChatCompletionRequest(req *models.ChatCompletionRequest) (*ollama.ChatRequest, error) {
	c.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"messages_count": len(req.Messages),
		"stream":         req.Stream,
	}).Debug("Converting OpenAI chat completion request")

	// Валидация запроса
	if err := c.validateChatRequest(req); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Конвертация через chat converter
	return c.chatConverter.ConvertRequest(req)
}

// ConvertChatCompletionResponse конвертирует Ollama ответ в OpenAI формат
func (c *Converter) ConvertChatCompletionResponse(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
	requestID string,
) (*models.ChatCompletionResponse, error) {
	c.logger.WithFields(logrus.Fields{
		"request_id":     requestID,
		"ollama_model":   ollamaResp.Model,
		"original_model": originalReq.Model,
	}).Debug("Converting Ollama chat response to OpenAI format")

	return c.responseConverter.ConvertChatResponse(ollamaResp, originalReq, requestID)
}

// ConvertStreamChunk конвертирует streaming chunk
func (c *Converter) ConvertStreamChunk(
	ollamaResp *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
	requestID string,
	isLast bool,
) (*models.ChatCompletionChunk, error) {
	return c.responseConverter.ConvertStreamChunk(ollamaResp, originalReq, requestID, isLast)
}

// ConvertModelsResponse конвертирует список моделей
func (c *Converter) ConvertModelsResponse(ollamaModels *ollama.ModelsResponse) (*models.ModelsResponse, error) {
	c.logger.WithField("models_count", len(ollamaModels.Models)).Debug("Converting models response")

	return c.responseConverter.ConvertModelsResponse(ollamaModels)
}

// ConvertErrorResponse конвертирует ошибку в OpenAI формат
func (c *Converter) ConvertErrorResponse(ollamaError error, errorType string, statusCode int) *models.ErrorResponse {
	return c.responseConverter.ConvertErrorResponse(ollamaError, errorType, statusCode)
}

// GetModelManager возвращает model manager
func (c *Converter) GetModelManager() ModelManager {
	return c.modelManager
}

// ValidateModel проверяет поддержку модели
func (c *Converter) ValidateModel(model string) error {
	if !c.modelManager.IsModelSupported(model, models.APITypeOpenAI) {
		// Пытаемся найти похожие модели
		supported := c.modelManager.ListSupportedModels(models.APITypeOpenAI)
		suggestion := c.findSimilarModel(model, supported)

		if suggestion != "" {
			return fmt.Errorf("model '%s' is not supported, did you mean '%s'?", model, suggestion)
		}

		return fmt.Errorf("model '%s' is not supported", model)
	}

	return nil
}

// GetSupportedModels возвращает список поддерживаемых моделей
func (c *Converter) GetSupportedModels() []string {
	return c.modelManager.ListSupportedModels(models.APITypeOpenAI)
}

// GetModelInfo возвращает информацию о модели
func (c *Converter) GetModelInfo(model string) map[string]interface{} {
	if mm, ok := c.modelManager.(*DefaultModelManager); ok {
		return mm.GetModelInfo(model)
	}

	return map[string]interface{}{
		"model": model,
		"error": "model info not available",
	}
}

// RefreshModelMappings обновляет маппинги моделей
func (c *Converter) RefreshModelMappings() {
	if mm, ok := c.modelManager.(*DefaultModelManager); ok {
		mm.RefreshMappings()
		c.logger.Info("Model mappings refreshed")
	}
}

// validateChatRequest выполняет валидацию chat completion запроса
func (c *Converter) validateChatRequest(req *models.ChatCompletionRequest) error {
	// Проверка обязательных полей
	if req.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(req.Messages) == 0 {
		return fmt.Errorf("messages array cannot be empty")
	}

	// Проверка моделей
	if err := c.ValidateModel(req.Model); err != nil {
		return err
	}

	// Проверка сообщений
	for i, msg := range req.Messages {
		if err := c.validateMessage(msg, i); err != nil {
			return err
		}
	}

	// Проверка параметров
	if err := c.validateParameters(req); err != nil {
		return err
	}

	return nil
}

// validateMessage валидирует отдельное сообщение
func (c *Converter) validateMessage(msg models.ChatMessage, index int) error {
	// Проверка роли
	validRoles := map[string]bool{
		"system":    true,
		"user":      true,
		"assistant": true,
		"tool":      true,
	}

	if !validRoles[msg.Role] {
		return fmt.Errorf("invalid role '%s' at message index %d", msg.Role, index)
	}

	// Проверка содержимого
	content := ""
	if contentStr, ok := msg.Content.(string); ok {
		content = contentStr
	}

	if content == "" && msg.Role != "assistant" && len(msg.ToolCalls) == 0 {
		return fmt.Errorf("message content cannot be empty at index %d", index)
	}

	// Валидация tool calls
	if len(msg.ToolCalls) > 0 {
		for j, toolCall := range msg.ToolCalls {
			if toolCall.Type != "function" {
				return fmt.Errorf("unsupported tool call type '%s' at message %d, tool call %d", toolCall.Type, index, j)
			}
			if toolCall.Function.Name == "" {
				return fmt.Errorf("function name is required at message %d, tool call %d", index, j)
			}
		}
	}

	return nil
}

// validateParameters валидирует параметры запроса
func (c *Converter) validateParameters(req *models.ChatCompletionRequest) error {
	// Валидация temperature
	if req.Temperature != nil {
		if *req.Temperature < 0 || *req.Temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2, got %f", *req.Temperature)
		}
	}

	// Валидация top_p
	if req.TopP != nil {
		if *req.TopP < 0 || *req.TopP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1, got %f", *req.TopP)
		}
	}

	// Валидация max_tokens
	if req.MaxTokens != nil {
		if *req.MaxTokens < 1 {
			return fmt.Errorf("max_tokens must be greater than 0, got %d", *req.MaxTokens)
		}
		if *req.MaxTokens > 128000 { // Разумный лимит
			return fmt.Errorf("max_tokens is too large, maximum allowed is 128000, got %d", *req.MaxTokens)
		}
	}

	// Валидация n (количество вариантов ответа)
	if req.N != nil && *req.N > 1 {
		c.logger.Warn("Multiple completions (n > 1) are not supported by Ollama, only first completion will be returned")
	}

	// Валидация presence_penalty и frequency_penalty
	if req.PresencePenalty != nil {
		if *req.PresencePenalty < -2 || *req.PresencePenalty > 2 {
			return fmt.Errorf("presence_penalty must be between -2 and 2, got %f", *req.PresencePenalty)
		}
	}

	if req.FrequencyPenalty != nil {
		if *req.FrequencyPenalty < -2 || *req.FrequencyPenalty > 2 {
			return fmt.Errorf("frequency_penalty must be between -2 and 2, got %f", *req.FrequencyPenalty)
		}
	}

	return nil
}

// findSimilarModel ищет похожую модель в списке поддерживаемых
func (c *Converter) findSimilarModel(target string, supported []string) string {
	target = models.NormalizeModelName(target)
	targetFamily := models.ExtractModelFamily(target)

	// Ищем точное совпадение по семейству
	for _, model := range supported {
		if models.ExtractModelFamily(model) == targetFamily {
			return model
		}
	}

	// Ищем частичное совпадение
	for _, model := range supported {
		normalized := models.NormalizeModelName(model)
		if len(target) > 3 && len(normalized) > 3 {
			// Простая проверка на вхождение подстроки
			if containsIgnoreCase(normalized, target) || containsIgnoreCase(target, normalized) {
				return model
			}
		}
	}

	return ""
}

// containsIgnoreCase проверяет содержит ли строка подстроку (игнорируя регистр)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}

// GenerateRequestID генерирует уникальный ID для запроса
func (c *Converter) GenerateRequestID() string {
	return models.GenerateRequestID("req")
}

// GetConversionStats возвращает статистику конвертации
func (c *Converter) GetConversionStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Информация о моделях
	stats["supported_openai_models"] = len(c.modelManager.ListSupportedModels(models.APITypeOpenAI))
	stats["supported_ollama_models"] = len(c.modelManager.ListSupportedModels(models.APITypeOllama))

	// Информация о маппингах
	if mm, ok := c.modelManager.(*DefaultModelManager); ok {
		mm.mutex.RLock()
		stats["total_mappings"] = len(mm.mappings)
		mm.mutex.RUnlock()
	}

	// Временная метка
	stats["last_updated"] = time.Now().Unix()

	return stats
}

