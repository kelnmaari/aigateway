package providers

import (
	"context"
	"fmt"

	"aigateway/internal/models"
)

// VLLMProvider реализует Provider interface для vLLM
// Note: Полная реализация будет в VLLM-01 задаче
type VLLMProvider struct {
	name    string
	baseURL string
}

// NewVLLMProvider создает новый vLLM provider
func NewVLLMProvider(name, baseURL string) *VLLMProvider {
	return &VLLMProvider{
		name:    name,
		baseURL: baseURL,
	}
}

// GetName возвращает имя provider
func (p *VLLMProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider
func (p *VLLMProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeVLLM
}

// HealthCheck проверяет доступность vLLM server
// TODO: Implement in VLLM-01
func (p *VLLMProvider) HealthCheck(ctx context.Context) error {
	return fmt.Errorf("vLLM health check not implemented yet (VLLM-01)")
}

// ListModels возвращает список моделей из vLLM
// TODO: Implement in VLLM-01
func (p *VLLMProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
	return nil, fmt.Errorf("vLLM list models not implemented yet (VLLM-01)")
}

// GetModelInfo получает информацию о конкретной модели
// TODO: Implement in VLLM-01
func (p *VLLMProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
	return nil, fmt.Errorf("vLLM get model info not implemented yet (VLLM-01)")
}

