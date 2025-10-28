package providers

import (
	"context"
	"time"

	"aigateway/internal/models"
)

// Provider представляет абстракцию для работы с model providers (Ollama, vLLM, OpenAI, etc)
type Provider interface {
	// GetName возвращает имя provider
	GetName() string

	// GetType возвращает тип provider (ollama, vllm, openai)
	GetType() models.ModelProviderType

	// HealthCheck проверяет доступность provider
	HealthCheck(ctx context.Context) error

	// ListModels возвращает список доступных моделей
	ListModels(ctx context.Context) ([]*ProviderModel, error)

	// GetModelInfo получает информацию о конкретной модели
	GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error)
}

// ProviderModel представляет информацию о модели из provider
type ProviderModel struct {
	ID             string                   // "llama2:7b", "gpt-4"
	Name           string                   // Display name
	Capabilities   []models.ModelCapability // ["chat", "embeddings"]
	Parameters     map[string]interface{}   // Model-specific parameters
	RequiresGPU    bool
	MinVRAMGB      *int
	ContextLength  *int
	Description    string
	Tags           []string
	ProviderMeta   map[string]interface{} // Provider-specific metadata
}

// ProviderHealth представляет результат health check
type ProviderHealth struct {
	Status      models.ModelHealthStatus
	Latency     time.Duration
	ErrorMsg    string
	CheckedAt   time.Time
}

