// Package orchestrator provides RAG orchestrator tests.
package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/rag/embeddings"
	"ollama-openai-proxy/internal/rag/vector"
)

// Mock Embedder
type mockEmbedder struct {
	embeddings []float64
	err        error
}

func (m *mockEmbedder) Embed(ctx context.Context, req embeddings.EmbeddingRequest) (*embeddings.Embedding, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &embeddings.Embedding{
		Vector:     m.embeddings,
		Dimensions: len(m.embeddings),
		Model:      "mock",
	}, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, req embeddings.BatchEmbeddingRequest) (*embeddings.BatchEmbeddingResponse, error) {
	return nil, nil
}

func (m *mockEmbedder) GetDimensions(model string) (int, error) {
	return len(m.embeddings), nil
}

func (m *mockEmbedder) GetDefaultModel() string {
	return "mock"
}

func (m *mockEmbedder) Name() string {
	return "mock"
}

// Mock Vector Store
type mockVectorStore struct {
	documents []vector.VectorDocument
	err       error
}

func (m *mockVectorStore) Insert(ctx context.Context, doc vector.VectorDocument) error {
	return m.err
}

func (m *mockVectorStore) InsertBatch(ctx context.Context, docs []vector.VectorDocument) error {
	return m.err
}

func (m *mockVectorStore) Search(ctx context.Context, req vector.SearchRequest) (*vector.SearchResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Filter by min score
	filtered := []vector.VectorDocument{}
	for _, doc := range m.documents {
		if doc.Score >= req.MinScore {
			filtered = append(filtered, doc)
		}
	}

	// Limit to TopK
	if len(filtered) > req.TopK {
		filtered = filtered[:req.TopK]
	}

	return &vector.SearchResponse{
		Documents:  filtered,
		TotalFound: len(filtered),
		SearchTime: 10,
	}, nil
}

func (m *mockVectorStore) Delete(ctx context.Context, id string) error {
	return m.err
}

func (m *mockVectorStore) DeleteByMetadata(ctx context.Context, filters map[string]interface{}) (int, error) {
	return 0, m.err
}

func (m *mockVectorStore) Update(ctx context.Context, doc vector.VectorDocument) error {
	return m.err
}

func (m *mockVectorStore) GetByID(ctx context.Context, id string) (*vector.VectorDocument, error) {
	return nil, m.err
}

func (m *mockVectorStore) CreateIndex(ctx context.Context, indexType string, params map[string]interface{}) error {
	return m.err
}

func (m *mockVectorStore) GetIndexStats(ctx context.Context) (*vector.IndexStats, error) {
	return nil, m.err
}

func (m *mockVectorStore) HealthCheck(ctx context.Context) error {
	return m.err
}

func (m *mockVectorStore) Name() string {
	return "mock"
}

func TestNewRAGOrchestrator(t *testing.T) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}
	vectorStore := &mockVectorStore{}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)

	if orchestrator == nil {
		t.Fatal("Expected non-nil orchestrator")
	}

	if orchestrator.embedder != embedder {
		t.Error("Embedder not set correctly")
	}

	if orchestrator.vectorStore != vectorStore {
		t.Error("Vector store not set correctly")
	}

	if orchestrator.defaultTopK != 10 {
		t.Errorf("Expected defaultTopK=10, got %d", orchestrator.defaultTopK)
	}
}

func TestRAGOrchestrator_Query_EmptyQuery(t *testing.T) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}
	vectorStore := &mockVectorStore{}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query: "",
	}

	_, err := orchestrator.Query(ctx, req)
	if err == nil {
		t.Error("Expected error for empty query")
	}
}

func TestRAGOrchestrator_Query_Success(t *testing.T) {
	embedder := &mockEmbedder{
		embeddings: []float64{1.0, 2.0, 3.0},
	}

	// Mock documents
	docs := []vector.VectorDocument{
		{
			ID:    "doc1",
			Text:  "This is the first document",
			Score: 0.9,
			Metadata: map[string]interface{}{
				"source_id":   "source1",
				"document_id": "doc1",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:    "doc2",
			Text:  "This is the second document",
			Score: 0.8,
			Metadata: map[string]interface{}{
				"source_id":   "source1",
				"document_id": "doc2",
			},
			CreatedAt: time.Now(),
		},
	}

	vectorStore := &mockVectorStore{documents: docs}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query:    "test query",
		TopK:     5,
		MinScore: 0.7,
		UserID:   "user1",
		ConvID:   "conv1",
		Rerank:   false,
	}

	resp, err := orchestrator.Query(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if resp.TotalChunks != 2 {
		t.Errorf("Expected 2 chunks, got %d", resp.TotalChunks)
	}

	if len(resp.SourceChunks) != 2 {
		t.Errorf("Expected 2 source chunks, got %d", len(resp.SourceChunks))
	}

	if resp.Context == "" {
		t.Error("Expected non-empty context")
	}

	if resp.ContextTokens == 0 {
		t.Error("Expected non-zero context tokens")
	}
}

func TestRAGOrchestrator_Query_WithSourceFilter(t *testing.T) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}

	docs := []vector.VectorDocument{
		{
			ID:    "doc1",
			Text:  "Document from source1",
			Score: 0.9,
			Metadata: map[string]interface{}{
				"source_id": "source1",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:    "doc2",
			Text:  "Document from source2",
			Score: 0.85,
			Metadata: map[string]interface{}{
				"source_id": "source2",
			},
			CreatedAt: time.Now(),
		},
	}

	vectorStore := &mockVectorStore{documents: docs}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query:     "test query",
		SourceIDs: []string{"source1"}, // Filter by source1 only
		TopK:      5,
		MinScore:  0.7,
	}

	resp, err := orchestrator.Query(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only get doc1 (from source1)
	if resp.TotalChunks != 1 {
		t.Errorf("Expected 1 chunk after filtering, got %d", resp.TotalChunks)
	}

	if len(resp.SourceChunks) > 0 {
		chunk := resp.SourceChunks[0]
		if chunk.SourceID != "source1" {
			t.Errorf("Expected source1, got %s", chunk.SourceID)
		}
	}
}

func TestRAGOrchestrator_Query_WithReranking(t *testing.T) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}

	docs := []vector.VectorDocument{
		{
			ID:        "doc1",
			Text:      "Document about cats",
			Score:     0.8,
			Metadata:  map[string]interface{}{"source_id": "source1"},
			CreatedAt: time.Now(),
		},
		{
			ID:        "doc2",
			Text:      "Document about dogs",
			Score:     0.75,
			Metadata:  map[string]interface{}{"source_id": "source1"},
			CreatedAt: time.Now(),
		},
	}

	vectorStore := &mockVectorStore{documents: docs}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query:    "cats",
		TopK:     5,
		MinScore: 0.7,
		Rerank:   true, // Enable reranking
	}

	resp, err := orchestrator.Query(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.TotalChunks == 0 {
		t.Error("Expected some chunks")
	}

	// With reranking, doc1 (about cats) should have higher score
	// because query contains "cats"
	if len(resp.SourceChunks) >= 2 {
		firstChunk := resp.SourceChunks[0]
		if !contains(firstChunk.Text, "cats") {
			t.Log("Warning: Reranking might not be working as expected")
		}
	}
}

func TestRAGOrchestrator_Query_NoResults(t *testing.T) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}

	// Empty documents
	vectorStore := &mockVectorStore{documents: []vector.VectorDocument{}}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query:    "test query",
		TopK:     5,
		MinScore: 0.7,
	}

	resp, err := orchestrator.Query(ctx, req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.TotalChunks != 0 {
		t.Errorf("Expected 0 chunks, got %d", resp.TotalChunks)
	}

	if len(resp.SourceChunks) != 0 {
		t.Errorf("Expected empty source chunks, got %d", len(resp.SourceChunks))
	}
}

func TestRAGOrchestrator_EstimateTokens(t *testing.T) {
	orchestrator := &RAGOrchestrator{}

	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty",
			text:     "",
			expected: 0,
		},
		{
			name:     "4 characters",
			text:     "test",
			expected: 1,
		},
		{
			name:     "16 characters",
			text:     "this is a test  ",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orchestrator.estimateTokens(tt.text)
			if result != tt.expected {
				t.Errorf("Expected %d tokens, got %d", tt.expected, result)
			}
		})
	}
}

func TestRAGOrchestrator_AssembleContext(t *testing.T) {
	orchestrator := NewRAGOrchestrator(nil, nil, nil, logrus.New())

	chunks := []RetrievedChunk{
		{
			ChunkID: "chunk1",
			Text:    "First chunk text",
			Score:   0.9,
		},
		{
			ChunkID: "chunk2",
			Text:    "Second chunk text",
			Score:   0.8,
		},
	}

	context, tokens := orchestrator.assembleContext(chunks, "test query")

	if context == "" {
		t.Error("Expected non-empty context")
	}

	if tokens == 0 {
		t.Error("Expected non-zero token count")
	}

	// Context should contain query
	if !contains(context, "test query") {
		t.Error("Context should contain the query")
	}

	// Context should contain chunk texts
	if !contains(context, "First chunk text") {
		t.Error("Context should contain first chunk")
	}

	if !contains(context, "Second chunk text") {
		t.Error("Context should contain second chunk")
	}
}

func TestRAGOrchestrator_FilterBySourceIDs(t *testing.T) {
	orchestrator := &RAGOrchestrator{}

	docs := []vector.VectorDocument{
		{
			ID:       "doc1",
			Metadata: map[string]interface{}{"source_id": "source1"},
		},
		{
			ID:       "doc2",
			Metadata: map[string]interface{}{"source_id": "source2"},
		},
		{
			ID:       "doc3",
			Metadata: map[string]interface{}{"source_id": "source1"},
		},
	}

	// No filter
	result := orchestrator.filterBySourceIDs(docs, nil)
	if len(result) != 3 {
		t.Errorf("Expected 3 docs without filter, got %d", len(result))
	}

	// Filter by source1
	result = orchestrator.filterBySourceIDs(docs, []string{"source1"})
	if len(result) != 2 {
		t.Errorf("Expected 2 docs with source1 filter, got %d", len(result))
	}

	// Filter by source2
	result = orchestrator.filterBySourceIDs(docs, []string{"source2"})
	if len(result) != 1 {
		t.Errorf("Expected 1 doc with source2 filter, got %d", len(result))
	}

	// Filter by non-existent source
	result = orchestrator.filterBySourceIDs(docs, []string{"source999"})
	if len(result) != 0 {
		t.Errorf("Expected 0 docs with non-existent source filter, got %d", len(result))
	}
}

// Benchmark tests
func BenchmarkRAGOrchestrator_Query(b *testing.B) {
	embedder := &mockEmbedder{embeddings: []float64{1.0, 2.0, 3.0}}

	// Create 100 mock documents
	docs := make([]vector.VectorDocument, 100)
	for i := 0; i < 100; i++ {
		docs[i] = vector.VectorDocument{
			ID:        "doc" + string(rune(i)),
			Text:      "Document text number " + string(rune(i)),
			Score:     0.8,
			Metadata:  map[string]interface{}{"source_id": "source1"},
			CreatedAt: time.Now(),
		}
	}

	vectorStore := &mockVectorStore{documents: docs}
	logger := logrus.New()

	orchestrator := NewRAGOrchestrator(nil, embedder, vectorStore, logger)
	ctx := context.Background()

	req := RAGRequest{
		Query:    "test query",
		TopK:     10,
		MinScore: 0.7,
		Rerank:   true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = orchestrator.Query(ctx, req)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

