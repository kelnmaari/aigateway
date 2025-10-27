// Package model provides interfaces for model manager
package model

import (
	"context"

	"aigateway/internal/client/ollama"
)

// OllamaClientInterface определяет интерфейс для Ollama клиентов
type OllamaClientInterface interface {
	Health(ctx context.Context) error
	GetModels(ctx context.Context) (*ollama.ModelsResponse, error)
	ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error)
	Generate(ctx context.Context, req *ollama.GenerateRequest) (*ollama.GenerateResponse, error)
	ShowModel(ctx context.Context, modelName string) (*ollama.ShowResponse, error)
	PullModel(ctx context.Context, modelName string) (*ollama.PullResponse, error)
	GetEmbeddings(ctx context.Context, model, text string) (*ollama.EmbeddingsResponse, error)
	IsModelAvailable(ctx context.Context, modelName string) (bool, error)
	Close()
}

