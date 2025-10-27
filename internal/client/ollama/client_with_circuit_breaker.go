// Package ollama provides Ollama client with circuit breaker integration
package ollama

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"aigateway/internal/circuit"
	"aigateway/internal/config"
	"aigateway/internal/errors"
)

// ClientWithCircuitBreaker оборачивает Ollama клиент с circuit breaker
type ClientWithCircuitBreaker struct {
	client         *Client
	circuitManager *circuit.Manager
	logger         *logrus.Logger
	enabled        bool
}

// NewClientWithCircuitBreaker создает новый клиент с circuit breaker
func NewClientWithCircuitBreaker(cfg *config.Config, logger *logrus.Logger) (*ClientWithCircuitBreaker, error) {
	// Создаем базовый клиент
	client, err := NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create base Ollama client: %w", err)
	}

	// Создаем circuit manager
	circuitManager := circuit.NewManager(cfg, logger)

	return &ClientWithCircuitBreaker{
		client:         client,
		circuitManager: circuitManager,
		logger:         logger,
		enabled:        cfg.Ollama.CircuitBreaker.Enabled,
	}, nil
}

// Health проверяет доступность Ollama с circuit breaker
func (c *ClientWithCircuitBreaker) Health(ctx context.Context) error {
	if !c.enabled {
		return c.client.Health(ctx)
	}

	breaker := c.circuitManager.GetBreaker("ollama-health")

	operation := func(ctx context.Context) (interface{}, error) {
		return nil, c.client.Health(ctx)
	}

	_, err := breaker.Execute(ctx, operation)
	if err != nil {
		// Если это ошибка circuit breaker, преобразуем её
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return err
	}

	return nil
}

// GetModels получает список моделей с circuit breaker
func (c *ClientWithCircuitBreaker) GetModels(ctx context.Context) (*ModelsResponse, error) {
	if !c.enabled {
		return c.client.GetModels(ctx)
	}

	breaker := c.circuitManager.GetBreaker("ollama-models")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.GetModels(ctx)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		// Если это ошибка circuit breaker, преобразуем её
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*ModelsResponse), nil
}

// ChatCompletion отправляет chat completion запрос с circuit breaker
func (c *ClientWithCircuitBreaker) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if !c.enabled {
		return c.client.ChatCompletion(ctx, req)
	}

	breaker := c.circuitManager.GetBreaker("ollama-chat")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.ChatCompletion(ctx, req)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		// Если это ошибка circuit breaker, преобразуем её
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*ChatResponse), nil
}

// ChatCompletionStream выполняет streaming chat completion
// Circuit breaker не применяется для streaming, так как это долгое соединение
func (c *ClientWithCircuitBreaker) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	streamingClient := NewStreamingClient(c.client)
	return streamingClient.ChatCompletionStream(ctx, req)
}

// Generate отправляет generate запрос с circuit breaker
func (c *ClientWithCircuitBreaker) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if !c.enabled {
		return c.client.Generate(ctx, req)
	}

	breaker := c.circuitManager.GetBreaker("ollama-generate")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.Generate(ctx, req)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*GenerateResponse), nil
}

// ShowModel получает информацию о модели с circuit breaker
func (c *ClientWithCircuitBreaker) ShowModel(ctx context.Context, modelName string) (*ShowResponse, error) {
	if !c.enabled {
		return c.client.ShowModel(ctx, modelName)
	}

	breaker := c.circuitManager.GetBreaker("ollama-show")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.ShowModel(ctx, modelName)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*ShowResponse), nil
}

// PullModel загружает модель с circuit breaker
func (c *ClientWithCircuitBreaker) PullModel(ctx context.Context, modelName string) (*PullResponse, error) {
	if !c.enabled {
		return c.client.PullModel(ctx, modelName)
	}

	breaker := c.circuitManager.GetBreaker("ollama-pull")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.PullModel(ctx, modelName)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*PullResponse), nil
}

// Embed создает embeddings (batch) с circuit breaker
func (c *ClientWithCircuitBreaker) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	if !c.enabled {
		return c.client.Embed(ctx, req)
	}

	breaker := c.circuitManager.GetBreaker("ollama-embed")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.Embed(ctx, req)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*EmbedResponse), nil
}

// Embeddings создает single embedding (legacy) с circuit breaker
func (c *ClientWithCircuitBreaker) Embeddings(ctx context.Context, req *EmbeddingsRequest) (*EmbeddingsResponse, error) {
	if !c.enabled {
		return c.client.Embeddings(ctx, req)
	}

	breaker := c.circuitManager.GetBreaker("ollama-embeddings")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.Embeddings(ctx, req)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*EmbeddingsResponse), nil
}

// GetEmbeddings - legacy helper method (deprecated, use Embed instead)
func (c *ClientWithCircuitBreaker) GetEmbeddings(ctx context.Context, model, text string) (*EmbeddingsResponse, error) {
	if !c.enabled {
		return c.client.GetEmbeddings(ctx, model, text)
	}

	breaker := c.circuitManager.GetBreaker("ollama-embeddings")

	operation := func(ctx context.Context) (interface{}, error) {
		return c.client.GetEmbeddings(ctx, model, text)
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return nil, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return nil, err
	}

	return result.(*EmbeddingsResponse), nil
}

// IsModelAvailable проверяет доступность модели с circuit breaker
func (c *ClientWithCircuitBreaker) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	if !c.enabled {
		return c.client.IsModelAvailable(ctx, modelName)
	}

	breaker := c.circuitManager.GetBreaker("ollama-availability")

	operation := func(ctx context.Context) (interface{}, error) {
		available, err := c.client.IsModelAvailable(ctx, modelName)
		return available, err
	}

	result, err := breaker.Execute(ctx, operation)
	if err != nil {
		if circuitErr, ok := err.(*circuit.CircuitBreakerError); ok {
			return false, errors.NewServiceUnavailableError("ollama", circuitErr)
		}
		return false, err
	}

	return result.(bool), nil
}

// Close закрывает клиент и очищает ресурсы
func (c *ClientWithCircuitBreaker) Close() {
	c.client.Close()
	c.logger.Debug("Circuit breaker client closed")
}

// GetCircuitBreakerStats возвращает статистику всех circuit breaker'ов
func (c *ClientWithCircuitBreaker) GetCircuitBreakerStats() map[string]circuit.Stats {
	return c.circuitManager.GetAllStats()
}

// ResetCircuitBreakers сбрасывает все circuit breaker'ы
func (c *ClientWithCircuitBreaker) ResetCircuitBreakers() {
	c.circuitManager.ResetAll()
	c.logger.Info("All circuit breakers reset")
}

