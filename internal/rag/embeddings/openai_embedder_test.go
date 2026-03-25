package embeddings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger returns a logger with FatalLevel to suppress output during tests.
func newTestLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.FatalLevel)
	return l
}

// makeEmbeddingResponse builds an OpenAI-format embedding response JSON.
func makeEmbeddingResponse(model string, embeddings [][]float64) openaiEmbeddingResponse {
	resp := openaiEmbeddingResponse{
		Object: "list",
		Model:  model,
	}
	totalTokens := 0
	for i, emb := range embeddings {
		resp.Data = append(resp.Data, struct {
			Object    string    `json:"object"`
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		}{
			Object:    "embedding",
			Index:     i,
			Embedding: emb,
		})
		totalTokens += 5
	}
	resp.Usage.PromptTokens = totalTokens
	resp.Usage.TotalTokens = totalTokens
	return resp
}

func TestNewOpenAIEmbedder(t *testing.T) {
	cfg := EmbedderConfig{
		Provider:     "openai",
		BaseURL:      "http://localhost:11434",
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      30,
		MaxBatchSize: 100,
	}
	logger := newTestLogger()

	embedder := NewOpenAIEmbedder(cfg, logger)

	require.NotNil(t, embedder)
	assert.Equal(t, "openai", embedder.Name())
	assert.Equal(t, "text-embedding-3-small", embedder.GetDefaultModel())
	assert.NotNil(t, embedder.client)
}

func TestOpenAIEmbedder_GetDimensions(t *testing.T) {
	cfg := EmbedderConfig{
		Dimensions: 768, // fallback
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	tests := []struct {
		name     string
		model    string
		expected int
	}{
		{"text-embedding-3-small", "text-embedding-3-small", 1536},
		{"text-embedding-3-large", "text-embedding-3-large", 3072},
		{"text-embedding-ada-002", "text-embedding-ada-002", 1536},
		{"mxbai-embed-large", "mxbai-embed-large", 1024},
		{"nomic-embed-text", "nomic-embed-text", 768},
		{"all-minilm", "all-minilm", 384},
		{"bge-m3", "bge-m3", 1024},
		{"unknown model falls back to config", "unknown-model-xyz", 768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dims, err := embedder.GetDimensions(tt.model)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, dims)
		})
	}
}

func TestOpenAIEmbedder_GetDefaultModel(t *testing.T) {
	cfg := EmbedderConfig{
		Model: "text-embedding-3-large",
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	assert.Equal(t, "text-embedding-3-large", embedder.GetDefaultModel())
}

func TestOpenAIEmbedder_Embed_Success(t *testing.T) {
	expectedVector := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	apiKey := "test-api-key-12345"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify HTTP method and path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/embeddings", r.URL.Path)

		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))

		// Parse and verify request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var reqBody openaiEmbeddingRequest
		err = json.Unmarshal(body, &reqBody)
		require.NoError(t, err)

		assert.Equal(t, "text-embedding-3-small", reqBody.Model)
		assert.Equal(t, "Hello, world!", reqBody.Input)

		// Return OpenAI-format response
		resp := makeEmbeddingResponse("text-embedding-3-small", [][]float64{expectedVector})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := EmbedderConfig{
		BaseURL:      server.URL,
		APIKey:       apiKey,
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      10,
		MaxBatchSize: 100,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	result, err := embedder.Embed(context.Background(), EmbeddingRequest{
		Text:  "Hello, world!",
		Model: "text-embedding-3-small",
		Metadata: map[string]any{
			"source": "test",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, expectedVector, result.Vector)
	assert.Equal(t, len(expectedVector), result.Dimensions)
	assert.Equal(t, "text-embedding-3-small", result.Model)
	assert.Equal(t, "test", result.Metadata["source"])
}

func TestOpenAIEmbedder_Embed_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": {"message": "internal server error", "type": "server_error"}}`))
	}))
	defer server.Close()

	cfg := EmbedderConfig{
		BaseURL:      server.URL,
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      10,
		MaxBatchSize: 100,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	result, err := embedder.Embed(context.Background(), EmbeddingRequest{
		Text:  "test text",
		Model: "text-embedding-3-small",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error 500")
}

func TestOpenAIEmbedder_Embed_DefaultModel(t *testing.T) {
	var receivedModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var reqBody openaiEmbeddingRequest
		err = json.Unmarshal(body, &reqBody)
		require.NoError(t, err)

		receivedModel = reqBody.Model

		resp := makeEmbeddingResponse(reqBody.Model, [][]float64{{0.1, 0.2, 0.3}})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := EmbedderConfig{
		BaseURL:      server.URL,
		Model:        "text-embedding-3-large",
		Dimensions:   3072,
		Timeout:      10,
		MaxBatchSize: 100,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	// Send request with empty Model — should use default
	result, err := embedder.Embed(context.Background(), EmbeddingRequest{
		Text:  "test with default model",
		Model: "", // intentionally empty
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "text-embedding-3-large", receivedModel)
	assert.Equal(t, "text-embedding-3-large", result.Model)
}

func TestOpenAIEmbedder_EmbedBatch_Success(t *testing.T) {
	apiKey := "batch-test-key"
	texts := []string{"first text", "second text", "third text"}
	vectors := [][]float64{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
		{0.7, 0.8, 0.9},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer "+apiKey, r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var reqBody openaiEmbeddingRequest
		err = json.Unmarshal(body, &reqBody)
		require.NoError(t, err)

		assert.Equal(t, "text-embedding-3-small", reqBody.Model)

		// Input should be an array of strings
		inputSlice, ok := reqBody.Input.([]any)
		require.True(t, ok, "input should be a slice")
		assert.Len(t, inputSlice, len(texts))

		resp := makeEmbeddingResponse("text-embedding-3-small", vectors)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := EmbedderConfig{
		BaseURL:      server.URL,
		APIKey:       apiKey,
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      10,
		MaxBatchSize: 100,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	result, err := embedder.EmbedBatch(context.Background(), BatchEmbeddingRequest{
		Texts: texts,
		Model: "text-embedding-3-small",
		Metadata: map[string]any{
			"batch": true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Embeddings, 3)
	assert.Equal(t, "text-embedding-3-small", result.Model)
	assert.Equal(t, 15, result.TotalTokens) // 3 embeddings * 5 tokens each

	for i, emb := range result.Embeddings {
		assert.Equal(t, vectors[i], emb.Vector)
		assert.Equal(t, len(vectors[i]), emb.Dimensions)
		assert.Equal(t, "text-embedding-3-small", emb.Model)
		assert.Equal(t, true, emb.Metadata["batch"])
	}
}

func TestOpenAIEmbedder_EmbedBatch_Empty(t *testing.T) {
	cfg := EmbedderConfig{
		BaseURL:      "http://should-not-be-called",
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      10,
		MaxBatchSize: 100,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	result, err := embedder.EmbedBatch(context.Background(), BatchEmbeddingRequest{
		Texts: []string{},
		Model: "text-embedding-3-small",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Embeddings)
	assert.Equal(t, 0, result.TotalTokens)
}

func TestOpenAIEmbedder_EmbedBatch_LargeBatch(t *testing.T) {
	var callCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		var reqBody openaiEmbeddingRequest
		err = json.Unmarshal(body, &reqBody)
		require.NoError(t, err)

		// Determine how many texts were sent in this batch
		inputSlice, ok := reqBody.Input.([]any)
		require.True(t, ok, "input should be a slice")

		// Generate response vectors for each input
		var batchVectors [][]float64
		for range inputSlice {
			batchVectors = append(batchVectors, []float64{0.1, 0.2, 0.3})
		}

		resp := makeEmbeddingResponse("text-embedding-3-small", batchVectors)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	maxBatch := 5
	cfg := EmbedderConfig{
		BaseURL:      server.URL,
		Model:        "text-embedding-3-small",
		Dimensions:   1536,
		Timeout:      10,
		MaxBatchSize: maxBatch,
	}
	embedder := NewOpenAIEmbedder(cfg, newTestLogger())

	// Create 13 texts — should result in 3 API calls (5 + 5 + 3)
	totalTexts := 13
	texts := make([]string, totalTexts)
	for i := range totalTexts {
		texts[i] = "text number " + string(rune('A'+i))
	}

	result, err := embedder.EmbedBatch(context.Background(), BatchEmbeddingRequest{
		Texts: texts,
		Model: "text-embedding-3-small",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Embeddings, totalTexts)
	assert.Equal(t, int32(3), callCount.Load(), "should split into 3 API calls for 13 items with batch size 5")
	assert.Equal(t, "text-embedding-3-small", result.Model)

	// Verify all embeddings have correct vector
	for _, emb := range result.Embeddings {
		assert.Equal(t, []float64{0.1, 0.2, 0.3}, emb.Vector)
		assert.Equal(t, 3, emb.Dimensions)
	}
}

func TestDefaultEmbedderConfig(t *testing.T) {
	cfg := DefaultEmbedderConfig()

	assert.Equal(t, "openai", cfg.Provider)
	assert.NotEmpty(t, cfg.BaseURL)
	assert.Equal(t, "text-embedding-3-small", cfg.Model)
	assert.Equal(t, 1536, cfg.Dimensions)
	assert.Equal(t, 30, cfg.Timeout)
	assert.Equal(t, 100, cfg.MaxBatchSize)
	assert.Equal(t, 10.0, cfg.RateLimit)
}
