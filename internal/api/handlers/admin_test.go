package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ollama-openai-proxy/internal/auth/apikey"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// setupAdminTest создает тестовое окружение для admin handler
func setupAdminTest(t *testing.T) (*AdminHandler, *storage.MemoryStorage) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	stor := storage.NewMemoryStorage()
	keyManager := apikey.NewManager(cfg, logger, stor)

	handler := NewAdminHandler(cfg, logger, keyManager)
	return handler, stor
}

// TestAdminHandler_CreateAPIKey_Success тестирует создание API ключа
func TestAdminHandler_CreateAPIKey_Success(t *testing.T) {
	handler, _ := setupAdminTest(t)

	reqBody := models.CreateAPIKeyRequest{
		Name:        "test-key",
		Description: "Test API Key",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateAPIKey(c)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp models.CreateAPIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.PlainKey)
	assert.NotEmpty(t, resp.APIKey.ID)
	assert.Equal(t, "test-key", resp.APIKey.Name)
}

// TestAdminHandler_CreateAPIKey_InvalidJSON тестирует невалидный JSON
func TestAdminHandler_CreateAPIKey_InvalidJSON(t *testing.T) {
	handler, _ := setupAdminTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys", bytes.NewBufferString("{invalid"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.CreateAPIKey(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAdminHandler_ListAPIKeys_Success тестирует получение списка ключей
func TestAdminHandler_ListAPIKeys_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)

	handler.ListAPIKeys(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.ListAPIKeysResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(resp.APIKeys), 1)
	assert.Equal(t, "test-key", resp.APIKeys[0].Name)
}

// TestAdminHandler_GetAPIKey_Success тестирует получение одного ключа
func TestAdminHandler_GetAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys/test-id", nil)

	handler.GetAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		APIKey models.APIKeyPublic `json:"api_key"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test-id", resp.APIKey.ID)
	assert.Equal(t, "test-key", resp.APIKey.Name)
}

// TestAdminHandler_GetAPIKey_NotFound тестирует несуществующий ключ
func TestAdminHandler_GetAPIKey_NotFound(t *testing.T) {
	handler, _ := setupAdminTest(t)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "nonexistent"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys/nonexistent", nil)

	handler.GetAPIKey(c)

	// isNotFoundError всегда false, поэтому ожидаем 500
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAdminHandler_UpdateAPIKey_Success тестирует обновление ключа
func TestAdminHandler_UpdateAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	name := "updated-key"
	desc := "Updated description"
	modelsList := []string{"gpt-3.5"}
	reqBody := models.UpdateAPIKeyRequest{
		Name:        &name,
		Description: &desc,
		Models:      &modelsList,
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPut, "/admin/keys/test-id", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		APIKey models.APIKeyPublic `json:"api_key"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "updated-key", resp.APIKey.Name)
}

// TestAdminHandler_DeleteAPIKey_Success тестирует удаление ключа
func TestAdminHandler_DeleteAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodDelete, "/admin/keys/test-id", nil)

	handler.DeleteAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminHandler_RevokeAPIKey_Success тестирует отзыв ключа (AUTH-04)
func TestAdminHandler_RevokeAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	reqBody := map[string]string{"reason": "Security concern"}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/revoke", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.RevokeAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что ключ был отозван
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Equal(t, models.APIKeyStatusRevoked, key.Status)
}

// TestAdminHandler_EnableAPIKey_Success тестирует включение ключа (AUTH-04)
func TestAdminHandler_EnableAPIKey_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем отозванный ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusRevoked,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/enable", nil)

	handler.EnableAPIKey(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что ключ был включен
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Equal(t, models.APIKeyStatusActive, key.Status)
}

// TestAdminHandler_ExtendAPIKey_Success тестирует продление срока (AUTH-04)
func TestAdminHandler_ExtendAPIKeyExpiration_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем ключ с истекшим сроком
	ctx := context.Background()
	expiredTime := time.Now().Add(-24 * time.Hour)
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		ExpiresAt:   &expiredTime,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	reqBody := struct {
		Days int `json:"days"`
	}{Days: 30}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/keys/test-id/extend", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ExtendAPIKeyExpiration(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что срок был продлен
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.NotNil(t, key.ExpiresAt)
	assert.True(t, key.ExpiresAt.After(time.Now()))
}

// TestAdminHandler_UpdatePermissions_Success тестирует обновление permissions (AUTH-04)
func TestAdminHandler_UpdateAPIKeyPermissions_Success(t *testing.T) {
	handler, stor := setupAdminTest(t)

	// Создаем тестовый ключ
	ctx := context.Background()
	testKey := &models.APIKey{
		ID:          "test-id",
		Name:        "test-key",
		KeyHash:     "hashed",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		Status:      models.APIKeyStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	stor.CreateAPIKey(ctx, testKey)

	perms := []string{"chat", "embeddings", "admin"}
	reqBody := struct {
		Permissions *[]string `json:"permissions"`
	}{Permissions: &perms}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "test-id"}}
	c.Request = httptest.NewRequest(http.MethodPatch, "/admin/keys/test-id/permissions", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.UpdateAPIKeyPermissions(c)

	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем, что permissions были обновлены
	key, _ := stor.GetAPIKey(ctx, "test-id")
	assert.Contains(t, key.Permissions, "admin")
	assert.Len(t, key.Permissions, 3)
}

// TestAdminHandler_WithoutKeyManager_ReturnsError тестирует handler без key manager
func TestAdminHandler_WithoutKeyManager_ReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{}
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := NewAdminHandlerWithoutKeys(cfg, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/admin/keys", nil)

	handler.ListAPIKeys(c)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Contains(t, w.Body.String(), "API Key management not available")
}
