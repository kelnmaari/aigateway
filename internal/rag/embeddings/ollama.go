// Package embeddings implements Ollama embeddings client.
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// OllamaEmbedder implements Embedder using Ollama API
type OllamaEmbedder struct {
	config      EmbedderConfig
	httpClient  *http.Client
	logger      *logrus.Logger
	rateLimiter *rate.Limiter
}

// NewOllamaEmbedder creates new Ollama embedder
func NewOllamaEmbedder(config EmbedderConfig, logger *logrus.Logger) *OllamaEmbedder {
	if logger == nil {
		logger = logrus.New()
	}

	// Setup HTTP client с timeout
	httpClient := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	// Setup rate limiter
	var limiter *rate.Limiter
	if config.RateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(config.RateLimit), 1)
	}

	return &OllamaEmbedder{
		config:      config,
		httpClient:  httpClient,
		logger:      logger,
		rateLimiter: limiter,
	}
}

// Name возвращает имя embedder'а
func (e *OllamaEmbedder) Name() string {
	return "ollama"
}

// GetDefaultModel возвращает модель по умолчанию
func (e *OllamaEmbedder) GetDefaultModel() string {
	return e.config.Model
}

// GetDimensions возвращает размерность векторов для модели
func (e *OllamaEmbedder) GetDimensions(model string) (int, error) {
	// Known models dimensions
	dimensions := map[string]int{
		"mxbai-embed-large":     1024,
		"nomic-embed-text":      768,
		"all-minilm":            384,
		"snowflake-arctic-embed": 1024,
	}

	if dim, ok := dimensions[model]; ok {
		return dim, nil
	}

	// Default to config dimensions
	return e.config.Dimensions, nil
}

// Embed генерирует embedding для одного текста
func (e *OllamaEmbedder) Embed(ctx context.Context, req EmbeddingRequest) (*Embedding, error) {
	// Rate limiting
	if e.rateLimiter != nil {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit error: %w", err)
		}
	}

	model := req.Model
	if model == "" {
		model = e.config.Model
	}

	// Ollama API request
	ollamaReq := map[string]interface{}{
		"model":  model,
		"prompt": req.Text,
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/embeddings", e.config.BaseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	startTime := time.Now()

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(startTime)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp struct {
		Embedding []float64 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"model":       model,
		"text_length": len(req.Text),
		"dimensions":  len(ollamaResp.Embedding),
		"duration_ms": duration.Milliseconds(),
	}).Debug("Embedding generated")

	embedding := &Embedding{
		Vector:     ollamaResp.Embedding,
		Dimensions: len(ollamaResp.Embedding),
		Model:      model,
		Metadata:   req.Metadata,
	}

	return embedding, nil
}

// EmbedBatch генерирует embeddings для batch текстов
func (e *OllamaEmbedder) EmbedBatch(ctx context.Context, req BatchEmbeddingRequest) (*BatchEmbeddingResponse, error) {
	if len(req.Texts) == 0 {
		return &BatchEmbeddingResponse{
			Embeddings: []Embedding{},
			Model:      req.Model,
		}, nil
	}

	model := req.Model
	if model == "" {
		model = e.config.Model
	}

	// Разбиваем на batches по MaxBatchSize
	batchSize := e.config.MaxBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	var allEmbeddings []Embedding
	totalTokens := 0

	for i := 0; i < len(req.Texts); i += batchSize {
		end := i + batchSize
		if end > len(req.Texts) {
			end = len(req.Texts)
		}

		batch := req.Texts[i:end]

		e.logger.WithFields(logrus.Fields{
			"batch_start": i,
			"batch_size":  len(batch),
			"total":       len(req.Texts),
		}).Debug("Processing batch")

		// Генерируем embeddings для каждого текста в batch
		for idx, text := range batch {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			embedding, err := e.Embed(ctx, EmbeddingRequest{
				Text:     text,
				Model:    model,
				Metadata: req.Metadata,
			})

			if err != nil {
				e.logger.WithError(err).WithFields(logrus.Fields{
					"batch_index": idx,
					"text_length": len(text),
				}).Error("Failed to generate embedding")
				
				// Возвращаем ошибку или пропускаем?
				// Сейчас возвращаем ошибку для строгого контроля
				return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i+idx, err)
			}

			allEmbeddings = append(allEmbeddings, *embedding)
			
			// Estimate tokens (1 token ≈ 4 chars)
			totalTokens += len(text) / 4
		}
	}

	e.logger.WithFields(logrus.Fields{
		"total_texts":      len(req.Texts),
		"total_embeddings": len(allEmbeddings),
		"total_tokens":     totalTokens,
		"model":            model,
	}).Info("Batch embeddings completed")

	return &BatchEmbeddingResponse{
		Embeddings:  allEmbeddings,
		Model:       model,
		TotalTokens: totalTokens,
	}, nil
}


