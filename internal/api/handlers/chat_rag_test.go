// Package handlers provides chat RAG integration tests.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/converter"
	"aigateway/internal/models"
	"aigateway/internal/rag/orchestrator"
)

// Mock RAG Orchestrator
type mockRAGOrchestrator struct {
	queryFunc func(ctx context.Context, req orchestrator.RAGRequest) (*orchestrator.RAGResponse, error)
}

func (m *mockRAGOrchestrator) Query(ctx context.Context, req orchestrator.RAGRequest) (*orchestrator.RAGResponse, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, req)
	}

	// Default mock response
	return &orchestrator.RAGResponse{
		Context: "# Relevant Context\n\nUser Query: test\n\n## Retrieved Information:\n\n### Source 1 (Score: 0.900)\nThis is relevant information from the knowledge base.\n\n",
		SourceChunks: []orchestrator.RetrievedChunk{
			{
				ChunkID:  "chunk1",
				SourceID: "source1",
				Text:     "This is relevant information from the knowledge base.",
				Score:    0.9,
			},
		},
		TotalChunks:   1,
		SearchTime:    50 * time.Millisecond,
		ContextTokens: 30,
	}, nil
}

func TestChatHandler_EnrichMessagesWithRAG_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs in tests

	// Create mock Ollama client
	mockClient := &mockOllamaClient{
		chatFunc: func(ctx context.Context, req models.OllamaChatRequest) (*models.OllamaChatResponse, error) {
			return &models.OllamaChatResponse{
				Model:     "test-model",
				Message:   models.OllamaMessage{Role: "assistant", Content: "Test response"},
				CreatedAt: time.Now(),
			}, nil
		},
	}

	handler := NewChatHandler(cfg, logger, mockClient)

	// Setup RAG orchestrator
	ragOrch := &mockRAGOrchestrator{}
	handler.SetRAGOrchestrator(ragOrch)

	// Create test request
	req := &models.ChatCompletionRequest{
		Model: "test-model",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "What is RAG?"},
		},
		RAGEnabled:   true,
		RAGSourceIDs: []string{"source1"},
		RAGTopK:      5,
		RAGMinScore:  0.7,
		RAGRerank:    true,
	}

	// Test enrichment
	ctx := context.Background()
	err := handler.enrichMessagesWithRAG(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify system message was added
	if len(req.Messages) != 2 {
		t.Fatalf("Expected 2 messages after enrichment, got %d", len(req.Messages))
	}

	// First message should be system message with RAG context
	systemMsg := req.Messages[0]
	if systemMsg.Role != "system" {
		t.Errorf("Expected first message role 'system', got %s", systemMsg.Role)
	}

	contentStr, ok := systemMsg.Content.(string)
	if !ok {
		t.Fatal("Expected system message content to be string")
	}

	if contentStr == "" {
		t.Error("Expected non-empty system message content")
	}

	if !contains(contentStr, "Relevant Context") {
		t.Error("Expected system message to contain RAG context")
	}

	// Second message should be original user message
	userMsg := req.Messages[1]
	if userMsg.Role != "user" {
		t.Errorf("Expected second message role 'user', got %s", userMsg.Role)
	}
}

func TestChatHandler_EnrichMessagesWithRAG_NoOrchestrator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockClient := &mockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	// Don't set RAG orchestrator
	req := &models.ChatCompletionRequest{
		Model:      "test-model",
		Messages:   []models.ChatMessage{{Role: "user", Content: "test"}},
		RAGEnabled: true,
	}

	ctx := context.Background()
	err := handler.enrichMessagesWithRAG(ctx, req)

	if err == nil {
		t.Error("Expected error when orchestrator not initialized")
	}
}

func TestChatHandler_EnrichMessagesWithRAG_NoUserMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockClient := &mockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	ragOrch := &mockRAGOrchestrator{}
	handler.SetRAGOrchestrator(ragOrch)

	// No user messages
	req := &models.ChatCompletionRequest{
		Model:      "test-model",
		Messages:   []models.ChatMessage{{Role: "system", Content: "system prompt"}},
		RAGEnabled: true,
	}

	ctx := context.Background()
	err := handler.enrichMessagesWithRAG(ctx, req)

	if err == nil {
		t.Error("Expected error when no user message found")
	}
}

func TestChatHandler_EnrichMessagesWithRAG_NoResults(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockClient := &mockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	// RAG orchestrator that returns no results
	ragOrch := &mockRAGOrchestrator{
		queryFunc: func(ctx context.Context, req orchestrator.RAGRequest) (*orchestrator.RAGResponse, error) {
			return &orchestrator.RAGResponse{
				Context:       "",
				SourceChunks:  []orchestrator.RetrievedChunk{},
				TotalChunks:   0,
				SearchTime:    10 * time.Millisecond,
				ContextTokens: 0,
			}, nil
		},
	}
	handler.SetRAGOrchestrator(ragOrch)

	req := &models.ChatCompletionRequest{
		Model:      "test-model",
		Messages:   []models.ChatMessage{{Role: "user", Content: "test query"}},
		RAGEnabled: true,
	}

	ctx := context.Background()
	err := handler.enrichMessagesWithRAG(ctx, req)

	// Should not error, just log warning
	if err != nil {
		t.Errorf("Expected no error for empty results, got %v", err)
	}

	// Messages should not be modified (no system message added)
	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message (no enrichment), got %d", len(req.Messages))
	}
}

func TestChatHandler_Completion_WithRAG(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	// Mock Ollama client
	mockClient := &mockOllamaClient{
		chatFunc: func(ctx context.Context, req models.OllamaChatRequest) (*models.OllamaChatResponse, error) {
			return &models.OllamaChatResponse{
				Model:   req.Model,
				Message: models.OllamaMessage{Role: "assistant", Content: "Answer based on RAG context"},
				CreatedAt: time.Now(),
				PromptEvalCount: 100,
				EvalCount:       50,
			}, nil
		},
	}

	handler := NewChatHandler(cfg, logger, mockClient)
	handler.converter = converter.NewSimpleConverter(cfg, logger)

	// Setup RAG orchestrator
	ragOrch := &mockRAGOrchestrator{}
	handler.SetRAGOrchestrator(ragOrch)

	// Create test request with RAG enabled
	chatReq := models.ChatCompletionRequest{
		Model: "test-model",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "What is RAG?"},
		},
		RAGEnabled:   true,
		RAGSourceIDs: []string{"source1"},
		RAGTopK:      5,
		RAGMinScore:  0.7,
	}

	// Create HTTP request
	reqBody, _ := json.Marshal(chatReq)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(reqBody))
	c.Request.Header.Set("Content-Type", "application/json")

	// Call handler
	handler.Completion(c)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Parse response
	var resp models.ChatCompletionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Verify response structure
	if len(resp.Choices) == 0 {
		t.Error("Expected at least one choice")
	}

	if resp.Usage.TotalTokens == 0 {
		t.Error("Expected non-zero token usage")
	}
}

func TestChatHandler_SetRAGOrchestrator(t *testing.T) {
	handler := &ChatHandler{}

	if handler.ragOrchestrator != nil {
		t.Error("Expected nil orchestrator initially")
	}

	ragOrch := &mockRAGOrchestrator{}
	handler.SetRAGOrchestrator(ragOrch)

	if handler.ragOrchestrator == nil {
		t.Error("Expected non-nil orchestrator after SetRAGOrchestrator")
	}
}

// Benchmark tests
func BenchmarkChatHandler_EnrichMessagesWithRAG(b *testing.B) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	mockClient := &mockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	ragOrch := &mockRAGOrchestrator{}
	handler.SetRAGOrchestrator(ragOrch)

	req := &models.ChatCompletionRequest{
		Model:        "test-model",
		Messages:     []models.ChatMessage{{Role: "user", Content: "test query"}},
		RAGEnabled:   true,
		RAGSourceIDs: []string{"source1"},
		RAGTopK:      5,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset messages for each iteration
		req.Messages = []models.ChatMessage{{Role: "user", Content: "test query"}}
		_ = handler.enrichMessagesWithRAG(ctx, req)
	}
}

// Helper types for testing
type mockOllamaClient struct {
	chatFunc   func(ctx context.Context, req models.OllamaChatRequest) (*models.OllamaChatResponse, error)
	streamFunc func(ctx context.Context, req models.OllamaChatRequest) (<-chan models.OllamaChatResponse, <-chan error)
}

func (m *mockOllamaClient) ChatCompletion(ctx context.Context, req models.OllamaChatRequest) (*models.OllamaChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &models.OllamaChatResponse{
		Model:   req.Model,
		Message: models.OllamaMessage{Role: "assistant", Content: "mock response"},
	}, nil
}

func (m *mockOllamaClient) StreamChatCompletion(ctx context.Context, req models.OllamaChatRequest) (<-chan models.OllamaChatResponse, <-chan error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, req)
	}
	respChan := make(chan models.OllamaChatResponse, 1)
	errChan := make(chan error, 1)
	close(respChan)
	close(errChan)
	return respChan, errChan
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}


