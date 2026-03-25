// Package embeddings provides embedding generation interfaces and implementations.
package embeddings

import (
	"context"
)

// Embedding represents a vector embedding
type Embedding struct {
	Vector     []float64      // Embedding vector
	Dimensions int            // Vector dimensions
	Model      string         // Model used for embedding
	Metadata   map[string]any // Additional metadata
}

// EmbeddingRequest request для генерации embeddings
type EmbeddingRequest struct {
	Text     string         // Текст для embedding
	Model    string         // Модель (optional, uses default if empty)
	Metadata map[string]any // Дополнительные метаданные
}

// BatchEmbeddingRequest batch request для embeddings
type BatchEmbeddingRequest struct {
	Texts    []string       // Массив текстов
	Model    string         // Модель
	Metadata map[string]any // Общие метаданные
}

// BatchEmbeddingResponse batch response
type BatchEmbeddingResponse struct {
	Embeddings  []Embedding // Embeddings в том же порядке что и texts
	Model       string      // Использованная модель
	TotalTokens int         // Общее количество токенов
}

// Embedder interface для генерации embeddings
type Embedder interface {
	// Embed генерирует embedding для одного текста
	Embed(ctx context.Context, req EmbeddingRequest) (*Embedding, error)

	// EmbedBatch генерирует embeddings для batch текстов
	EmbedBatch(ctx context.Context, req BatchEmbeddingRequest) (*BatchEmbeddingResponse, error)

	// GetDimensions возвращает размерность векторов для модели
	GetDimensions(model string) (int, error)

	// GetDefaultModel возвращает модель по умолчанию
	GetDefaultModel() string

	// Name возвращает имя embedder'а
	Name() string
}

// EmbedderConfig конфигурация для embedder
type EmbedderConfig struct {
	Provider     string  // "openai", "custom"
	BaseURL      string  // Base URL для API
	APIKey       string  // API key for authentication
	Model        string  // Модель по умолчанию
	Dimensions   int     // Размерность векторов
	Timeout      int     // Timeout в секундах
	MaxBatchSize int     // Максимальный размер batch
	RateLimit    float64 // Ограничение запросов/сек (0 = без ограничений)
}

// DefaultEmbedderConfig возвращает конфигурацию по умолчанию
func DefaultEmbedderConfig() EmbedderConfig {
	return EmbedderConfig{
		Provider:     "openai",
		BaseURL:      "https://api.openai.com",
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      30,
		MaxBatchSize: 100,
		RateLimit:    10.0, // 10 req/sec
	}
}
