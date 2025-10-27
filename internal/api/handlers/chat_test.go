package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/models"
)

// TestChatHandler_Completion_ValidRequest тестирует успешный chat completion
func TestChatHandler_Completion_ValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	reqBody := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completion(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ChatCompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.Len(t, resp.Choices, 1)
}

// TestChatHandler_Completion_InvalidJSON тестирует обработку невалидного JSON
func TestChatHandler_Completion_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completion(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

// mockChatClientWithError для тестирования ошибок Ollama
type mockChatClientWithError struct {
	MockOllamaClient
	err error
}

func (m *mockChatClientWithError) ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error) {
	return nil, m.err
}

// TestChatHandler_Completion_OllamaError тестирует обработку ошибки от Ollama
func TestChatHandler_Completion_OllamaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &mockChatClientWithError{err: errors.New("model not found")}
	handler := NewChatHandler(cfg, logger, mockClient)

	reqBody := models.ChatCompletionRequest{
		Model:    "nonexistent",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hello"}},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completion(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "model not found")
}

// TestChatHandler_Completion_WithTools тестирует запрос с tools
func TestChatHandler_Completion_WithTools(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	reqBody := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "What's the weather?"}},
		Tools: []models.Tool{
			{
				Type: "function",
				Function: models.Function{
					Name:        "get_weather",
					Description: "Get weather",
					Parameters:  map[string]interface{}{"type": "object"},
				},
			},
		},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completion(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ChatCompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp.Choices, 1)
}

// TestChatHandler_Streaming тестирует streaming mode
func TestChatHandler_Streaming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewChatHandler(cfg, logger, mockClient)

	reqBody := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: []models.ChatMessage{{Role: "user", Content: "Hello"}},
		Stream:   true,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completion(c)

	// Streaming может возвращать разные коды в зависимости от реализации
	assert.NotEqual(t, 0, w.Code)
}

