// Package rag provides RAG capabilities for GitLab code review
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	reqBody := map[string]interface{}{
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

	if p.modelAlias != "" {
		// Use explicitly configured model
		endpoint, running = p.instanceGetter.GetModel(p.modelAlias)
		modelName = p.modelAlias
		if !running {
			return "", fmt.Errorf("embedding model '%s' is not running", p.modelAlias)
		}
	} else {
		// Auto-detect any running embedding model
		var found bool
		modelName, endpoint, found = p.instanceGetter.GetRunningEmbeddingModel()
		if !found {
			return "", fmt.Errorf("no running embedding model found (start a TEI model via Admin -> Models)")
		}
		p.logger.WithField("model", modelName).Debug("Auto-detected running embedding model")
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
	reqBody := map[string]interface{}{
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

