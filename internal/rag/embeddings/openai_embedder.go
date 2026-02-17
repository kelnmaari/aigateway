// Package embeddings provides OpenAI-compatible embedding generation.
package embeddings

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

// OpenAIEmbedder generates embeddings using OpenAI-compatible API (POST /v1/embeddings).
// Works with OpenAI, Azure OpenAI, vLLM, TEI, and any OpenAI-compatible endpoint.
type OpenAIEmbedder struct {
	config EmbedderConfig
	client *http.Client
	logger *logrus.Logger
}

// NewOpenAIEmbedder creates a new OpenAI-compatible embedder.
func NewOpenAIEmbedder(config EmbedderConfig, logger *logrus.Logger) *OpenAIEmbedder {
	return &OpenAIEmbedder{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
		logger: logger,
	}
}

// Name returns the embedder name.
func (e *OpenAIEmbedder) Name() string {
	return "openai"
}

// GetDefaultModel returns the default embedding model.
func (e *OpenAIEmbedder) GetDefaultModel() string {
	return e.config.Model
}

// GetDimensions returns the vector dimensions for a given model.
func (e *OpenAIEmbedder) GetDimensions(model string) (int, error) {
	// Known model dimensions
	knownDimensions := map[string]int{
		"text-embedding-3-small":  1536,
		"text-embedding-3-large":  3072,
		"text-embedding-ada-002":  1536,
		"mxbai-embed-large":      1024,
		"nomic-embed-text":       768,
		"all-minilm":             384,
		"bge-m3":                 1024,
	}

	if dims, ok := knownDimensions[model]; ok {
		return dims, nil
	}

	// Fallback to configured dimensions
	return e.config.Dimensions, nil
}

// openaiEmbeddingRequest is the request body for POST /v1/embeddings.
type openaiEmbeddingRequest struct {
	Input          interface{} `json:"input"`                     // string or []string
	Model          string      `json:"model"`
	EncodingFormat string      `json:"encoding_format,omitempty"` // "float" (default)
}

// openaiEmbeddingResponse is the response from POST /v1/embeddings.
type openaiEmbeddingResponse struct {
	Object string `json:"object"` // "list"
	Data   []struct {
		Object    string    `json:"object"` // "embedding"
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// Embed generates an embedding for a single text.
func (e *OpenAIEmbedder) Embed(ctx context.Context, req EmbeddingRequest) (*Embedding, error) {
	model := req.Model
	if model == "" {
		model = e.config.Model
	}

	apiReq := openaiEmbeddingRequest{
		Input: req.Text,
		Model: model,
	}

	resp, err := e.callAPI(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("openai embed: empty response data")
	}

	vector := resp.Data[0].Embedding

	embedding := &Embedding{
		Vector:     vector,
		Dimensions: len(vector),
		Model:      model,
		Metadata:   req.Metadata,
	}

	return embedding, nil
}

// EmbedBatch generates embeddings for a batch of texts.
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, req BatchEmbeddingRequest) (*BatchEmbeddingResponse, error) {
	if len(req.Texts) == 0 {
		return &BatchEmbeddingResponse{
			Embeddings:  []Embedding{},
			Model:       req.Model,
			TotalTokens: 0,
		}, nil
	}

	model := req.Model
	if model == "" {
		model = e.config.Model
	}

	var allEmbeddings []Embedding
	totalTokens := 0

	// Process in batches
	batchSize := e.config.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(req.Texts); i += batchSize {
		end := i + batchSize
		if end > len(req.Texts) {
			end = len(req.Texts)
		}

		batch := req.Texts[i:end]

		apiReq := openaiEmbeddingRequest{
			Input: batch,
			Model: model,
		}

		resp, err := e.callAPI(ctx, apiReq)
		if err != nil {
			return nil, fmt.Errorf("openai embed batch (offset %d): %w", i, err)
		}

		totalTokens += resp.Usage.TotalTokens

		for _, d := range resp.Data {
			allEmbeddings = append(allEmbeddings, Embedding{
				Vector:     d.Embedding,
				Dimensions: len(d.Embedding),
				Model:      model,
				Metadata:   req.Metadata,
			})
		}
	}

	return &BatchEmbeddingResponse{
		Embeddings:  allEmbeddings,
		Model:       model,
		TotalTokens: totalTokens,
	}, nil
}

// callAPI makes the HTTP request to the OpenAI-compatible embeddings endpoint.
func (e *OpenAIEmbedder) callAPI(ctx context.Context, req openaiEmbeddingRequest) (*openaiEmbeddingResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := strings.TrimRight(e.config.BaseURL, "/") + "/v1/embeddings"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add API key if configured
	if e.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	}

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiEmbeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
