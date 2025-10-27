// Package converter provides conversion between OpenAI and Ollama API formats
package converter

import (
	"fmt"

	ollamaapi "aigateway/internal/client/ollama"
	"aigateway/internal/models"
)

// ConvertEmbeddingRequest конвертирует OpenAI embedding request в Ollama format
// defaultModel - модель по умолчанию из конфига (может быть пустой)
func ConvertEmbeddingRequest(openaiReq *models.EmbeddingRequest, defaultModel string) (*ollamaapi.EmbedRequest, error) {
	if openaiReq == nil {
		return nil, fmt.Errorf("embedding request is nil")
	}

	if openaiReq.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	// Определяем модель для использования
	model := resolveEmbeddingModel(openaiReq.Model, defaultModel)
	if model == "" {
		return nil, fmt.Errorf("model is required (not specified in request and no default model configured)")
	}

	// Подготавливаем input
	// OpenAI поддерживает: string, []string, []int, [][]int
	// Ollama также поддерживает все эти форматы
	input := openaiReq.Input

	// Создаем Ollama request
	ollamaReq := &ollamaapi.EmbedRequest{
		Model: model,
		Input: input,
	}

	// Опциональные параметры
	if openaiReq.Dimensions != nil && *openaiReq.Dimensions > 0 {
		ollamaReq.Dimensions = *openaiReq.Dimensions
	}

	// Encoding format игнорируем - Ollama всегда возвращает float

	return ollamaReq, nil
}

// ConvertEmbeddingResponse конвертирует Ollama embedding response в OpenAI format
func ConvertEmbeddingResponse(ollamaResp *ollamaapi.EmbedResponse, model string) (*models.EmbeddingResponse, error) {
	if ollamaResp == nil {
		return nil, fmt.Errorf("ollama embedding response is nil")
	}

	// Конвертируем [][]float32 в []Embedding с []float64
	embeddings := make([]models.Embedding, len(ollamaResp.Embeddings))

	for i, embFloat32 := range ollamaResp.Embeddings {
		// Конвертируем float32 -> float64
		embFloat64 := make([]float64, len(embFloat32))
		for j, val := range embFloat32 {
			embFloat64[j] = float64(val)
		}

		embeddings[i] = models.Embedding{
			Object:    "embedding",
			Index:     i,
			Embedding: embFloat64,
		}
	}

	// Подсчитываем токены
	// PromptEvalCount - это общее количество токенов для всех входов
	totalTokens := ollamaResp.PromptEvalCount

	// Если prompt_eval_count не задан, используем эвристику
	if totalTokens == 0 && len(embeddings) > 0 {
		// Грубая оценка: ~1 токен на 4 символа в среднем
		totalTokens = len(embeddings) * 10 // Минимальная оценка
	}

	// Создаем OpenAI response
	response := &models.EmbeddingResponse{
		Object: "list",
		Data:   embeddings,
		Model:  model,
		Usage: models.Usage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}

	return response, nil
}

// resolveEmbeddingModel определяет модель для embeddings
// Логика:
// 1. Если requestModel пустой -> используем defaultModel
// 2. Если requestModel начинается с "text-embedding-" (OpenAI модель) -> используем defaultModel
// 3. Иначе используем requestModel (это Ollama модель)
func resolveEmbeddingModel(requestModel, defaultModel string) string {
	// Если модель не указана в запросе, используем дефолтную
	if requestModel == "" {
		return defaultModel
	}

	// Если указана OpenAI модель (text-embedding-*), заменяем на дефолтную
	// OpenAI модели: text-embedding-ada-002, text-embedding-3-small, text-embedding-3-large
	if len(requestModel) >= 15 && requestModel[:15] == "text-embedding-" {
		if defaultModel != "" {
			return defaultModel
		}
		// Если дефолтной нет, оставляем как есть (может быть маппинг где-то еще)
		return requestModel
	}

	// Это Ollama модель, используем как есть
	return requestModel
}

// normalizeInput нормализует input к массиву строк
func normalizeInput(input interface{}) ([]string, error) {
	switch v := input.(type) {
	case string:
		return []string{v}, nil
	case []string:
		return v, nil
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			} else {
				return nil, fmt.Errorf("invalid input type at index %d: expected string", i)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported input type: %T", input)
	}
}

