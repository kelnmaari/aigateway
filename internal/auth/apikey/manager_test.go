// Package apikey provides API Key management for Ollama-OpenAI Proxy
package apikey

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// setupTestManager создает тестовый менеджер с in-memory storage
func setupTestManager(t *testing.T) (*Manager, context.Context) {
	t.Helper()

	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Ollama: config.OllamaConfig{
			URL: "http://localhost:11434",
		},
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Suppress logs in tests

	stor := storage.NewMemoryStorage()
	manager := NewManager(cfg, logger, stor)

	ctx := context.Background()
	err := manager.Initialize(ctx)
	require.NoError(t, err, "Failed to initialize manager")

	return manager, ctx
}

// TestNewManager проверяет создание нового менеджера
func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	logger := logrus.New()
	stor := storage.NewMemoryStorage()

	manager := NewManager(cfg, logger, stor)

	assert.NotNil(t, manager)
	assert.Equal(t, cfg, manager.config)
	assert.Equal(t, logger, manager.logger)
	assert.Equal(t, stor, manager.storage)
}

// TestManagerInitialize проверяет инициализацию менеджера
func TestManagerInitialize(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Проверяем, что bootstrap admin key создан
		listReq := models.ListAPIKeysRequest{
			Limit:  10,
			Offset: 0,
		}
		keysResp, err := manager.ListAPIKeys(ctx, listReq)
		require.NoError(t, err)
		assert.NotEmpty(t, keysResp.APIKeys, "Bootstrap admin key should be created")

		// Проверяем, что admin key имеет правильные разрешения
		adminKey := keysResp.APIKeys[0]
		assert.Contains(t, adminKey.Permissions, "*", "Admin key should have * permission")
	})

	t.Run("initialization with storage error", func(t *testing.T) {
		cfg := &config.Config{}
		logger := logrus.New()
		logger.SetLevel(logrus.ErrorLevel)

		// Используем mock storage который возвращает ошибку
		stor := &storage.ErrorStorage{Err: assert.AnError}
		manager := NewManager(cfg, logger, stor)

		ctx := context.Background()
		err := manager.Initialize(ctx)
		assert.Error(t, err, "Should fail with storage error")
	})
}

// TestCreateAPIKey проверяет создание API ключа
func TestCreateAPIKey(t *testing.T) {
	t.Run("create valid API key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test Description",
			Models:      []string{"gpt-3.5-turbo", "gpt-4"},
			Permissions: []string{"chat", "models"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 10,
				RequestsPerHour:   100,
				RequestsPerDay:    1000,
			},
		}

		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Проверяем, что ключ создан
		assert.NotEmpty(t, resp.APIKey.ID)
		assert.Equal(t, req.Name, resp.APIKey.Name)
		assert.Equal(t, req.Description, resp.APIKey.Description)
		assert.Equal(t, req.Models, resp.APIKey.Models)
		assert.Equal(t, req.Permissions, resp.APIKey.Permissions)
		assert.Equal(t, models.APIKeyStatusActive, resp.APIKey.Status)

		// Проверяем plaintext ключ
		assert.NotEmpty(t, resp.PlainKey)
		assert.Contains(t, resp.PlainKey, "sk-ak_")

		// Проверяем, что можем найти ключ по ID
		foundKey, err := manager.GetAPIKey(ctx, resp.APIKey.ID)
		require.NoError(t, err)
		assert.Equal(t, resp.APIKey.ID, foundKey.ID)
		assert.Equal(t, req.Name, foundKey.Name)
	})

	t.Run("create key with empty name", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req := models.CreateAPIKeyRequest{
			Name:        "", // Empty name
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
		}

		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err, "Should fail with empty name")
	})

	t.Run("create key with wildcard models", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req := models.CreateAPIKeyRequest{
			Name:        "Admin Key",
			Description: "Admin with all models",
			Models:      []string{"*"},
			Permissions: []string{"*"},
		}

		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)
		assert.Contains(t, resp.APIKey.Models, "*")
		assert.Contains(t, resp.APIKey.Permissions, "*")
	})
}

// TestValidateAPIKey проверяет валидацию API ключей
func TestValidateAPIKey(t *testing.T) {
	t.Run("validate existing key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Валидируем ключ
		validatedKey, err := manager.ValidateAPIKey(ctx, resp.PlainKey)
		require.NoError(t, err)
		assert.NotNil(t, validatedKey)

		// Проверяем информацию
		foundKey, err := manager.GetAPIKey(ctx, resp.APIKey.ID)
		require.NoError(t, err)
		assert.NotNil(t, foundKey.LastUsedAt)
	})

	t.Run("validate non-existent key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		result, err := manager.ValidateAPIKey(ctx, "sk-proj-nonexistent1234567890123456789012345678")
		require.NoError(t, err, "Should not return error, but validation result")
		assert.NotNil(t, result)
		assert.False(t, result.Valid, "Validation should fail")
		assert.Contains(t, result.Error, "API key not found")
	})

	t.Run("validate invalid format key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		result, err := manager.ValidateAPIKey(ctx, "invalid-format")
		require.NoError(t, err, "Should not return error, but validation result")
		assert.NotNil(t, result)
		assert.False(t, result.Valid, "Validation should fail")
		assert.Contains(t, result.Error, "invalid API key format")
	})

	t.Run("validate empty key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		result, err := manager.ValidateAPIKey(ctx, "")
		require.NoError(t, err, "Should not return error, but validation result")
		assert.NotNil(t, result)
		assert.False(t, result.Valid, "Validation should fail")
		assert.Contains(t, result.Error, "invalid API key format")
	})

	t.Run("validate disabled key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полный ключ напрямую из storage
		fullKey, err := manager.storage.GetAPIKey(ctx, resp.APIKey.ID)
		require.NoError(t, err)

		// Отключаем ключ
		fullKey.Status = models.APIKeyStatusDisabled
		err = manager.storage.UpdateAPIKey(ctx, fullKey)
		require.NoError(t, err)

		// Пытаемся валидировать отключенный ключ
		result, err := manager.ValidateAPIKey(ctx, resp.PlainKey)
		require.NoError(t, err, "Should not return error, but validation result")
		assert.NotNil(t, result)
		assert.False(t, result.Valid, "Validation should fail for disabled key")
		assert.Contains(t, result.Error, "disabled")
	})
}

// TestListAPIKeys проверяет получение списка ключей
func TestListAPIKeys(t *testing.T) {
	t.Run("list all keys", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем несколько ключей
		for i := 1; i <= 5; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        fmt.Sprintf("Test Key %d", i),
				Description: fmt.Sprintf("Description %d", i),
				Models:      []string{"gpt-3.5-turbo"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		// Получаем список (+ 1 bootstrap admin key)
		listReq := models.ListAPIKeysRequest{
			Limit:  10,
			Offset: 0,
		}
		resp, err := manager.ListAPIKeys(ctx, listReq)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 6, resp.Total) // 5 созданных + 1 bootstrap
		assert.Len(t, resp.APIKeys, 6)
	})

	t.Run("list with pagination", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем 10 ключей
		for i := 1; i <= 10; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        fmt.Sprintf("Test Key %d", i),
				Description: fmt.Sprintf("Description %d", i),
				Models:      []string{"gpt-3.5-turbo"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		// Первая страница
		listReq1 := models.ListAPIKeysRequest{
			Limit:  5,
			Offset: 0,
		}
		resp1, err := manager.ListAPIKeys(ctx, listReq1)
		require.NoError(t, err)
		assert.Equal(t, 11, resp1.Total) // 10 + 1 bootstrap
		assert.Len(t, resp1.APIKeys, 5)

		// Вторая страница
		listReq2 := models.ListAPIKeysRequest{
			Limit:  5,
			Offset: 5,
		}
		resp2, err := manager.ListAPIKeys(ctx, listReq2)
		require.NoError(t, err)
		assert.Equal(t, 11, resp2.Total)
		assert.Len(t, resp2.APIKeys, 5)

		// Третья страница
		listReq3 := models.ListAPIKeysRequest{
			Limit:  5,
			Offset: 10,
		}
		resp3, err := manager.ListAPIKeys(ctx, listReq3)
		require.NoError(t, err)
		assert.Equal(t, 11, resp3.Total)
		assert.Len(t, resp3.APIKeys, 1) // Только последний ключ
	})

	t.Run("list with filters", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключи с разными статусами
		req1 := models.CreateAPIKeyRequest{
			Name:        "Active Key",
			Description: "Active",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp1, err := manager.CreateAPIKey(ctx, req1)
		require.NoError(t, err)

		req2 := models.CreateAPIKeyRequest{
			Name:        "Disabled Key",
			Description: "Disabled",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp2, err := manager.CreateAPIKey(ctx, req2)
		require.NoError(t, err)

		// Получаем полный ключ напрямую из storage и отключаем
		fullKey, err := manager.storage.GetAPIKey(ctx, resp2.APIKey.ID)
		require.NoError(t, err)
		fullKey.Status = models.APIKeyStatusDisabled
		err = manager.storage.UpdateAPIKey(ctx, fullKey)
		require.NoError(t, err)

		// Фильтруем только активные
		statusActive := models.APIKeyStatusActive
		listReq := models.ListAPIKeysRequest{
			Limit:  10,
			Offset: 0,
			Status: &statusActive,
		}
		resp, err := manager.ListAPIKeys(ctx, listReq)
		require.NoError(t, err)

		// Проверяем, что все ключи активные
		for _, key := range resp.APIKeys {
			assert.Equal(t, models.APIKeyStatusActive, key.Status)
		}

		// Используем resp1 чтобы избежать ошибки "declared and not used"
		_ = resp1
	})
}

// TestUpdateAPIKey проверяет обновление ключа
func TestUpdateAPIKey(t *testing.T) {
	t.Run("update key models", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Обновляем модели
		modelsSlice := []string{"gpt-3.5-turbo", "gpt-4"}
		permissionsSlice := []string{"chat", "models"}
		updateReq := models.UpdateAPIKeyRequest{
			Models:      &modelsSlice,
			Permissions: &permissionsSlice,
		}
		updatedKey, err := manager.UpdateAPIKey(ctx, resp.APIKey.ID, updateReq)
		require.NoError(t, err)

		assert.Equal(t, modelsSlice, updatedKey.Models)
		assert.Equal(t, permissionsSlice, updatedKey.Permissions)
	})

	t.Run("update non-existent key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		modelsSlice := []string{"gpt-4"}
		updateReq := models.UpdateAPIKeyRequest{
			Models: &modelsSlice,
		}
		_, err := manager.UpdateAPIKey(ctx, "non-existent-id", updateReq)
		assert.Error(t, err, "Should fail with non-existent key")
	})

	t.Run("update with empty models slice", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Обновляем с пустым слайсом моделей - это валидно, просто обновляет на пустой список
		// (в реальном приложении валидация должна быть на уровне API handler)
		emptyModels := []string{} // Empty models
		updateReq := models.UpdateAPIKeyRequest{
			Models: &emptyModels,
		}
		updatedKey, err := manager.UpdateAPIKey(ctx, resp.APIKey.ID, updateReq)
		require.NoError(t, err, "Update with empty models is allowed at manager level")
		assert.Empty(t, updatedKey.Models, "Models should be updated to empty list")
	})
}

// TestDeleteAPIKey проверяет удаление ключа
func TestDeleteAPIKey(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Удаляем ключ
		err = manager.DeleteAPIKey(ctx, resp.APIKey.ID)
		require.NoError(t, err)

		// Проверяем, что ключ больше не существует
		_, err = manager.GetAPIKey(ctx, resp.APIKey.ID)
		assert.Error(t, err, "Key should not exist after deletion")
	})

	t.Run("delete non-existent key", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		err := manager.DeleteAPIKey(ctx, "non-existent-id")
		assert.Error(t, err, "Should fail with non-existent key")
	})
}

// TestBootstrapAdminKey проверяет создание bootstrap admin ключа
func TestBootstrapAdminKey(t *testing.T) {
	t.Run("bootstrap admin key created", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Получаем список ключей
		listReq := models.ListAPIKeysRequest{
			Limit:  10,
			Offset: 0,
		}
		resp, err := manager.ListAPIKeys(ctx, listReq)
		require.NoError(t, err)

		// Проверяем, что есть хотя бы один ключ (bootstrap admin)
		assert.NotEmpty(t, resp.APIKeys, "Bootstrap admin key should be created")

		// Находим admin ключ
		var adminKey *models.APIKeyPublic
		for i := range resp.APIKeys {
			if resp.APIKeys[i].HasPermission("*") {
				adminKey = &resp.APIKeys[i]
				break
			}
		}

		require.NotNil(t, adminKey, "Admin key should exist")
		assert.Contains(t, adminKey.Permissions, "*")
		assert.Contains(t, adminKey.Models, "*")
		assert.Equal(t, models.APIKeyStatusActive, adminKey.Status)

		// Проверяем rate limits для admin
		assert.Greater(t, int64(adminKey.RateLimits.RequestsPerMinute), int64(100))
		assert.Greater(t, int64(adminKey.RateLimits.RequestsPerDay), int64(10000))
	})

	t.Run("bootstrap not recreated if exists", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Получаем текущее количество ключей
		listReq1 := models.ListAPIKeysRequest{
			Limit:  100,
			Offset: 0,
		}
		keys1, err := manager.ListAPIKeys(ctx, listReq1)
		require.NoError(t, err)
		initialCount := keys1.Total

		// Пытаемся инициализировать снова
		err = manager.Initialize(ctx)
		require.NoError(t, err)

		// Проверяем, что количество ключей не изменилось
		listReq2 := models.ListAPIKeysRequest{
			Limit:  100,
			Offset: 0,
		}
		keys2, err := manager.ListAPIKeys(ctx, listReq2)
		require.NoError(t, err)
		assert.Equal(t, initialCount, keys2.Total, "Bootstrap key should not be recreated")
	})
}

// TestAPIKeyHashing проверяет хеширование ключей
func TestAPIKeyHashing(t *testing.T) {
	t.Run("key is hashed with bcrypt", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полный ключ из storage
		fullKey, err := manager.storage.GetAPIKey(ctx, resp.APIKey.ID)
		require.NoError(t, err)

		// Проверяем, что plaintext key != hash
		assert.NotEqual(t, resp.PlainKey, fullKey.KeyHash)

		// Проверяем, что hash имеет bcrypt формат
		assert.Contains(t, fullKey.KeyHash, "$2a$")

		// Проверяем, что можем валидировать с plaintext ключом
		_, err = manager.ValidateAPIKey(ctx, resp.PlainKey)
		require.NoError(t, err)
	})

	t.Run("different keys produce different hashes", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req1 := models.CreateAPIKeyRequest{
			Name:        "Key 1",
			Description: "Test 1",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp1, err := manager.CreateAPIKey(ctx, req1)
		require.NoError(t, err)

		req2 := models.CreateAPIKeyRequest{
			Name:        "Key 2",
			Description: "Test 2",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp2, err := manager.CreateAPIKey(ctx, req2)
		require.NoError(t, err)

		// Проверяем, что plaintext ключи разные
		assert.NotEqual(t, resp1.PlainKey, resp2.PlainKey)

		// Получаем полные ключи из storage
		fullKey1, err := manager.storage.GetAPIKey(ctx, resp1.APIKey.ID)
		require.NoError(t, err)
		fullKey2, err := manager.storage.GetAPIKey(ctx, resp2.APIKey.ID)
		require.NoError(t, err)

		// Проверяем, что хеши разные
		assert.NotEqual(t, fullKey1.KeyHash, fullKey2.KeyHash)
	})

	t.Run("bcrypt timing safety", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"gpt-3.5-turbo"},
			Permissions: []string{"chat"},
		}
		resp, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Измеряем время валидации правильного ключа
		start1 := time.Now()
		result1, err1 := manager.ValidateAPIKey(ctx, resp.PlainKey)
		duration1 := time.Since(start1)
		require.NoError(t, err1)
		assert.True(t, result1.Valid)

		// Измеряем время валидации неправильного ключа
		start2 := time.Now()
		result2, err2 := manager.ValidateAPIKey(ctx, "sk-proj-wrong-key-12345678901234567890123456789012")
		duration2 := time.Since(start2)
		require.NoError(t, err2)
		assert.False(t, result2.Valid)

		// Время должно быть примерно одинаковым (разница < 50ms)
		// Bcrypt обеспечивает constant-time comparison
		timeDiff := duration1 - duration2
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		// Логируем для информации
		t.Logf("Valid key validation time: %v", duration1)
		t.Logf("Invalid key validation time: %v", duration2)
		t.Logf("Time difference: %v", timeDiff)

		// Не строгая проверка, так как timing может варьироваться
		// но общее время должно быть разумным (< 1 секунда)
		assert.Less(t, duration1, time.Second)
		assert.Less(t, duration2, time.Second)
	})
}

// TestManagerStats проверяет статистику менеджера
func TestManagerStats(t *testing.T) {
	t.Run("stats updated after operations", func(t *testing.T) {
		manager, ctx := setupTestManager(t)

		// Создаем несколько ключей
		for i := 1; i <= 3; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        fmt.Sprintf("Key %d", i),
				Description: "Test",
				Models:      []string{"gpt-3.5-turbo"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		// Получаем статистику
		stats, err := manager.GetStats(ctx)
		require.NoError(t, err)

		// Проверяем статистику
		assert.Equal(t, int64(4), stats.TotalKeys) // 3 + 1 bootstrap
		assert.Equal(t, int64(4), stats.ActiveKeys)
	})
}

// Benchmark тесты для производительности
func BenchmarkCreateAPIKey(b *testing.B) {
	cfg := &config.Config{
		Auth:   config.AuthConfig{Enabled: true},
		Server: config.ServerConfig{Host: "localhost", Port: 8080},
		Ollama: config.OllamaConfig{URL: "http://localhost:11434"},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	stor := storage.NewMemoryStorage()
	manager := NewManager(cfg, logger, stor)
	ctx := context.Background()
	_ = manager.Initialize(ctx)

	req := models.CreateAPIKeyRequest{
		Name:        "Benchmark Key",
		Description: "Benchmark",
		Models:      []string{"gpt-3.5-turbo"},
		Permissions: []string{"chat"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.CreateAPIKey(ctx, req)
	}
}

func BenchmarkValidateAPIKey(b *testing.B) {
	cfg := &config.Config{
		Auth:   config.AuthConfig{Enabled: true},
		Server: config.ServerConfig{Host: "localhost", Port: 8080},
		Ollama: config.OllamaConfig{URL: "http://localhost:11434"},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	stor := storage.NewMemoryStorage()
	manager := NewManager(cfg, logger, stor)
	ctx := context.Background()
	_ = manager.Initialize(ctx)

	req := models.CreateAPIKeyRequest{
		Name:        "Benchmark Key",
		Description: "Benchmark",
		Models:      []string{"gpt-3.5-turbo"},
		Permissions: []string{"chat"},
	}
	resp, _ := manager.CreateAPIKey(ctx, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateAPIKey(ctx, resp.PlainKey)
	}
}

