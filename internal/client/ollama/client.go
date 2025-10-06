// Package ollama provides HTTP client for Ollama API
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/metrics"
)

// Client представляет HTTP клиент для Ollama API
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *logrus.Logger
	timeout    time.Duration
	retries    int
	retryDelay time.Duration
}

// NewClient создает новый экземпляр Ollama клиента
func NewClient(cfg *config.Config, logger *logrus.Logger) (*Client, error) {
	baseURL, err := url.Parse(cfg.Ollama.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}

	// Создаем HTTP клиент с увеличенными timeout'ами для больших моделей
	timeout := cfg.Ollama.Timeout
	if timeout < 60*time.Second {
		timeout = 60 * time.Second // Минимум 1 минута для больших моделей
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:          cfg.Ollama.ConnectionPoolSize,
			MaxIdleConnsPerHost:   cfg.Ollama.ConnectionPoolSize,
			IdleConnTimeout:       90 * time.Second,
			DisableKeepAlives:     !cfg.Ollama.KeepAlive,
			ResponseHeaderTimeout: 120 * time.Second, // 2 минуты ждем headers от Ollama
		},
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
		timeout:    cfg.Ollama.Timeout,
		retries:    cfg.Ollama.RetryAttempts,
		retryDelay: cfg.Ollama.RetryDelay,
	}, nil
}

// Health проверяет доступность Ollama сервера
func (c *Client) Health(ctx context.Context) error {
	url := c.baseURL.ResolveReference(&url.URL{Path: "/api/tags"})

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.doWithRetry(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	c.logger.Debug("Ollama health check passed")
	return nil
}

// doWithRetry выполняет HTTP запрос с retry логикой
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			c.logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"delay":   c.retryDelay,
			}).Debug("Retrying Ollama request")

			time.Sleep(c.retryDelay)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.logger.WithError(err).WithField("attempt", attempt).Warn("Ollama request failed")
			continue
		}

		// Если статус код указывает на временную ошибку, пробуем еще раз
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			c.logger.WithField("status_code", resp.StatusCode).WithField("attempt", attempt).Warn("Ollama server error")
			continue
		}

		// Успешный ответ или клиентская ошибка (не ретраим)
		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded, last error: %w", lastErr)
}

// parseErrorResponse извлекает ошибку из ответа Ollama
func (c *Client) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read error response: %w", err)
	}

	var errorResponse struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Errorf("Ollama error: %s", errorResponse.Error)
}

// makeRequest создает HTTP запрос с общими заголовками
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL.ResolveReference(&url.URL{Path: path})

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)

		c.logger.WithFields(logrus.Fields{
			"method": method,
			"url":    url.String(),
			"body":   string(jsonBody),
		}).Debug("Creating Ollama request")
	} else {
		c.logger.WithFields(logrus.Fields{
			"method": method,
			"url":    url.String(),
		}).Debug("Creating Ollama request")
	}

	req, err := http.NewRequestWithContext(ctx, method, url.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Устанавливаем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return req, nil
}

// Embed создает embeddings для одного или нескольких входов (batch)
// Использует новый /api/embed endpoint с поддержкой batch processing
func (c *Client) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error) {
	start := time.Now()
	operation := "embed"
	model := req.Model

	// Record metrics at the end
	defer func() {
		if metrics.DefaultMetrics != nil {
			duration := time.Since(start)
			status := "success"
			metrics.DefaultMetrics.RecordOllamaRequest(model, operation, status, duration)
		}
	}()

	httpReq, err := c.makeRequest(ctx, "POST", "/api/embed", req)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_creation_failed")
		}
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_failed")
		}
		return nil, fmt.Errorf("embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "bad_status_code")
		}
		return nil, c.parseErrorResponse(resp)
	}

	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "decode_failed")
		}
		return nil, fmt.Errorf("failed to decode embed response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":       req.Model,
		"embeddings":  len(embedResp.Embeddings),
		"duration_ms": embedResp.TotalDuration / 1_000_000,
	}).Debug("Embeddings generated successfully")

	return &embedResp, nil
}

// Embeddings создает single embedding (legacy endpoint)
// Использует старый /api/embeddings endpoint для обратной совместимости
func (c *Client) Embeddings(ctx context.Context, req *EmbeddingsRequest) (*EmbeddingsResponse, error) {
	start := time.Now()
	operation := "embeddings_legacy"
	model := req.Model

	// Record metrics at the end
	defer func() {
		if metrics.DefaultMetrics != nil {
			duration := time.Since(start)
			status := "success"
			metrics.DefaultMetrics.RecordOllamaRequest(model, operation, status, duration)
		}
	}()

	httpReq, err := c.makeRequest(ctx, "POST", "/api/embeddings", req)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_creation_failed")
		}
		return nil, fmt.Errorf("failed to create embeddings request: %w", err)
	}

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "request_failed")
		}
		return nil, fmt.Errorf("embeddings request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "bad_status_code")
		}
		return nil, c.parseErrorResponse(resp)
	}

	var embResp EmbeddingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		if metrics.DefaultMetrics != nil {
			metrics.DefaultMetrics.RecordOllamaError(model, operation, "decode_failed")
		}
		return nil, fmt.Errorf("failed to decode embeddings response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"model":          req.Model,
		"embedding_size": len(embResp.Embedding),
	}).Debug("Embedding generated successfully")

	return &embResp, nil
}

// Close закрывает соединения клиента
func (c *Client) Close() {
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	c.logger.Debug("Ollama client connections closed")
}
