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

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
)

// TestCompletionsHandler_HandleCompletions_Success тестирует успешный completion
func TestCompletionsHandler_HandleCompletions_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewCompletionsHandler(cfg, logger, mockClient)

	reqBody := models.CompletionRequest{
		Model:  "text-davinci-003",
		Prompt: "Hello, world!",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleCompletions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.CompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
	assert.NotEmpty(t, resp.Choices)
}

// TestCompletionsHandler_HandleCompletions_InvalidJSON тестирует невалидный JSON
func TestCompletionsHandler_HandleCompletions_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewCompletionsHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleCompletions(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

// mockCompletionsClientWithError для тестирования ошибок Ollama в completions
type mockCompletionsClientWithError struct {
	MockOllamaClient
	err error
}

func (m *mockCompletionsClientWithError) ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error) {
	return nil, m.err
}

// TestCompletionsHandler_HandleCompletions_OllamaError тестирует ошибку от Ollama
func TestCompletionsHandler_HandleCompletions_OllamaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &mockCompletionsClientWithError{err: errors.New("model error")}
	handler := NewCompletionsHandler(cfg, logger, mockClient)

	reqBody := models.CompletionRequest{
		Model:  "nonexistent",
		Prompt: "Hello",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleCompletions(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestCompletionsHandler_HandleCompletions_ArrayPrompt тестирует array prompt
func TestCompletionsHandler_HandleCompletions_ArrayPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewCompletionsHandler(cfg, logger, mockClient)

	reqBody := models.CompletionRequest{
		Model:  "text-davinci-003",
		Prompt: []string{"Hello", "World"},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleCompletions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.CompletionResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.ID)
}
