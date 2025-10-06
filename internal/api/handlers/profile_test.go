// Package handlers provides profiling tests
// Run with: go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=BenchmarkProfile ./internal/api/handlers
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"ollama-openai-proxy/internal/models"
)

// BenchmarkProfileChatCompletion профилирует chat completion под нагрузкой
func BenchmarkProfileChatCompletion(b *testing.B) {
	router := setupBenchmarkRouter()

	reqBody := models.ChatCompletionRequest{
		Model: "llama2",
		Messages: []models.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Explain quantum computing in simple terms."},
			{Role: "assistant", Content: "Quantum computing uses quantum bits..."},
			{Role: "user", Content: "Can you explain more about superposition?"},
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
	}
}

// BenchmarkProfileEmbeddings профилирует embeddings под нагрузкой
func BenchmarkProfileEmbeddings(b *testing.B) {
	router := setupBenchmarkRouter()

	// Batch embeddings
	inputs := make([]string, 20)
	for i := range inputs {
		inputs[i] = "This is a long test sentence for embedding generation with enough content to be realistic and representative of actual usage patterns in production environments."
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
	}
}

// BenchmarkProfileMixed профилирует смешанную нагрузку
func BenchmarkProfileMixed(b *testing.B) {
	router := setupBenchmarkRouter()

	// Подготавливаем запросы
	chatBody := prepareChatCompletionRequest()
	embedBody := prepareEmbeddingRequest()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Chat completion
		chatReq := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(chatBody))
		chatReq.Header.Set("Content-Type", "application/json")
		chatW := httptest.NewRecorder()
		router.ServeHTTP(chatW, chatReq)

		// Embeddings
		embedReq := httptest.NewRequest("POST", "/v1/embeddings", bytes.NewReader(embedBody))
		embedReq.Header.Set("Content-Type", "application/json")
		embedW := httptest.NewRecorder()
		router.ServeHTTP(embedW, embedReq)

		// Models list
		modelsReq := httptest.NewRequest("GET", "/v1/models", nil)
		modelsW := httptest.NewRecorder()
		router.ServeHTTP(modelsW, modelsReq)
	}
}

func prepareChatCompletionRequest() []byte {
	reqBody := models.ChatCompletionRequest{
		Model: "llama2",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello, world!"},
		},
	}
	jsonBody, _ := json.Marshal(reqBody)
	return jsonBody
}

func prepareEmbeddingRequest() []byte {
	reqBody := models.EmbeddingRequest{
		Model: "llama2",
		Input: "Test embedding sentence",
	}
	jsonBody, _ := json.Marshal(reqBody)
	return jsonBody
}
