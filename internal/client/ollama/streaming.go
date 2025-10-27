// Package ollama provides streaming support for Ollama API
package ollama

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// StreamingClient обрабатывает streaming запросы к Ollama
type StreamingClient struct {
	*Client
}

// NewStreamingClient создает клиент с поддержкой streaming
func NewStreamingClient(client *Client) *StreamingClient {
	return &StreamingClient{Client: client}
}

// ChatCompletionStream отправляет streaming chat completion запрос
func (c *StreamingClient) ChatCompletionStream(ctx context.Context, req *ChatRequest) (<-chan *ChatResponse, <-chan error) {
	// Каналы для streaming данных
	responseChan := make(chan *ChatResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		if err := c.processStreamingRequest(ctx, req, responseChan, errorChan); err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
			}
		}
	}()

	return responseChan, errorChan
}

// processStreamingRequest обрабатывает streaming запрос
func (c *StreamingClient) processStreamingRequest(ctx context.Context, req *ChatRequest, responseChan chan<- *ChatResponse, errorChan chan<- error) error {
	// Убеждаемся что включен streaming
	req.Stream = true

	c.logger.WithFields(logrus.Fields{
		"model":    req.Model,
		"messages": len(req.Messages),
	}).Debug("Starting Ollama streaming request")

	// Создаем HTTP запрос БЕЗ retry для streaming (retry ломает body)
	httpReq, err := c.makeRequest(ctx, "POST", "/api/chat", req)
	if err != nil {
		return fmt.Errorf("failed to create streaming request: %w", err)
	}

	// ПРЯМОЙ запрос без retry для streaming
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send streaming request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp)
	}

	c.logger.Debug("Successfully connected to Ollama streaming endpoint")

	// Обрабатываем streaming ответ
	return c.processStreamingResponse(ctx, resp.Body, responseChan)
}

// processStreamingResponse обрабатывает streaming ответ от Ollama
func (c *StreamingClient) processStreamingResponse(ctx context.Context, body io.ReadCloser, responseChan chan<- *ChatResponse) error {
	scanner := bufio.NewScanner(body)

	c.logger.Debug("Started processing Ollama streaming response")

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			c.logger.Debug("Streaming cancelled by context")
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue // Пропускаем пустые строки
		}

		// Парсим JSON chunk
		var chunk ChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			c.logger.WithError(err).WithField("line", line).Warn("Failed to parse streaming chunk")
			continue // Пропускаем невалидные chunks
		}

		// 🔍 КРИТИЧНОЕ ЛОГИРОВАНИЕ: проверяем ОБА места для tool_calls
		msgToolCallsCount := len(chunk.Message.ToolCalls)
		respToolCallsCount := len(chunk.ToolCalls)

		c.logger.WithFields(logrus.Fields{
			"model":           chunk.Model,
			"done":            chunk.Done,
			"content":         len(chunk.Message.Content),
			"msg_tool_calls":  msgToolCallsCount,  // В Message
			"resp_tool_calls": respToolCallsCount, // В Response (как в официальном API)
			"thinking":        len(chunk.Message.Thinking),
		}).Debug("Received Ollama streaming chunk")

		// 🎯 DEBUG: Логируем tool_calls если есть В ЛЮБОМ МЕСТЕ
		if msgToolCallsCount > 0 {
			c.logger.WithFields(logrus.Fields{
				"location":   "message.tool_calls",
				"tool_calls": chunk.Message.ToolCalls,
			}).Info("🎯 TOOL CALLS DETECTED in Message!")
		}

		if respToolCallsCount > 0 {
			c.logger.WithFields(logrus.Fields{
				"location":   "response.tool_calls",
				"tool_calls": chunk.ToolCalls,
			}).Info("🎯 TOOL CALLS DETECTED in Response!")
		}

		// Отправляем chunk в канал
		select {
		case responseChan <- &chunk:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Если ответ завершен, заканчиваем
		if chunk.Done {
			c.logger.Debug("Ollama streaming completed")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading streaming response: %w", err)
	}

	return nil
}

// GenerateStream отправляет streaming generate запрос
func (c *StreamingClient) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *GenerateResponse, <-chan error) {
	responseChan := make(chan *GenerateResponse, 10)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		if err := c.processGenerateStreamingRequest(ctx, req, responseChan); err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
			}
		}
	}()

	return responseChan, errorChan
}

// processGenerateStreamingRequest обрабатывает streaming generate запрос
func (c *StreamingClient) processGenerateStreamingRequest(ctx context.Context, req *GenerateRequest, responseChan chan<- *GenerateResponse) error {
	req.Stream = true

	httpReq, err := c.makeRequest(ctx, "POST", "/api/generate", req)
	if err != nil {
		return fmt.Errorf("failed to create generate streaming request: %w", err)
	}

	resp, err := c.doWithRetry(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send generate streaming request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp)
	}

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var chunk GenerateResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			c.logger.WithError(err).Warn("Failed to parse generate streaming chunk")
			continue
		}

		select {
		case responseChan <- &chunk:
		case <-ctx.Done():
			return ctx.Err()
		}

		if chunk.Done {
			break
		}
	}

	return scanner.Err()
}

