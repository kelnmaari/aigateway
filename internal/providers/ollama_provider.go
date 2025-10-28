package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/models"
)

// OllamaProvider реализует Provider interface для Ollama
type OllamaProvider struct {
	name    string
	baseURL string
	client  *http.Client
}

// NewOllamaProvider создает новый Ollama provider
func NewOllamaProvider(name, baseURL string) *OllamaProvider {
	return &OllamaProvider{
		name:    name,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetName возвращает имя provider
func (p *OllamaProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider
func (p *OllamaProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeOllama
}

// HealthCheck проверяет доступность Ollama server
func (p *OllamaProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama server returned status %d", resp.StatusCode)
	}

	return nil
}

// OllamaListResponse представляет ответ Ollama API /api/tags
type OllamaListResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

// OllamaModelInfo представляет информацию о модели из Ollama
type OllamaModelInfo struct {
	Name       string                 `json:"name"`
	Model      string                 `json:"model"`
	ModifiedAt string                 `json:"modified_at"`
	Size       int64                  `json:"size"`
	Digest     string                 `json:"digest"`
	Details    map[string]interface{} `json:"details"`
}

// ListModels возвращает список моделей из Ollama
func (p *OllamaProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error: %s", string(body))
	}

	var listResp OllamaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var providerModels []*ProviderModel
	for _, ollamaModel := range listResp.Models {
		model := &ProviderModel{
			ID:           ollamaModel.Name,
			Name:         ollamaModel.Name,
			Capabilities: []models.ModelCapability{models.CapabilityChat}, // По умолчанию все Ollama модели поддерживают chat
			RequiresGPU:  true,                                            // Предполагаем что нужен GPU
			Tags:         []string{"ollama", "local"},
			ProviderMeta: map[string]interface{}{
				"size":        ollamaModel.Size,
				"digest":      ollamaModel.Digest,
				"modified_at": ollamaModel.ModifiedAt,
			},
		}

		// Попытка определить context length из details
		if details, ok := ollamaModel.Details["parameter_size"]; ok {
			if paramSize, ok := details.(string); ok {
				model.Description = fmt.Sprintf("Parameter size: %s", paramSize)
			}
		}

		// Определение capabilities из имени модели
		modelNameLower := strings.ToLower(ollamaModel.Name)
		if strings.Contains(modelNameLower, "vision") || strings.Contains(modelNameLower, "llava") {
			model.Capabilities = append(model.Capabilities, models.CapabilityVision)
		}
		if strings.Contains(modelNameLower, "embed") {
			model.Capabilities = append(model.Capabilities, models.CapabilityEmbeddings)
		}
		if strings.Contains(modelNameLower, "code") {
			model.Capabilities = append(model.Capabilities, models.CapabilityCodeCompletion)
		}

		providerModels = append(providerModels, model)
	}

	return providerModels, nil
}

// GetModelInfo получает информацию о конкретной модели
func (p *OllamaProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
	// Ollama не имеет endpoint для получения информации о конкретной модели
	// Поэтому получаем список всех моделей и ищем нужную
	models, err := p.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	for _, model := range models {
		if model.ID == modelID {
			return model, nil
		}
	}

	return nil, fmt.Errorf("model not found: %s", modelID)
}

