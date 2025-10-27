// Package handlers provides interfaces for handlers
package handlers

import (
	"context"

	"aigateway/internal/client/ollama"
)

// OllamaClientInterface определяет интерфейс для Ollama клиентов в handlers
type OllamaClientInterface interface {
	Health(ctx context.Context) error
	GetModels(ctx context.Context) (*ollama.ModelsResponse, error)
	ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error)
	ChatCompletionStream(ctx context.Context, req *ollama.ChatRequest) (<-chan *ollama.ChatResponse, <-chan error)
	Generate(ctx context.Context, req *ollama.GenerateRequest) (*ollama.GenerateResponse, error)
	ShowModel(ctx context.Context, modelName string) (*ollama.ShowResponse, error)
	PullModel(ctx context.Context, modelName string) (*ollama.PullResponse, error)
	Embed(ctx context.Context, req *ollama.EmbedRequest) (*ollama.EmbedResponse, error)
	Embeddings(ctx context.Context, req *ollama.EmbeddingsRequest) (*ollama.EmbeddingsResponse, error)
	GetEmbeddings(ctx context.Context, model, text string) (*ollama.EmbeddingsResponse, error)
	IsModelAvailable(ctx context.Context, modelName string) (bool, error)
	Close()
}

