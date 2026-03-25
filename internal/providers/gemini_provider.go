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
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"
)

// GeminiProvider реализует Provider interface для Google Gemini API.
type GeminiProvider struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewGeminiProvider создает новый Gemini provider.
func NewGeminiProvider(name, baseURL, apiKey string) *GeminiProvider {
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}
	return &GeminiProvider{
		name:    name,
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetName возвращает имя provider.
func (p *GeminiProvider) GetName() string {
	return p.name
}

// GetType возвращает тип provider.
func (p *GeminiProvider) GetType() models.ModelProviderType {
	return models.ProviderTypeGemini
}

// HealthCheck проверяет доступность Gemini API.
func (p *GeminiProvider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1beta/models?key=%s&pageSize=1", p.baseURL, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("unauthorized: invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// geminiModelsResponse represents the response from GET /v1beta/models.
type geminiModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"` // "models/gemini-1.5-pro"
		DisplayName                string   `json:"displayName"`
		Description                string   `json:"description"`
		Version                    string   `json:"version"`
		InputTokenLimit            int      `json:"inputTokenLimit"`
		OutputTokenLimit           int      `json:"outputTokenLimit"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	} `json:"models"`
	NextPageToken string `json:"nextPageToken"`
}

// ListModels возвращает список моделей из Gemini.
func (p *GeminiProvider) ListModels(ctx context.Context) ([]*ProviderModel, error) {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", p.baseURL, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list models returned %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp geminiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make([]*ProviderModel, 0, len(modelsResp.Models))
	for _, m := range modelsResp.Models {
		// Extract model ID from name (e.g., "models/gemini-1.5-pro" -> "gemini-1.5-pro")
		modelID := m.Name
		if after, ok := strings.CutPrefix(modelID, "models/"); ok {
			modelID = after
		}

		capabilities := inferGeminiCapabilities(modelID, m.SupportedGenerationMethods)
		contextLen := m.InputTokenLimit + m.OutputTokenLimit

		result = append(result, &ProviderModel{
			ID:            modelID,
			Name:          m.DisplayName,
			Capabilities:  capabilities,
			ContextLength: &contextLen,
			Description:   m.Description,
			Tags:          []string{"gemini", "google"},
			ProviderMeta: map[string]any{
				"version":            m.Version,
				"input_token_limit":  m.InputTokenLimit,
				"output_token_limit": m.OutputTokenLimit,
				"generation_methods": m.SupportedGenerationMethods,
			},
		})
	}

	return result, nil
}

// GetModelInfo получает информацию о конкретной модели.
func (p *GeminiProvider) GetModelInfo(ctx context.Context, modelID string) (*ProviderModel, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s?key=%s", p.baseURL, modelID, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

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
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		Description                string   `json:"description"`
		Version                    string   `json:"version"`
		InputTokenLimit            int      `json:"inputTokenLimit"`
		OutputTokenLimit           int      `json:"outputTokenLimit"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	mid := m.Name
	if after, ok := strings.CutPrefix(mid, "models/"); ok {
		mid = after
	}

	capabilities := inferGeminiCapabilities(mid, m.SupportedGenerationMethods)
	contextLen := m.InputTokenLimit + m.OutputTokenLimit

	return &ProviderModel{
		ID:            mid,
		Name:          m.DisplayName,
		Capabilities:  capabilities,
		ContextLength: &contextLen,
		Description:   m.Description,
		Tags:          []string{"gemini", "google"},
		ProviderMeta: map[string]any{
			"version":            m.Version,
			"input_token_limit":  m.InputTokenLimit,
			"output_token_limit": m.OutputTokenLimit,
			"generation_methods": m.SupportedGenerationMethods,
		},
	}, nil
}

// inferGeminiCapabilities infers capabilities based on model ID and supported methods.
func inferGeminiCapabilities(modelID string, methods []string) []models.ModelCapability {
	id := strings.ToLower(modelID)

	// Check for embedding models
	if strings.Contains(id, "embedding") {
		return models.EmbeddingModelCapabilities()
	}

	caps := models.ChatModelCapabilities()

	// Gemini Pro and Flash models support function calling and vision
	if strings.Contains(id, "gemini") {
		caps = append(caps, models.CapabilityFunctionCalling, models.CapabilityVision)
	}

	return caps
}
