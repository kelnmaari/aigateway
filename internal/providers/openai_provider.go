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

// OpenAIProvider реализует Provider interface для OpenAI API.
type OpenAIProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOpenAIProvider создает новый OpenAI provider.
func NewOpenAIProvider(name, baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	return &OpenAIProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName возвращает имя provider.
func (p *OpenAIProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider.
func (p *OpenAIProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeOpenAI
}

// HealthCheck проверяет доступность OpenAI API.
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("unauthorized: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// openaiModelsResponse represents the response from GET /v1/models.
type openaiModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

// ListModels возвращает список моделей из OpenAI.
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list models returned %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp openaiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make([]*ProviderModel, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		capabilities := inferOpenAICapabilities(m.ID)
		result = append(result, &ProviderModel{
			ID:           m.ID,
			Name:         m.ID,
			Capabilities: capabilities,
			Description:  fmt.Sprintf("OpenAI model: %s", m.ID),
			Tags:         []string{"openai"},
			ProviderMeta: map[string]interface{}{
				"owned_by": m.OwnedBy,
				"created":  m.Created,
			},
		})
	}

	return result, nil
}

// GetModelInfo получает информацию о конкретной модели.
func (p *OpenAIProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models/"+modelID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get model info failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get model info returned %d: %s", resp.StatusCode, string(body))
	}

	var m struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	capabilities := inferOpenAICapabilities(m.ID)
	return &ProviderModel{
		ID:           m.ID,
		Name:         m.ID,
		Capabilities: capabilities,
		Description:  fmt.Sprintf("OpenAI model: %s", m.ID),
		Tags:         []string{"openai"},
		ProviderMeta: map[string]interface{}{
			"owned_by": m.OwnedBy,
			"created":  m.Created,
		},
	}, nil
}

// inferOpenAICapabilities infers capabilities based on model ID naming patterns.
func inferOpenAICapabilities(modelID string) []models.ModelCapability {
	id := strings.ToLower(modelID)

	if strings.Contains(id, "embedding") {
		return models.EmbeddingModelCapabilities()
	}

	caps := models.ChatModelCapabilities()

	if strings.Contains(id, "gpt-4") || strings.Contains(id, "o1") || strings.Contains(id, "o3") {
		caps = append(caps, models.CapabilityVision, models.CapabilityFunctionCalling)
	} else if strings.Contains(id, "gpt-3.5") {
		caps = append(caps, models.CapabilityFunctionCalling)
	}

	return caps
}
