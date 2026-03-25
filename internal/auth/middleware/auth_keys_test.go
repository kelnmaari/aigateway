// Package middleware provides HTTP middleware for authentication
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/auth/apikey"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// setupTestRouter создает тестовый роутер с auth middleware
func setupTestRouter(t *testing.T) (*gin.Engine, *apikey.Manager, context.Context) {
	t.Helper()

	// Отключаем логи Gin в тестах
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	stor := storage.NewMemoryStorage()
	manager := apikey.NewManager(cfg, logger, stor)

	ctx := context.Background()
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	router := gin.New()
	router.Use(gin.Recovery())

	// Применяем auth middleware
	authenticator := NewAPIKeyAuthenticator(cfg, logger, manager)
	router.Use(authenticator.AuthenticationMiddleware())
	router.Use(authenticator.ModelAuthorizationMiddleware())

	// Добавляем тестовый endpoint
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		// Проверяем что ключ был аутентифицирован
		keyInfo, exists := c.Get("api_key_info")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no api key in context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"key_id":  keyInfo.(*models.APIKeyPublic).ID,
		})
	})

	router.GET("/v1/models", func(c *gin.Context) {
		keyInfo, exists := c.Get("api_key_info")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no api key in context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"models": []string{"model1", "model2"},
			"key_id": keyInfo.(*models.APIKeyPublic).ID,
		})
	})

	return router, manager, ctx
}

// TestAuthMiddleware_ValidAPIKey проверяет успешную аутентификацию
func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	router, manager, ctx := setupTestRouter(t)

	// Создаем тестовый API ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Integration test key",
		Models:      []string{"*"},
		Permissions: []string{"chat", "models"},
	}
	keyResp, err := manager.CreateAPIKey(ctx, req)
	require.NoError(t, err)

	// Создаем HTTP запрос с валидным API ключом
	body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

	// Выполняем запрос
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	// Проверяем результат
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["message"])
	assert.Equal(t, keyResp.APIKey.ID, response["key_id"])
}

// TestAuthMiddleware_MissingAPIKey проверяет отсутствие API ключа
func TestAuthMiddleware_MissingAPIKey(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Создаем HTTP запрос БЕЗ API ключа
	body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	// НЕ устанавливаем Authorization header

	// Выполняем запрос
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	// Проверяем результат
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorObj := response["error"].(map[string]any)
	assert.Equal(t, "missing_api_key", errorObj["code"])
}

// TestAuthMiddleware_InvalidAPIKey проверяет невалидный API ключ
func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	// Создаем HTTP запрос с невалидным API ключом
	body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer sk-proj-invalid-key-1234567890123456789012345678")

	// Выполняем запрос
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	// Проверяем результат
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorObj := response["error"].(map[string]any)
	assert.Equal(t, "invalid_api_key", errorObj["code"])
}

// TestAuthMiddleware_DisabledAPIKey проверяет отключенный API ключ
func TestAuthMiddleware_DisabledAPIKey(t *testing.T) {
	router, manager, ctx := setupTestRouter(t)

	// Создаем API ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Test",
		Models:      []string{"*"},
		Permissions: []string{"chat"},
	}
	keyResp, err := manager.CreateAPIKey(ctx, req)
	require.NoError(t, err)

	// Отключаем ключ через storage
	statusDisabled := models.APIKeyStatusDisabled
	updateReq := models.UpdateAPIKeyRequest{
		Status: &statusDisabled,
	}
	_, err = manager.UpdateAPIKey(ctx, keyResp.APIKey.ID, updateReq)
	require.NoError(t, err)

	// Пытаемся использовать отключенный ключ
	body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

	// Выполняем запрос
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	// Проверяем результат
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	errorObj := response["error"].(map[string]any)
	assert.Equal(t, "invalid_api_key", errorObj["code"])
}

// TestAuthMiddleware_ModelAuthorization проверяет авторизацию для моделей
func TestAuthMiddleware_ModelAuthorization(t *testing.T) {
	t.Run("access allowed model", func(t *testing.T) {
		router, manager, ctx := setupTestRouter(t)

		// Создаем ключ с доступом только к gpt-3.5-turbo
		req := models.CreateAPIKeyRequest{
			Name:        "Limited Key",
			Description: "Key with limited model access",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		keyResp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Запрос к разрешенной модели
		body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("access denied model", func(t *testing.T) {
		router, manager, ctx := setupTestRouter(t)

		// Создаем ключ с доступом только к gpt-3.5-turbo
		req := models.CreateAPIKeyRequest{
			Name:        "Limited Key",
			Description: "Key with limited model access",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		keyResp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Запрос к НЕразрешенной модели
		body := []byte(`{"model": "gpt-4", "messages": [{"role": "user", "content": "Hello"}]}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]any
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		errorObj := response["error"].(map[string]any)
		assert.Equal(t, "model_access_denied", errorObj["code"])
	})

	t.Run("wildcard model access", func(t *testing.T) {
		router, manager, ctx := setupTestRouter(t)

		// Создаем ключ с доступом ко всем моделям
		req := models.CreateAPIKeyRequest{
			Name:        "Admin Key",
			Description: "Key with wildcard access",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		keyResp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Запрос к любой модели должен пройти
		body := []byte(`{"model": "some-random-model", "messages": [{"role": "user", "content": "Hello"}]}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAuthMiddleware_MultipleRequests проверяет множественные запросы
func TestAuthMiddleware_MultipleRequests(t *testing.T) {
	router, manager, ctx := setupTestRouter(t)

	// Создаем API ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Test",
		Models:      []string{"*"},
		Permissions: []string{"chat", "models"},
	}
	keyResp, err := manager.CreateAPIKey(ctx, req)
	require.NoError(t, err)

	// Выполняем несколько запросов подряд
	for i := range 5 {
		body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}
}

// TestAuthMiddleware_DifferentEndpoints проверяет разные endpoints
func TestAuthMiddleware_DifferentEndpoints(t *testing.T) {
	router, manager, ctx := setupTestRouter(t)

	// Создаем API ключ с разными permissions
	req := models.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Test",
		Models:      []string{"*"},
		Permissions: []string{"chat", "models"},
	}
	keyResp, err := manager.CreateAPIKey(ctx, req)
	require.NoError(t, err)

	t.Run("chat endpoint", func(t *testing.T) {
		body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("models endpoint", func(t *testing.T) {
		httpReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAuthMiddleware_InvalidAuthorizationFormat проверяет неправильный формат заголовка
func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	testCases := []struct {
		name   string
		header string
	}{
		{
			name:   "missing bearer prefix",
			header: "sk-proj-test-key-1234567890123456789012345678",
		},
		{
			name:   "wrong prefix",
			header: "Basic sk-proj-test-key-1234567890123456789012345678",
		},
		{
			name:   "empty after bearer",
			header: "Bearer ",
		},
		{
			name:   "only bearer",
			header: "Bearer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
			httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", tc.header)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httpReq)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// TestAuthMiddleware_ConcurrentRequests проверяет concurrent безопасность
func TestAuthMiddleware_ConcurrentRequests(t *testing.T) {
	router, manager, ctx := setupTestRouter(t)

	// Создаем API ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Test Key",
		Description: "Test",
		Models:      []string{"*"},
		Permissions: []string{"chat"},
	}
	keyResp, err := manager.CreateAPIKey(ctx, req)
	require.NoError(t, err)

	// Запускаем 10 goroutines с параллельными запросами
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			body := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}`)
			httpReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httpReq)

			assert.Equal(t, http.StatusOK, w.Code, "Concurrent request %d should succeed", id)
			done <- true
		}(i)
	}

	// Ждем завершения всех goroutines
	for range 10 {
		<-done
	}
}

// BenchmarkAuthMiddleware_ValidateAPIKey бенчмарк для валидации API ключа
func BenchmarkAuthMiddleware_ValidateAPIKey(b *testing.B) {
	// Setup
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Auth:   config.AuthConfig{Enabled: true},
		Server: config.ServerConfig{Host: "localhost", Port: 8080},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	stor := storage.NewMemoryStorage()
	manager := apikey.NewManager(cfg, logger, stor)
	ctx := context.Background()
	_ = manager.Initialize(ctx)

	router := gin.New()
	authenticator := NewAPIKeyAuthenticator(cfg, logger, manager)
	router.Use(authenticator.AuthenticationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Создаем тестовый ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Benchmark Key",
		Description: "Benchmark",
		Models:      []string{"*"},
		Permissions: []string{"chat"},
	}
	keyResp, _ := manager.CreateAPIKey(ctx, req)

	// Benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := []byte(`{"test": "data"}`)
		httpReq := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		httpReq.Header.Set("Authorization", "Bearer "+keyResp.PlainKey)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httpReq)
	}
}
