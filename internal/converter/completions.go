// Package converter provides conversion between OpenAI and Ollama API formats
package converter

import (
	"fmt"

	"aigateway/internal/models"
)

// ConvertCompletionToChatRequest конвертирует legacy CompletionRequest в ChatCompletionRequest
// Legacy completions API использует простой prompt, который мы конвертируем в chat format
func ConvertCompletionToChatRequest(req *models.CompletionRequest) (*models.ChatCompletionRequest, error) {
	if req == nil {
		return nil, fmt.Errorf("completion request is nil")
	}

	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	// Конвертируем prompt в messages
	messages, err := promptToMessages(req.Prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to convert prompt to messages: %w", err)
	}

	// Создаем ChatCompletionRequest
	chatReq := &models.ChatCompletionRequest{
		Model:            req.Model,
		Messages:         messages,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		N:                req.N,
		Stream:           req.Stream,
		Stop:             req.Stop,
		MaxTokens:        req.MaxTokens,
		PresencePenalty:  req.PresencePenalty,
		FrequencyPenalty: req.FrequencyPenalty,
		LogitBias:        req.LogitBias,
		User:             req.User,
	}

	return chatReq, nil
}

// ConvertChatToCompletionResponse конвертирует ChatCompletionResponse в CompletionResponse
func ConvertChatToCompletionResponse(chatResp *models.ChatCompletionResponse, originalModel string) (*models.CompletionResponse, error) {
	if chatResp == nil {
		return nil, fmt.Errorf("chat response is nil")
	}

	// Конвертируем choices
	choices := make([]models.CompletionChoice, len(chatResp.Choices))
	for i, chatChoice := range chatResp.Choices {
		// Извлекаем content как string
		content := ""
		if chatChoice.Message.Content != nil {
			if str, ok := chatChoice.Message.Content.(string); ok {
				content = str
			}
		}

		choices[i] = models.CompletionChoice{
			Text:         content,
			Index:        chatChoice.Index,
			FinishReason: chatChoice.FinishReason,
			// LogProbs не поддерживается в chat API, оставляем nil
		}
	}

	// Создаем CompletionResponse
	response := &models.CompletionResponse{
		ID:      chatResp.ID,
		Object:  "text_completion", // Legacy completion object type
		Created: chatResp.Created,
		Model:   originalModel,
		Choices: choices,
		Usage:   chatResp.Usage,
	}

	return response, nil
}

// promptToMessages конвертирует prompt (string или []string) в chat messages
func promptToMessages(prompt interface{}) ([]models.ChatMessage, error) {
	if prompt == nil {
		return nil, fmt.Errorf("prompt is required")
	}

	switch p := prompt.(type) {
	case string:
		// Один prompt -> одно user сообщение
		return []models.ChatMessage{
			{
				Role:    "user",
				Content: p,
			},
		}, nil

	case []string:
		// Массив prompts -> несколько user сообщений
		if len(p) == 0 {
			return nil, fmt.Errorf("prompt array is empty")
		}
		messages := make([]models.ChatMessage, len(p))
		for i, promptStr := range p {
			messages[i] = models.ChatMessage{
				Role:    "user",
				Content: promptStr,
			}
		}
		return messages, nil

	case []interface{}:
		// Интерфейс массив -> конвертируем в строки
		if len(p) == 0 {
			return nil, fmt.Errorf("prompt array is empty")
		}
		messages := make([]models.ChatMessage, len(p))
		for i, item := range p {
			if str, ok := item.(string); ok {
				messages[i] = models.ChatMessage{
					Role:    "user",
					Content: str,
				}
			} else {
				return nil, fmt.Errorf("prompt array contains non-string element at index %d", i)
			}
		}
		return messages, nil

	default:
		return nil, fmt.Errorf("unsupported prompt type: %T", prompt)
	}
}

// ConvertCompletionStreamChunk конвертирует streaming chat chunk в completion format
func ConvertCompletionStreamChunk(chatChunk *models.ChatCompletionChunk, model string) (*models.CompletionStreamChunk, error) {
	if chatChunk == nil {
		return nil, fmt.Errorf("chat chunk is nil")
	}

	// Конвертируем choices
	choices := make([]models.CompletionStreamChoice, len(chatChunk.Choices))
	for i, chatChoice := range chatChunk.Choices {
		// Извлекаем content как string
		content := ""
		if chatChoice.Delta.Content != nil {
			if str, ok := chatChoice.Delta.Content.(string); ok {
				content = str
			}
		}

		// Извлекаем finish_reason
		finishReason := ""
		if chatChoice.FinishReason != nil {
			finishReason = *chatChoice.FinishReason
		}

		choices[i] = models.CompletionStreamChoice{
			Text:         content,
			Index:        chatChoice.Index,
			FinishReason: finishReason,
		}
	}

	// Создаем CompletionStreamChunk
	chunk := &models.CompletionStreamChunk{
		ID:      chatChunk.ID,
		Object:  "text_completion",
		Created: chatChunk.Created,
		Model:   model,
		Choices: choices,
	}

	return chunk, nil
}

