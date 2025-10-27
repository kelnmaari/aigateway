// Package test provides testing helpers and utilities for Ollama-OpenAI Proxy
package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
)

// MockOllamaClient предоставляет mock implementation для Ollama клиента
type MockOllamaClient struct {
	mock.Mock
}

func (m *MockOllamaClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockOllamaClient) GetModels(ctx context.Context) (*ollama.ModelsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ollama.ModelsResponse), args.Error(1)
}

func (m *MockOllamaClient) ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ollama.ChatResponse), args.Error(1)
}

func (m *MockOllamaClient) Generate(ctx context.Context, req *ollama.GenerateRequest) (*ollama.GenerateResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ollama.GenerateResponse), args.Error(1)
}

func (m *MockOllamaClient) ShowModel(ctx context.Context, modelName string) (*ollama.ShowResponse, error) {
	args := m.Called(ctx, modelName)
	return args.Get(0).(*ollama.ShowResponse), args.Error(1)
}

func (m *MockOllamaClient) PullModel(ctx context.Context, modelName string) (*ollama.PullResponse, error) {
	args := m.Called(ctx, modelName)
	return args.Get(0).(*ollama.PullResponse), args.Error(1)
}

func (m *MockOllamaClient) GetEmbeddings(ctx context.Context, model, text string) (*ollama.EmbeddingsResponse, error) {
	args := m.Called(ctx, model, text)
	return args.Get(0).(*ollama.EmbeddingsResponse), args.Error(1)
}

func (m *MockOllamaClient) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	args := m.Called(ctx, modelName)
	return args.Bool(0), args.Error(1)
}

func (m *MockOllamaClient) Close() {
	m.Called()
}

// TestConfig создает тестовую конфигурацию
func TestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Ollama: config.OllamaConfig{
			URL:                "http://localhost:11434",
			Timeout:            30 * time.Second,
			RetryAttempts:      3,
			RetryDelay:         1 * time.Second,
			ConnectionPoolSize: 10,
			KeepAlive:          true,
		},
		Auth: config.AuthConfig{
			Enabled:     false,
			StorageType: "json",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "text",
			Output: "stdout",
		},
		Models: config.ModelsConfig{
			Cache: struct {
				Enabled         bool          `mapstructure:"enabled"`
				TTL             time.Duration `mapstructure:"ttl"`
				RefreshInterval time.Duration `mapstructure:"refresh_interval"`
			}{
				Enabled:         true,
				TTL:             5 * time.Minute,
				RefreshInterval: 1 * time.Minute,
			},
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
		},
	}
}

// TestLogger создает тестовый logger
func TestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

// TestRouter создает тестовый Gin router
func TestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// TestRecorder создает HTTP test recorder
func TestRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

// AssertJSONResponse проверяет JSON ответ
func AssertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(t, expectedStatus, recorder.Code)
	assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
}

// AssertErrorResponse проверяет error ответ
func AssertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, expectedErrorCode string) {
	AssertJSONResponse(t, recorder, expectedStatus)
	// TODO: Добавить парсинг JSON и проверку error.code
}

// CreateTestModels создает тестовые модели для mock'ов
func CreateTestModels() *ollama.ModelsResponse {
	return &ollama.ModelsResponse{
		Models: []ollama.Model{
			{
				Name:       "llama2",
				ModifiedAt: time.Now(),
				Size:       4000000000, // 4GB
				Digest:     "sha256:test123",
				Details: ollama.Details{
					Format:            "gguf",
					Family:            "llama",
					ParameterSize:     "7B",
					QuantizationLevel: "Q4_0",
				},
			},
			{
				Name:       "mistral",
				ModifiedAt: time.Now(),
				Size:       3500000000, // 3.5GB
				Digest:     "sha256:test456",
				Details: ollama.Details{
					Format:            "gguf",
					Family:            "mistral",
					ParameterSize:     "7B",
					QuantizationLevel: "Q4_0",
				},
			},
		},
	}
}

// CreateTestChatResponse создает тестовый chat response
func CreateTestChatResponse(model, content string) *ollama.ChatResponse {
	return &ollama.ChatResponse{
		Model:     model,
		CreatedAt: time.Now(),
		Message: ollama.ChatMessage{
			Role:    "assistant",
			Content: content,
		},
		Done:               true,
		TotalDuration:      1000000000, // 1 second in nanoseconds
		LoadDuration:       100000000,  // 100ms
		PromptEvalCount:    10,
		PromptEvalDuration: 200000000, // 200ms
		EvalCount:          20,
		EvalDuration:       700000000, // 700ms
	}
}

// SetupTestContext создает контекст для тестирования с timeout
func SetupTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

// RunWithTimeout выполняет функцию с timeout для тестов
func RunWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		fn()
	}()

	select {
	case <-done:
		// Функция завершилась вовремя
	case <-time.After(timeout):
		t.Fatalf("Test timed out after %v", timeout)
	}
}

// ExpectPanic проверяет что функция вызывает panic
func ExpectPanic(t *testing.T, fn func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic, but function completed normally")
		}
	}()
	fn()
}

// CompareJSON сравнивает два JSON объекта (для тестирования)
func CompareJSON(t *testing.T, expected, actual interface{}) {
	// TODO: Реализовать детальное сравнение JSON объектов
	assert.Equal(t, expected, actual)
}

// TestServer создает тестовый HTTP сервер
func TestServer(handler http.Handler) *httptest.Server {
	return httptest.NewServer(handler)
}

// CreateTestRequest создает HTTP запрос для тестирования
func CreateTestRequest(method, url string, body interface{}) (*http.Request, error) {
	// TODO: Реализовать создание HTTP запросов с JSON body
	return http.NewRequest(method, url, nil)
}

// MockHealthyOllamaClient создает простой mock клиент для тестов
func MockHealthyOllamaClient() *MockOllamaClient {
	return &MockOllamaClient{}
}

// MockFailingOllamaClient создает простой failing mock клиент
func MockFailingOllamaClient(errorMsg string) *MockOllamaClient {
	return &MockOllamaClient{}
}

