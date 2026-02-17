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

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com"
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider реализует Provider interface для Anthropic Claude API.
type AnthropicProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewAnthropicProvider создает новый Anthropic provider.
func NewAnthropicProvider(name, baseURL, apiKey string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = anthropicDefaultBaseURL
	}
	return &AnthropicProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName возвращает имя provider.
func (p *AnthropicProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider.
func (p *AnthropicProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeAnthropic
}

// HealthCheck проверяет доступность Anthropic API.
func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(req)

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

// anthropicModelsResponse represents the response from GET /v1/models.
type anthropicModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		CreatedAt   string `json:"created_at"`
		Type        string `json:"type"`
	} `json:"data"`
	HasMore bool   `json:"has_more"`
	FirstID string `json:"first_id"`
	LastID  string `json:"last_id"`
}

// ListModels возвращает список моделей из Anthropic.
func (p *AnthropicProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list models returned %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp anthropicModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make([]*ProviderModel, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		name := m.DisplayName
		if name == "" {
			name = m.ID
		}
		capabilities := inferAnthropicCapabilities(m.ID)
		result = append(result, &ProviderModel{
			ID:           m.ID,
			Name:         name,
			Capabilities: capabilities,
			Description:  fmt.Sprintf("Anthropic model: %s", name),
			Tags:         []string{"anthropic", "claude"},
			ProviderMeta: map[string]interface{}{
				"created_at": m.CreatedAt,
				"type":       m.Type,
			},
		})
	}

	return result, nil
}

// GetModelInfo получает информацию о конкретной модели.
func (p *AnthropicProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/models/"+modelID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(req)

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
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		CreatedAt   string `json:"created_at"`
		Type        string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	name := m.DisplayName
	if name == "" {
		name = m.ID
	}
	capabilities := inferAnthropicCapabilities(m.ID)
	return &ProviderModel{
		ID:           m.ID,
		Name:         name,
		Capabilities: capabilities,
		Description:  fmt.Sprintf("Anthropic model: %s", name),
		Tags:         []string{"anthropic", "claude"},
		ProviderMeta: map[string]interface{}{
			"created_at": m.CreatedAt,
			"type":       m.Type,
		},
	}, nil
}

// setHeaders sets the required Anthropic API headers.
func (p *AnthropicProvider) setHeaders(req *http.Request) {
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)
	req.Header.Set("Content-Type", "application/json")
}

// inferAnthropicCapabilities infers capabilities based on model ID.
func inferAnthropicCapabilities(modelID string) []models.ModelCapability {
	id := strings.ToLower(modelID)

	caps := models.ChatModelCapabilities()
	caps = append(caps, models.CapabilityFunctionCalling)

	// Claude 3+ models support vision
	if strings.Contains(id, "claude-3") || strings.Contains(id, "claude-4") {
		caps = append(caps, models.CapabilityVision)
	}

	return caps
}
