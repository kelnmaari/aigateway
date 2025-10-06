// Package converter provides streaming conversion between OpenAI and Ollama formats
package converter

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
)

// StreamConverter обрабатывает конвертацию streaming responses
type StreamConverter struct {
	logger       *logrus.Logger
	config       *config.Config
	hadToolCalls bool // 🔧 Запоминаем что в запросе были tool_calls
}

// NewStreamConverter создает новый stream converter
func NewStreamConverter(cfg *config.Config, logger *logrus.Logger) *StreamConverter {
	return &StreamConverter{
		logger:       logger,
		config:       cfg,
		hadToolCalls: false,
	}
}

// Reset сбрасывает состояние конвертера для нового запроса
func (s *StreamConverter) Reset() {
	s.hadToolCalls = false
}

// ConvertOllamaChunkToOpenAI конвертирует Ollama chunk в OpenAI streaming format
func (s *StreamConverter) ConvertOllamaChunkToOpenAI(
	ollamaChunk *ollama.ChatResponse,
	originalReq *models.ChatCompletionRequest,
	requestID string,
) (*models.ChatCompletionChunk, error) {

	s.logger.WithFields(logrus.Fields{
		"request_id":    requestID,
		"chunk_done":    ollamaChunk.Done,
		"chunk_content": len(ollamaChunk.Message.Content),
		"model":         ollamaChunk.Model,
	}).Debug("Converting Ollama chunk to OpenAI format")

	// Создаем OpenAI streaming chunk
	chunk := &models.ChatCompletionChunk{
		ID:      requestID,
		Object:  "chat.completion.chunk",
		Created: ollamaChunk.CreatedAt.Unix(),
		Model:   originalReq.Model, // Используем модель из запроса
	}

	// Обработка content с поддержкой thinking для gpt-oss моделей
	content := ollamaChunk.Message.Content

	// Для gpt-oss моделей: thinking является частью генерации
	// Отправляем thinking как content для отображения процесса рассуждения
	if content == "" && ollamaChunk.Message.Thinking != "" {
		content = ollamaChunk.Message.Thinking

		s.logger.WithFields(logrus.Fields{
			"request_id":    requestID,
			"model":         ollamaChunk.Model,
			"thinking_len":  len(content),
			"thinking_text": content,
		}).Debug("Using thinking as content for gpt-oss model")
	}

	delta := models.ChatMessage{
		Role:    ollamaChunk.Message.Role,
		Content: content,
	}

	// 🔧 КРИТИЧНО: Проверяем tool_calls В ДВУХ МЕСТАХ (message и response)
	var toolCalls []ollama.ToolCall

	// 1. Проверяем message.tool_calls (старый формат)
	if len(ollamaChunk.Message.ToolCalls) > 0 {
		toolCalls = ollamaChunk.Message.ToolCalls
		s.logger.WithField("source", "message.tool_calls").Debug("Found tool_calls in streaming message")
	}

	// 2. Проверяем response.tool_calls (официальный API формат)
	if len(ollamaChunk.ToolCalls) > 0 {
		toolCalls = ollamaChunk.ToolCalls
		s.logger.WithField("source", "response.tool_calls").Debug("Found tool_calls in streaming response")
	}

	// 3. Конвертируем если нашли
	if len(toolCalls) > 0 {
		delta.ToolCalls = s.convertToolCallsToOpenAI(toolCalls)
		// ВАЖНО: Очищаем content если есть tool_calls
		delta.Content = ""
		s.hadToolCalls = true // 🔧 КРИТИЧНО: Запоминаем что были tool_calls!
		s.logger.WithFields(logrus.Fields{
			"tool_calls_count": len(delta.ToolCalls),
		}).Info("✅ Tool calls converted in STREAMING chunk (StreamConverter)!")
	}

	// Определяем finish_reason для последнего chunk'а
	var finishReason *string
	if ollamaChunk.Done {
		// 🔧 КРИТИЧНО: Если В ЛЮБОЙ МОМЕНТ БЫЛИ tool_calls, finish_reason = "tool_calls"
		reason := "stop"
		if s.hadToolCalls {
			reason = "tool_calls"
			s.logger.Info("🎯 Setting finish_reason=tool_calls (had tool calls in stream)")
		}
		finishReason = &reason
	}

	// Создаем choice для chunk'а
	choice := models.ChatCompletionChunkChoice{
		Index:        0,
		Delta:        delta,
		FinishReason: finishReason,
	}

	chunk.Choices = []models.ChatCompletionChunkChoice{choice}

	// Добавляем system fingerprint
	chunk.SystemFingerprint = fmt.Sprintf("ollama-%s", ollamaChunk.Model)

	return chunk, nil
}

// ConvertToServerSentEvent конвертирует chunk в SSE формат
func (s *StreamConverter) ConvertToServerSentEvent(chunk *models.ChatCompletionChunk) (string, error) {
	// Конвертируем chunk в JSON
	jsonData, err := json.Marshal(chunk)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chunk to JSON: %w", err)
	}

	// Формируем SSE event
	sseEvent := fmt.Sprintf("data: %s\n\n", string(jsonData))

	// 🔍 КРИТИЧНОЕ ЛОГИРОВАНИЕ: показываем весь JSON для debugging
	hasToolCalls := len(chunk.Choices) > 0 && len(chunk.Choices[0].Delta.ToolCalls) > 0
	s.logger.WithFields(logrus.Fields{
		"chunk_id":       chunk.ID,
		"chunk_size":     len(jsonData),
		"has_tool_calls": hasToolCalls,
		"json_data":      string(jsonData), // ⬅️ ВЕСЬ JSON!
	}).Debug("Converted chunk to SSE format")

	return sseEvent, nil
}

// CreateDoneEvent создает финальный SSE event [DONE]
func (s *StreamConverter) CreateDoneEvent() string {
	return "data: [DONE]\n\n"
}

// CreateErrorEvent создает SSE event с ошибкой
func (s *StreamConverter) CreateErrorEvent(err error) string {
	errorData := map[string]interface{}{
		"error": map[string]interface{}{
			"message": err.Error(),
			"type":    "api_error",
			"code":    "streaming_error",
		},
	}

	jsonData, jsonErr := json.Marshal(errorData)
	if jsonErr != nil {
		// Fallback если не можем сериализовать ошибку
		return fmt.Sprintf("data: {\"error\":{\"message\":\"%s\",\"type\":\"api_error\"}}\n\n", err.Error())
	}

	return fmt.Sprintf("data: %s\n\n", string(jsonData))
}

// StreamingMetrics содержит метрики streaming запроса
type StreamingMetrics struct {
	RequestID        string        `json:"request_id"`
	Model            string        `json:"model"`
	StartTime        time.Time     `json:"start_time"`
	FirstChunkTime   time.Time     `json:"first_chunk_time,omitempty"`
	LastChunkTime    time.Time     `json:"last_chunk_time,omitempty"`
	TotalDuration    time.Duration `json:"total_duration"`
	ChunksCount      int           `json:"chunks_count"`
	TotalTokens      int           `json:"total_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	Success          bool          `json:"success"`
	ErrorMessage     string        `json:"error_message,omitempty"`
}

// NewStreamingMetrics создает новые метрики для streaming запроса
func NewStreamingMetrics(requestID, model string) *StreamingMetrics {
	return &StreamingMetrics{
		RequestID: requestID,
		Model:     model,
		StartTime: time.Now(),
	}
}

// RecordFirstChunk записывает время первого chunk'а
func (m *StreamingMetrics) RecordFirstChunk() {
	if m.FirstChunkTime.IsZero() {
		m.FirstChunkTime = time.Now()
	}
}

// RecordChunk записывает получение chunk'а
func (m *StreamingMetrics) RecordChunk(tokens int) {
	m.ChunksCount++
	m.CompletionTokens += tokens
	m.LastChunkTime = time.Now()
}

// Finalize завершает метрики
func (m *StreamingMetrics) Finalize(success bool, err error) {
	m.Success = success
	m.TotalDuration = time.Since(m.StartTime)

	if err != nil {
		m.ErrorMessage = err.Error()
	}

	// Если нет first chunk time, используем start time
	if m.FirstChunkTime.IsZero() {
		m.FirstChunkTime = m.StartTime
	}

	// Если нет last chunk time, используем текущее время
	if m.LastChunkTime.IsZero() {
		m.LastChunkTime = time.Now()
	}
}

// GetTimeToFirstChunk возвращает время до первого chunk'а
func (m *StreamingMetrics) GetTimeToFirstChunk() time.Duration {
	if m.FirstChunkTime.IsZero() {
		return 0
	}
	return m.FirstChunkTime.Sub(m.StartTime)
}

// GetStreamingRate возвращает скорость streaming (chunks per second)
func (m *StreamingMetrics) GetStreamingRate() float64 {
	if m.TotalDuration == 0 {
		return 0
	}
	return float64(m.ChunksCount) / m.TotalDuration.Seconds()
}

// GetTokensPerSecond возвращает скорость генерации токенов
func (m *StreamingMetrics) GetTokensPerSecond() float64 {
	if m.TotalDuration == 0 {
		return 0
	}
	return float64(m.CompletionTokens) / m.TotalDuration.Seconds()
}

// convertToolCallsToOpenAI конвертирует Ollama tool calls в OpenAI формат
func (s *StreamConverter) convertToolCallsToOpenAI(ollamaToolCalls []ollama.ToolCall) []models.ToolCall {
	openaiToolCalls := make([]models.ToolCall, len(ollamaToolCalls))

	for i, ollamaTool := range ollamaToolCalls {
		// Генерируем ID для tool call
		toolCallID := fmt.Sprintf("call_%d_%d", time.Now().UnixNano(), i)

		// Конвертируем аргументы в JSON строку если это map
		var argsJSON string
		switch args := ollamaTool.Function.Arguments.(type) {
		case string:
			argsJSON = args
		case map[string]interface{}:
			argsBytes, err := json.Marshal(args)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to marshal tool call arguments")
				argsJSON = "{}"
			} else {
				argsJSON = string(argsBytes)
			}
		default:
			argsBytes, err := json.Marshal(args)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to marshal tool call arguments")
				argsJSON = "{}"
			} else {
				argsJSON = string(argsBytes)
			}
		}

		openaiToolCalls[i] = models.ToolCall{
			Index: i, // 🔧 КРИТИЧНО: Index обязателен для OpenAI streaming chunks!
			ID:    toolCallID,
			Type:  "function",
			Function: models.FunctionCall{
				Name:      ollamaTool.Function.Name,
				Arguments: argsJSON,
			},
		}
	}

	return openaiToolCalls
}
