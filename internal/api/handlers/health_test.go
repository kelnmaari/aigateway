package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/config"
)

// Note: Mock client defined in test_helpers.go

// TestHealthHandler_Health тестирует базовый health check endpoint
func TestHealthHandler_Health(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "health_check_always_returns_ok",
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			cfg := &config.Config{}
			logger := logrus.New()
			logger.SetOutput(io.Discard) // Suppress logs in tests

			mockClient := &MockOllamaClient{}
			handler := NewHealthHandler(cfg, logger, mockClient)

			// Create request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

			// Execute
			handler.Health(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

// mockOllamaClientWithError для тестирования ошибок
type mockOllamaClientWithError struct {
	MockOllamaClient
	healthError error
}

func (m *mockOllamaClientWithError) Health(ctx context.Context) error {
	return m.healthError
}

// TestHealthHandler_Ready тестирует ready check endpoint с проверкой Ollama
func TestHealthHandler_Ready(t *testing.T) {
	t.Run("ollama_available_returns_ok", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)
		cfg := &config.Config{}
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		mockClient := &MockOllamaClient{} // Health() возвращает nil
		handler := NewHealthHandler(cfg, logger, mockClient)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

		// Execute
		handler.Ready(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ready")
	})

	t.Run("ollama_unavailable_returns_503", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)
		cfg := &config.Config{}
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		mockClient := &mockOllamaClientWithError{
			healthError: errors.New("connection refused"),
		}
		handler := NewHealthHandler(cfg, logger, mockClient)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)

		// Execute
		handler.Ready(c)

		// Assert
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "not_ready") // JSON response с checks
	})
}

// TestHealthHandler_Live тестирует liveness check endpoint
func TestHealthHandler_Live(t *testing.T) {
	t.Run("liveness_always_returns_ok", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)
		cfg := &config.Config{}
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		mockClient := &MockOllamaClient{}
		handler := NewHealthHandler(cfg, logger, mockClient)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)

		// Execute
		handler.Live(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "alive")
	})

	t.Run("liveness_ok_even_if_ollama_down", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)
		cfg := &config.Config{}
		logger := logrus.New()
		logger.SetOutput(io.Discard)

		// Liveness не зависит от Ollama, но проверяем что даже с ошибкой всё OK
		mockClient := &mockOllamaClientWithError{
			healthError: errors.New("ollama down"),
		}
		handler := NewHealthHandler(cfg, logger, mockClient)

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)

		// Execute
		handler.Live(c)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "alive")
	})
}

// mockOllamaClientToggleable для тестирования переходов состояния
type mockOllamaClientToggleable struct {
	MockOllamaClient
	healthy *bool
}

func (m *mockOllamaClientToggleable) Health(ctx context.Context) error {
	if *m.healthy {
		return nil
	}
	return errors.New("connection refused")
}

// TestHealthHandler_Integration тестирует полную интеграцию health endpoints
func TestHealthHandler_Integration(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Ollama: config.OllamaConfig{
			URL: "http://localhost:11434",
		},
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	// Test scenario: Ollama transitions from down to up
	ollamaHealthy := false
	mockClient := &mockOllamaClientToggleable{
		healthy: &ollamaHealthy,
	}
	handler := NewHealthHandler(cfg, logger, mockClient)

	// 1. Health endpoint always works
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)
	handler.Health(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// 2. Ready endpoint fails when Ollama is down
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)
	handler.Ready(c)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	// 3. Liveness always works regardless of Ollama
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.Live(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// 4. Ollama comes back online
	ollamaHealthy = true

	// 5. Ready endpoint now succeeds
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/ready", nil)
	handler.Ready(c)
	assert.Equal(t, http.StatusOK, w.Code)
}

// mockOllamaClientContextAware для тестирования отмены контекста
type mockOllamaClientContextAware struct {
	MockOllamaClient
}

func (m *mockOllamaClientContextAware) Health(ctx context.Context) error {
	// Simulate context cancellation check
	return ctx.Err()
}

// TestHealthHandler_ContextCancellation тестирует обработку отмены контекста
func TestHealthHandler_ContextCancellation(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockClient := &mockOllamaClientContextAware{}
	handler := NewHealthHandler(cfg, logger, mockClient)

	// Create request with cancelled context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	req = req.WithContext(ctx)
	c.Request = req

	// Execute
	handler.Ready(c)

	// Assert - should still handle gracefully
	require.NotEqual(t, 0, w.Code, "Handler should respond even with cancelled context")
}

