package handlers

import (
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

// TestModelsHandler_List_Success тестирует успешное получение списка моделей
func TestModelsHandler_List_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{} // GetModels возвращает дефолтный список
	handler := NewModelsHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "list", resp.Object)
	assert.GreaterOrEqual(t, len(resp.Data), 0)
}

// mockModelsClientWithError для тестирования ошибок Ollama в models
type mockModelsClientWithError struct {
	MockOllamaClient
	err error
}

func (m *mockModelsClientWithError) GetModels(ctx context.Context) (*ollama.ModelsResponse, error) {
	return nil, m.err
}

// TestModelsHandler_List_OllamaError тестирует обработку ошибки от Ollama
func TestModelsHandler_List_OllamaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &mockModelsClientWithError{err: errors.New("ollama unavailable")}
	handler := NewModelsHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	handler.List(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestModelsHandler_List_ResponseFormat тестирует формат ответа
func TestModelsHandler_List_ResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &MockOllamaClient{}
	handler := NewModelsHandler(cfg, logger, mockClient)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)

	handler.List(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ModelsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	// Проверяем структуру ответа
	assert.Equal(t, "list", resp.Object)
	assert.NotNil(t, resp.Data)

	// Если есть модели, проверяем их структуру
	if len(resp.Data) > 0 {
		model := resp.Data[0]
		assert.NotEmpty(t, model.ID)
		assert.Equal(t, "model", model.Object)
		assert.NotZero(t, model.Created)
		assert.NotEmpty(t, model.OwnedBy)
	}
}
