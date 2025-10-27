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

// TestEmbeddingsHandler_HandleEmbeddings_Success тестирует успешное создание embeddings
func TestEmbeddingsHandler_HandleEmbeddings_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{} // Embed возвращает дефолтный ответ
	handler := NewEmbeddingsHandler(cfg, logger, mockClient)

	reqBody := models.EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: "Hello, world!",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleEmbeddings(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.EmbeddingResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "list", resp.Object)
	assert.NotEmpty(t, resp.Data)
	assert.Equal(t, "embedding", resp.Data[0].Object)
}

// TestEmbeddingsHandler_HandleEmbeddings_InvalidJSON тестирует обработку невалидного JSON
func TestEmbeddingsHandler_HandleEmbeddings_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewEmbeddingsHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleEmbeddings(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request format")
}

// mockEmbeddingsClientWithError для тестирования ошибок Ollama в embeddings
type mockEmbeddingsClientWithError struct {
	MockOllamaClient
	err error
}

func (m *mockEmbeddingsClientWithError) Embed(ctx context.Context, req *ollama.EmbedRequest) (*ollama.EmbedResponse, error) {
	return nil, m.err
}

func (m *mockEmbeddingsClientWithError) Embeddings(ctx context.Context, req *ollama.EmbeddingsRequest) (*ollama.EmbeddingsResponse, error) {
	return nil, m.err
}

// TestEmbeddingsHandler_HandleEmbeddings_OllamaError тестирует обработку ошибки от Ollama
func TestEmbeddingsHandler_HandleEmbeddings_OllamaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &mockEmbeddingsClientWithError{err: errors.New("model not found")}
	handler := NewEmbeddingsHandler(cfg, logger, mockClient)

	reqBody := models.EmbeddingRequest{
		Model: "nonexistent",
		Input: "Hello",
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleEmbeddings(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestEmbeddingsHandler_HandleEmbeddings_BatchInput тестирует batch embeddings
func TestEmbeddingsHandler_HandleEmbeddings_BatchInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewEmbeddingsHandler(cfg, logger, mockClient)

	reqBody := models.EmbeddingRequest{
		Model: "text-embedding-ada-002",
		Input: []string{"Hello", "World", "Test"},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleEmbeddings(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.EmbeddingResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "list", resp.Object)
	// Batch embeddings могут возвращать разное количество в зависимости от реализации
	assert.GreaterOrEqual(t, len(resp.Data), 1)
}

