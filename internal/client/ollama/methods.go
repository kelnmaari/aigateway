// Package ollama provides methods for Ollama API operations
package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/metrics"
)

// GetModels получает список доступных моделей из Ollama
func (c *Client) GetModels(ctx context.Context) (*ModelsResponse, error) {
	start := time.Now()
	operation := "get_models"
	
	// Record metrics at the end
	defer func() {
		if metrics.DefaultMetrics != nil {
			duration := time.Since(start)
			status := "success"
			metrics.DefaultMetrics.RecordOllamaRequest("", operation, status, duration)
		}
	}()

	req, err := c.makeRequest(ctx, "GET", "/api/tags", nil)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError("", operation, "request_creation_failed")
		}
		return nil, fmt.Errorf("failed to create models request: %w", err)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError("", operation, "request_failed")
		}
		return nil, fmt.Errorf("failed to get models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError("", operation, "bad_status_code")
		}
		return nil, c.parseErrorResponse(resp)
	}

	var modelsResponse ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResponse); err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError("", operation, "decode_failed")
		}
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"models_count": len(modelsResponse.Models),
	}).Debug("Successfully retrieved models from Ollama")

	return &modelsResponse, nil
}

// ChatCompletion отправляет chat completion запрос в Ollama
func (c *Client) ChatCompletion(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	start := time.Now()
	operation := "chat_completion"
	model := req.Model
	
	// Record metrics at the end
	defer func() {
		if metrics.DefaultMetrics != nil {
			duration := time.Since(start)
			status := "success"
			metrics.DefaultMetrics.RecordOllamaRequest(model, operation, status, duration)
		}
	}()

	httpReq, err := c.makeRequest(ctx, "POST", "/api/chat", req)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_creation_failed")
		}
		return nil, fmt.Errorf("failed to create chat request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"messages_count": len(req.Messages),
		"stream":         req.Stream,
	}).Debug("Sending chat completion request to Ollama")

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_failed")
		}
		return nil, fmt.Errorf("failed to send chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "bad_status_code")
		}
		return nil, c.parseErrorResponse(resp)
	}

	var chatResponse ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "decode_failed")
		}
		return nil, fmt.Errorf("failed to decode chat response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":             chatResponse.Model,
		"response_length":   len(chatResponse.Message.Content),
		"total_duration_ms": chatResponse.TotalDuration / 1000000, // Convert to ms
		"eval_count":        chatResponse.EvalCount,
		"eval_duration_ms":  chatResponse.EvalDuration / 1000000,
	}).Debug("Received chat completion response from Ollama")

	return &chatResponse, nil
}

// Generate отправляет generate запрос в Ollama (для простой генерации текста)
func (c *Client) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	httpReq, err := c.makeRequest(ctx, "POST", "/api/generate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create generate request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":         req.Model,
		"prompt_length": len(req.Prompt),
		"stream":        req.Stream,
	}).Debug("Sending generate request to Ollama")

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send generate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var generateResponse GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&generateResponse); err != nil {
		return nil, fmt.Errorf("failed to decode generate response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":             generateResponse.Model,
		"response_length":   len(generateResponse.Response),
		"total_duration_ms": generateResponse.TotalDuration / 1000000,
		"eval_count":        generateResponse.EvalCount,
	}).Debug("Received generate response from Ollama")

	return &generateResponse, nil
}

// ShowModel получает детальную информацию о модели
func (c *Client) ShowModel(ctx context.Context, modelName string) (*ShowResponse, error) {
	req := &ShowRequest{Name: modelName}

	httpReq, err := c.makeRequest(ctx, "POST", "/api/show", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create show request: %w", err)
	}

	c.logger.WithField("model", modelName).Debug("Requesting model information from Ollama")

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var showResponse ShowResponse
	if err := json.NewDecoder(resp.Body).Decode(&showResponse); err != nil {
		return nil, fmt.Errorf("failed to decode show response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":      modelName,
		"family":     showResponse.Details.Family,
		"parameters": showResponse.Details.ParameterSize,
	}).Debug("Received model information from Ollama")

	return &showResponse, nil
}

// PullModel загружает модель из registry
func (c *Client) PullModel(ctx context.Context, modelName string) (*PullResponse, error) {
	req := &PullRequest{
		Name:   modelName,
		Stream: false, // Для MVP не используем streaming
	}

	httpReq, err := c.makeRequest(ctx, "POST", "/api/pull", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	c.logger.WithField("model", modelName).Info("Pulling model from Ollama registry")

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to pull model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var pullResponse PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pullResponse); err != nil {
		return nil, fmt.Errorf("failed to decode pull response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":  modelName,
		"status": pullResponse.Status,
	}).Info("Model pull completed")

	return &pullResponse, nil
}

// GetEmbeddings получает embeddings для текста
func (c *Client) GetEmbeddings(ctx context.Context, model, text string) (*EmbeddingsResponse, error) {
	req := &EmbeddingsRequest{
		Model:  model,
		Prompt: text,
	}

	httpReq, err := c.makeRequest(ctx, "POST", "/api/embeddings", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":       model,
		"text_length": len(text),
	}).Debug("Requesting embeddings from Ollama")

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get embeddings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var embeddingsResponse EmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode embeddings response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":          model,
		"embedding_dims": len(embeddingsResponse.Embedding),
	}).Debug("Received embeddings from Ollama")

	return &embeddingsResponse, nil
}

// IsModelAvailable проверяет доступность модели
func (c *Client) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	models, err := c.GetModels(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get models list: %w", err)
	}

	for _, model := range models.Models {
		if model.Name == modelName {
			return true, nil
		}
	}

	return false, nil
}

