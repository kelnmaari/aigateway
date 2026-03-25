// Package rag provides RAG capabilities for GitLab code review
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TEIEmbeddingProvider implements EmbeddingProvider using TEI (Text Embeddings Inference)
type TEIEmbeddingProvider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *logrus.Logger
}

// TEIConfig holds configuration for TEI embedding provider
type TEIConfig struct {
	URL     string        `yaml:"url" json:"url"`
	APIKey  string        `yaml:"api_key" json:"api_key"`
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// NewTEIEmbeddingProvider creates a new TEI embedding provider
func NewTEIEmbeddingProvider(config TEIConfig, logger *logrus.Logger) *TEIEmbeddingProvider {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &TEIEmbeddingProvider{
		baseURL: config.URL,
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// GenerateEmbedding creates a vector embedding for the given text
func (p *TEIEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// GenerateEmbeddings creates vector embeddings for multiple texts
func (p *TEIEmbeddingProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// TEI uses OpenAI-compatible API format
	reqBody := map[string]any{
		"input": texts,
		"model": "default", // TEI ignores model name but requires it
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := p.baseURL + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	p.logger.WithFields(logrus.Fields{
		"url":   url,
		"count": len(texts),
	}).Debug("Generating embeddings via TEI")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TEI returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Sort by index and extract embeddings
	embeddings := make([][]float32, len(texts))
	for _, item := range result.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	p.logger.WithField("count", len(embeddings)).Debug("Embeddings generated successfully")
	return embeddings, nil
}

// HealthCheck verifies the TEI service is available
func (p *TEIEmbeddingProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("TEI health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TEI unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// ModelInstanceProvider provides access to running inference models
type ModelInstanceProvider interface {
	// GetModel returns a running model instance by alias
	GetModel(alias string) (endpoint string, running bool)
	// GetRunningEmbeddingModel finds any running embedding model automatically
	GetRunningEmbeddingModel() (alias string, endpoint string, found bool)
}

// DynamicEmbeddingProvider implements EmbeddingProvider with dynamic URL resolution
// It gets the embedding model URL from the inference registry at runtime
// If ModelAlias is empty, it automatically finds any running embedding model
type DynamicEmbeddingProvider struct {
	modelAlias     string                // Alias of the embedding model to use (empty = auto-detect)
	instanceGetter ModelInstanceProvider // Provides model instance info
	apiKey         string
	httpClient     *http.Client
	logger         *logrus.Logger
}

// DynamicEmbeddingConfig holds configuration for dynamic embedding provider
type DynamicEmbeddingConfig struct {
	ModelAlias string        // Embedding model alias (empty = auto-detect any running embedding model)
	APIKey     string        // Optional API key for embedding requests
	Timeout    time.Duration // HTTP timeout
}

// NewDynamicEmbeddingProvider creates an embedding provider that resolves URLs dynamically
// If ModelAlias is empty, it will automatically find any running embedding model (TEI or with "embeddings" capability)
func NewDynamicEmbeddingProvider(cfg DynamicEmbeddingConfig, instanceGetter ModelInstanceProvider, logger *logrus.Logger) *DynamicEmbeddingProvider {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &DynamicEmbeddingProvider{
		modelAlias:     cfg.ModelAlias,
		instanceGetter: instanceGetter,
		apiKey:         cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// getBaseURL resolves the embedding model URL from inference registry
// If modelAlias is empty, automatically finds any running embedding model
func (p *DynamicEmbeddingProvider) getBaseURL() (string, error) {
	var endpoint string
	var running bool
	var modelName string

	p.logger.WithField("configured_alias", p.modelAlias).Debug("Looking for embedding model")

	if p.modelAlias != "" {
		// Use explicitly configured model
		endpoint, running = p.instanceGetter.GetModel(p.modelAlias)
		modelName = p.modelAlias
		p.logger.WithFields(logrus.Fields{
			"alias":    p.modelAlias,
			"endpoint": endpoint,
			"running":  running,
		}).Debug("Checked configured embedding model")
		if !running {
			return "", fmt.Errorf("embedding model '%s' is not running", p.modelAlias)
		}
	} else {
		// Auto-detect any running embedding model
		var found bool
		modelName, endpoint, found = p.instanceGetter.GetRunningEmbeddingModel()
		p.logger.WithFields(logrus.Fields{
			"found":    found,
			"model":    modelName,
			"endpoint": endpoint,
		}).Debug("Auto-detect embedding model result")
		if !found {
			return "", fmt.Errorf("no running embedding model found (start a TEI model via Admin -> Models)")
		}
	}

	if endpoint == "" {
		return "", fmt.Errorf("embedding model '%s' has no endpoint", modelName)
	}
	return endpoint, nil
}

// GenerateEmbedding creates a vector embedding for the given text
func (p *DynamicEmbeddingProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := p.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	return embeddings[0], nil
}

// GenerateEmbeddings creates vector embeddings for multiple texts
func (p *DynamicEmbeddingProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	baseURL, err := p.getBaseURL()
	if err != nil {
		return nil, err
	}

	// Use OpenAI-compatible API format
	reqBody := map[string]any{
		"input": texts,
		"model": "default", // TEI ignores model name but requires it
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := baseURL + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	p.logger.WithFields(logrus.Fields{
		"url":        url,
		"count":      len(texts),
		"modelAlias": p.modelAlias,
	}).Debug("Generating embeddings via dynamic provider")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding model returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Sort by index and extract embeddings
	embeddings := make([][]float32, len(texts))
	for _, item := range result.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	p.logger.WithField("count", len(embeddings)).Debug("Dynamic embeddings generated successfully")
	return embeddings, nil
}

// HealthCheck verifies the embedding model is running and healthy
func (p *DynamicEmbeddingProvider) HealthCheck(ctx context.Context) error {
	baseURL, err := p.getBaseURL()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("embedding model health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("embedding model unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// ModelAlias returns the configured embedding model alias
func (p *DynamicEmbeddingProvider) ModelAlias() string {
	return p.modelAlias
}

// getBaseURLForModel resolves the embedding model URL for a specific alias
// If modelAlias is empty, falls back to default behavior (configured model or auto-detect)
func (p *DynamicEmbeddingProvider) getBaseURLForModel(modelAlias string) (string, error) {
	// If no override provided, use default behavior
	if modelAlias == "" {
		return p.getBaseURL()
	}

	p.logger.WithField("override_alias", modelAlias).Debug("Using model override for embedding")

	// Get endpoint for the specific model
	endpoint, running := p.instanceGetter.GetModel(modelAlias)

	p.logger.WithFields(logrus.Fields{
		"alias":    modelAlias,
		"endpoint": endpoint,
		"running":  running,
	}).Debug("GetModel result for embedding override")

	if !running {
		return "", fmt.Errorf("embedding model '%s' is not running", modelAlias)
	}
	if endpoint == "" {
		return "", fmt.Errorf("embedding model '%s' has no endpoint", modelAlias)
	}

	p.logger.WithFields(logrus.Fields{
		"alias":    modelAlias,
		"endpoint": endpoint,
	}).Debug("Resolved embedding model from override")

	return endpoint, nil
}

// GenerateEmbeddingWithModel creates embedding using specific model alias
func (p *DynamicEmbeddingProvider) GenerateEmbeddingWithModel(ctx context.Context, text string, modelAlias string) ([]float32, error) {
	// Check for empty text before calling API
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("cannot generate embedding for empty text")
	}

	embeddings, err := p.GenerateEmbeddingsWithModel(ctx, []string{text}, modelAlias)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}
	if embeddings[0] == nil || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned for text")
	}
	return embeddings[0], nil
}

// GenerateEmbeddingsWithModel creates embeddings using specific model alias
func (p *DynamicEmbeddingProvider) GenerateEmbeddingsWithModel(ctx context.Context, texts []string, modelAlias string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Filter out empty texts - TEI requires non-empty inputs
	var filteredTexts []string
	var originalIndices []int
	for i, text := range texts {
		trimmed := strings.TrimSpace(text)
		if trimmed != "" {
			filteredTexts = append(filteredTexts, trimmed)
			originalIndices = append(originalIndices, i)
		}
	}

	if len(filteredTexts) == 0 {
		// All texts were empty, return nil embeddings
		return make([][]float32, len(texts)), nil
	}

	baseURL, err := p.getBaseURLForModel(modelAlias)
	if err != nil {
		return nil, err
	}

	// Use OpenAI-compatible API format - use filtered texts
	reqBody := map[string]any{
		"input": filteredTexts,
		"model": "default",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := baseURL + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	p.logger.WithFields(logrus.Fields{
		"url":        url,
		"count":      len(texts),
		"modelAlias": modelAlias,
	}).Debug("Generating embeddings with model override")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding model returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Log what we got from the API
	if len(result.Data) > 0 {
		p.logger.WithFields(logrus.Fields{
			"embeddings_count": len(result.Data),
			"first_dim":        len(result.Data[0].Embedding),
		}).Debug("Received embeddings from API")
	} else {
		p.logger.Warn("API returned empty embeddings data")
	}

	// Build result array matching filtered texts
	filteredEmbeddings := make([][]float32, len(filteredTexts))
	for _, item := range result.Data {
		if item.Index < len(filteredEmbeddings) {
			filteredEmbeddings[item.Index] = item.Embedding
		}
	}

	// Map back to original indices
	embeddings := make([][]float32, len(texts))
	for i, origIdx := range originalIndices {
		if i < len(filteredEmbeddings) {
			embeddings[origIdx] = filteredEmbeddings[i]
		}
	}

	p.logger.WithFields(logrus.Fields{
		"count":      len(filteredTexts),
		"modelAlias": modelAlias,
	}).Debug("Embeddings with model override generated successfully")

	return embeddings, nil
}
