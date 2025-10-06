// Package handlers provides benchmark tests for API handlers
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

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
)

// MockOllamaClient для benchmark тестов
type MockOllamaClient struct{}

func (m *MockOllamaClient) Health(ctx context.Context) error {
	return nil
}

func (m *MockOllamaClient) GetModels(ctx context.Context) (*ollama.ModelsResponse, error) {
	return &ollama.ModelsResponse{
		Models: []ollama.Model{
			{Name: "llama2", Size: 3825819519},
			{Name: "mistral", Size: 4109865159},
		},
	}, nil
}

func (m *MockOllamaClient) ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error) {
	return &ollama.ChatResponse{
		Model:     req.Model,
		CreatedAt: time.Now(),
		Message: ollama.ChatMessage{
			Role:    "assistant",
			Content: "This is a test response from the mock Ollama client.",
		},
		Done:               true,
		TotalDuration:      1000000000,
		LoadDuration:       100000000,
		PromptEvalCount:    10,
		PromptEvalDuration: 200000000,
		EvalCount:          20,
		EvalDuration:       700000000,
	}, nil
}

func (m *MockOllamaClient) ChatCompletionStream(ctx context.Context, req *ollama.ChatRequest) (<-chan *ollama.ChatResponse, <-chan error) {
	responseChan := make(chan *ollama.ChatResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		// Симулируем streaming ответ
		chunks := []string{"Hello", " ", "world", "!", " ", "This", " ", "is", " ", "streaming", "."}
		for _, chunk := range chunks {
			responseChan <- &ollama.ChatResponse{
				Model:     req.Model,
				CreatedAt: time.Now(),
				Message: ollama.ChatMessage{
					Role:    "assistant",
					Content: chunk,
				},
				Done: false,
			}
		}

		// Финальный chunk
		responseChan <- &ollama.ChatResponse{
			Model:     req.Model,
			CreatedAt: time.Now(),
			Message: ollama.ChatMessage{
				Role:    "assistant",
				Content: "",
			},
			Done: true,
		}
	}()

	return responseChan, errorChan
}

func (m *MockOllamaClient) Generate(ctx context.Context, req *ollama.GenerateRequest) (*ollama.GenerateResponse, error) {
	return &ollama.GenerateResponse{
		Model:    req.Model,
		Response: "Generated text response",
		Done:     true,
	}, nil
}

func (m *MockOllamaClient) ShowModel(ctx context.Context, modelName string) (*ollama.ShowResponse, error) {
	return &ollama.ShowResponse{
		Modelfile: "Modelfile content",
	}, nil
}

func (m *MockOllamaClient) PullModel(ctx context.Context, modelName string) (*ollama.PullResponse, error) {
	return &ollama.PullResponse{
		Status: "success",
	}, nil
}

func (m *MockOllamaClient) Embed(ctx context.Context, req *ollama.EmbedRequest) (*ollama.EmbedResponse, error) {
	// Создаем mock embedding
	embedding := make([]float32, 384) // Типичный размер для небольших моделей
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}

	return &ollama.EmbedResponse{
		Model:           req.Model,
		Embeddings:      [][]float32{embedding},
		TotalDuration:   1000000000,
		LoadDuration:    100000000,
		PromptEvalCount: 10,
	}, nil
}

func (m *MockOllamaClient) Embeddings(ctx context.Context, req *ollama.EmbeddingsRequest) (*ollama.EmbeddingsResponse, error) {
	embedding := make([]float64, 384)
	for i := range embedding {
		embedding[i] = float64(i) * 0.001
	}

	return &ollama.EmbeddingsResponse{
		Embedding: embedding,
	}, nil
}

func (m *MockOllamaClient) GetEmbeddings(ctx context.Context, model, text string) (*ollama.EmbeddingsResponse, error) {
	return m.Embeddings(ctx, &ollama.EmbeddingsRequest{
		Model:  model,
		Prompt: text,
	})
}

func (m *MockOllamaClient) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	return true, nil
}

func (m *MockOllamaClient) Close() {}

// setupBenchmarkRouter создает роутер для benchmark тестов
func setupBenchmarkRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Ollama: config.OllamaConfig{
			Timeout: 120,
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Минимальное логирование для бенчмарков

	mockClient := &MockOllamaClient{}

	router := gin.New()

	// Chat handler
	chatHandler := NewChatHandler(cfg, logger, mockClient)
	router.POST("/v1/chat/completions", chatHandler.Completion)

	// Models handler
	modelsHandler := NewModelsHandler(cfg, logger, mockClient)
	router.GET("/v1/models", modelsHandler.List)

	// Embeddings handler
	embeddingsHandler := NewEmbeddingsHandler(cfg, logger, mockClient)
	router.POST("/v1/embeddings", embeddingsHandler.HandleEmbeddings)

	// Completions handler
	completionsHandler := NewCompletionsHandler(cfg, logger, mockClient)
	router.POST("/v1/completions", completionsHandler.HandleCompletions)

	return router
}

// BenchmarkChatCompletionHandler тестирует производительность chat completion endpoint
func BenchmarkChatCompletionHandler(b *testing.B) {
	router := setupBenchmarkRouter()

	reqBody := models.ChatCompletionRequest{
		Model: "llama2",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello, how are you?"},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkChatCompletionHandlerLargeContext тестирует с большим контекстом
func BenchmarkChatCompletionHandlerLargeContext(b *testing.B) {
	router := setupBenchmarkRouter()

	// Создаем большой контекст (10 сообщений)
	messages := make([]models.ChatMessage, 10)
	for i := 0; i < 10; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		messages[i] = models.ChatMessage{
			Role:    role,
			Content: "This is a test message with some content to simulate a realistic conversation. Let's add more text to make it longer and more representative of actual usage.",
		}
	}

	reqBody := models.ChatCompletionRequest{
		Model:    "llama2",
		Messages: messages,
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkModelsListHandler тестирует производительность models list endpoint
func BenchmarkModelsListHandler(b *testing.B) {
	router := setupBenchmarkRouter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/v1/models", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkEmbeddingsHandler тестирует производительность embeddings endpoint
func BenchmarkEmbeddingsHandler(b *testing.B) {
	router := setupBenchmarkRouter()

	reqBody := models.EmbeddingRequest{
		Model: "llama2",
		Input: "This is a test sentence for embedding generation.",
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/embeddings", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkEmbeddingsHandlerBatch тестирует batch embeddings
func BenchmarkEmbeddingsHandlerBatch(b *testing.B) {
	router := setupBenchmarkRouter()

	// Batch из 10 текстов
	inputs := make([]string, 10)
	for i := range inputs {
		inputs[i] = "This is test sentence number " + string(rune(i+'0')) + " for batch embedding generation."
	}

	reqBody := models.EmbeddingRequest{
		Model: "llama2",
		Input: inputs,
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/embeddings", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkCompletionsHandler тестирует производительность legacy completions endpoint
func BenchmarkCompletionsHandler(b *testing.B) {
	router := setupBenchmarkRouter()

	reqBody := models.CompletionRequest{
		Model:  "llama2",
		Prompt: "Once upon a time",
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/completions", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkParallelChatCompletion тестирует параллельные запросы
func BenchmarkParallelChatCompletion(b *testing.B) {
	router := setupBenchmarkRouter()

	reqBody := models.ChatCompletionRequest{
		Model: "llama2",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello, how are you?"},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}
