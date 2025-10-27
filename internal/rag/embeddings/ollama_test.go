// Package embeddings provides Ollama embedder tests.
package embeddings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestNewOllamaEmbedder(t *testing.T) {
	config := DefaultEmbedderConfig()
	logger := logrus.New()

	embedder := NewOllamaEmbedder(config, logger)

	if embedder == nil {
		t.Fatal("Expected non-nil embedder")
	}

	if embedder.Name() != "ollama" {
		t.Errorf("Expected name 'ollama', got %s", embedder.Name())
	}

	if embedder.GetDefaultModel() != config.Model {
		t.Errorf("Expected default model %s, got %s", config.Model, embedder.GetDefaultModel())
	}
}

func TestOllamaEmbedder_GetDimensions(t *testing.T) {
	embedder := NewOllamaEmbedder(DefaultEmbedderConfig(), logrus.New())

	tests := []struct {
		model      string
		expected   int
	}{
		{"mxbai-embed-large", 1024},
		{"nomic-embed-text", 768},
		{"all-minilm", 384},
		{"unknown-model", 1024}, // Falls back to config dimensions
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			dims, err := embedder.GetDimensions(tt.model)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if dims != tt.expected {
				t.Errorf("Expected %d dimensions, got %d", tt.expected, dims)
			}
		})
	}
}

func TestOllamaEmbedder_Embed_Success(t *testing.T) {
	// Mock Ollama server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if !strings.HasSuffix(r.URL.Path, "/api/embeddings") {
			t.Errorf("Expected /api/embeddings endpoint, got %s", r.URL.Path)
		}

		// Return mock embedding
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3, 0.4, 0.5]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL
	config.Timeout = 5

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	req := EmbeddingRequest{
		Text:  "Test text for embedding",
		Model: "test-model",
		Metadata: map[string]interface{}{
			"test": true,
		},
	}

	embedding, err := embedder.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if embedding == nil {
		t.Fatal("Expected non-nil embedding")
	}

	if len(embedding.Vector) != 5 {
		t.Errorf("Expected 5-dimensional vector, got %d", len(embedding.Vector))
	}

	if embedding.Dimensions != 5 {
		t.Errorf("Expected dimensions=5, got %d", embedding.Dimensions)
	}

	if embedding.Model != "test-model" {
		t.Errorf("Expected model='test-model', got %s", embedding.Model)
	}

	// Verify vector values
	expected := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	for i, v := range embedding.Vector {
		if v != expected[i] {
			t.Errorf("Vector[%d]: expected %f, got %f", i, expected[i], v)
		}
	}
}

func TestOllamaEmbedder_Embed_ServerError(t *testing.T) {
	// Mock server that returns error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	req := EmbeddingRequest{
		Text:  "Test text",
		Model: "test-model",
	}

	_, err := embedder.Embed(ctx, req)
	if err == nil {
		t.Error("Expected error for server error")
	}
}

func TestOllamaEmbedder_Embed_DefaultModel(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL
	config.Model = "default-model"

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	// Don't specify model in request
	req := EmbeddingRequest{
		Text: "Test text",
	}

	embedding, err := embedder.Embed(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should use default model
	if embedding.Model != "default-model" {
		t.Errorf("Expected default model to be used, got %s", embedding.Model)
	}
}

func TestOllamaEmbedder_EmbedBatch_Success(t *testing.T) {
	callCount := 0

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL
	config.MaxBatchSize = 10

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	req := BatchEmbeddingRequest{
		Texts: []string{
			"First text",
			"Second text",
			"Third text",
		},
		Model: "test-model",
	}

	resp, err := embedder.EmbedBatch(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	// Should have 3 embeddings (one per text)
	if len(resp.Embeddings) != 3 {
		t.Errorf("Expected 3 embeddings, got %d", len(resp.Embeddings))
	}

	// Should have made 3 API calls (one per text)
	if callCount != 3 {
		t.Errorf("Expected 3 API calls, got %d", callCount)
	}

	if resp.Model != "test-model" {
		t.Errorf("Expected model='test-model', got %s", resp.Model)
	}

	if resp.TotalTokens == 0 {
		t.Error("Expected non-zero token count")
	}
}

func TestOllamaEmbedder_EmbedBatch_Empty(t *testing.T) {
	config := DefaultEmbedderConfig()
	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	req := BatchEmbeddingRequest{
		Texts: []string{},
		Model: "test-model",
	}

	resp, err := embedder.EmbedBatch(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error for empty batch, got %v", err)
	}

	if len(resp.Embeddings) != 0 {
		t.Errorf("Expected 0 embeddings for empty batch, got %d", len(resp.Embeddings))
	}
}

func TestOllamaEmbedder_EmbedBatch_LargeBatch(t *testing.T) {
	callCount := 0

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL
	config.MaxBatchSize = 5 // Small batch size

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	// Create 12 texts (should be split into 3 batches of 5, 5, 2)
	texts := make([]string, 12)
	for i := 0; i < 12; i++ {
		texts[i] = "Text " + string(rune(i+'0'))
	}

	req := BatchEmbeddingRequest{
		Texts: texts,
		Model: "test-model",
	}

	resp, err := embedder.EmbedBatch(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have 12 embeddings
	if len(resp.Embeddings) != 12 {
		t.Errorf("Expected 12 embeddings, got %d", len(resp.Embeddings))
	}

	// Should have made 12 calls (one per text with batch size 5)
	if callCount != 12 {
		t.Logf("Call count: %d (expected 12 individual calls)", callCount)
	}
}

func TestDefaultEmbedderConfig(t *testing.T) {
	config := DefaultEmbedderConfig()

	if config.Provider != "ollama" {
		t.Errorf("Expected provider 'ollama', got %s", config.Provider)
	}

	if config.BaseURL == "" {
		t.Error("Expected non-empty BaseURL")
	}

	if config.Model == "" {
		t.Error("Expected non-empty default Model")
	}

	if config.Dimensions <= 0 {
		t.Error("Expected positive Dimensions")
	}

	if config.Timeout <= 0 {
		t.Error("Expected positive Timeout")
	}

	if config.MaxBatchSize <= 0 {
		t.Error("Expected positive MaxBatchSize")
	}
}

// Benchmark tests
func BenchmarkOllamaEmbedder_Embed(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	req := EmbeddingRequest{
		Text:  "Benchmark text for embedding generation",
		Model: "test-model",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.Embed(ctx, req)
	}
}

func BenchmarkOllamaEmbedder_EmbedBatch(b *testing.B) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
	}))
	defer mockServer.Close()

	config := DefaultEmbedderConfig()
	config.BaseURL = mockServer.URL

	embedder := NewOllamaEmbedder(config, logrus.New())
	ctx := context.Background()

	texts := []string{
		"First text",
		"Second text",
		"Third text",
		"Fourth text",
		"Fifth text",
	}

	req := BatchEmbeddingRequest{
		Texts: texts,
		Model: "test-model",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedBatch(ctx, req)
	}
}


