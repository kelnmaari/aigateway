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

// DeepSeekProvider реализует Provider interface для DeepSeek API.
// DeepSeek использует OpenAI-совместимый формат запросов и ответов.
type DeepSeekProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewDeepSeekProvider создает новый DeepSeek provider.
func NewDeepSeekProvider(name, baseURL, apiKey string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return &DeepSeekProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName возвращает имя provider.
func (p *DeepSeekProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider.
func (p *DeepSeekProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeDeepSeek
}

// HealthCheck проверяет доступность DeepSeek API.
func (p *DeepSeekProvider) HealthCheck(ctx context.Context) error {
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

// ListModels возвращает список моделей из DeepSeek.
func (p *DeepSeekProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
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
		capabilities := inferDeepSeekCapabilities(m.ID)
		result = append(result, &ProviderModel{
			ID:           m.ID,
			Name:         m.ID,
			Capabilities: capabilities,
			Description:  fmt.Sprintf("DeepSeek model: %s", m.ID),
			Tags:         []string{"deepseek"},
			ProviderMeta: map[string]any{
				"owned_by": m.OwnedBy,
				"created":  m.Created,
			},
		})
	}

	return result, nil
}

// GetModelInfo получает информацию о конкретной модели DeepSeek.
func (p *DeepSeekProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
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

	capabilities := inferDeepSeekCapabilities(m.ID)
	return &ProviderModel{
		ID:           m.ID,
		Name:         m.ID,
		Capabilities: capabilities,
		Description:  fmt.Sprintf("DeepSeek model: %s", m.ID),
		Tags:         []string{"deepseek"},
		ProviderMeta: map[string]any{
			"owned_by": m.OwnedBy,
			"created":  m.Created,
		},
	}, nil
}

// inferDeepSeekCapabilities infers capabilities based on DeepSeek model ID.
func inferDeepSeekCapabilities(modelID string) []models.ModelCapability {
	id := strings.ToLower(modelID)

	caps := models.ChatModelCapabilities()

	// DeepSeek-V3 and DeepSeek-R1 support function calling
	if strings.Contains(id, "deepseek-v3") || strings.Contains(id, "deepseek-chat") ||
		strings.Contains(id, "deepseek-r1") || strings.Contains(id, "deepseek-reasoner") {
		caps = append(caps, models.CapabilityFunctionCalling)
	}

	return caps
}
