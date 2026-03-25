// Package auth provides penetration tests for authentication system
package auth

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aigateway/internal/auth/apikey"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// setupPenetrationTestManager создает тестовый менеджер для penetration тестов
func setupPenetrationTestManager(t *testing.T) (*apikey.Manager, context.Context) {
	t.Helper()

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

	return manager, ctx
}

// TestPenetration_InvalidKeys тестирует различные варианты недействительных ключей
func TestPenetration_InvalidKeys(t *testing.T) {
	t.Run("completely fake keys", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		fakeKeys := []string{
			"sk-proj-fakekeyfakekeyfakekeyfakekeyfakekeyfakekeyfakekeyfakekeyfakekey",
			"sk-proj-11111111111111111111111111111111111111111111111111111111111111",
			"sk-proj-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"sk-proj-0000000000000000000000000000000000000000000000000000000000000000",
			"sk-proj-zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		}

		for _, fakeKey := range fakeKeys {
			result, err := manager.ValidateAPIKey(ctx, fakeKey)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Fake key should be invalid: %s", fakeKey)
		}
	})

	t.Run("keys from other systems", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Ключи, похожие на ключи других систем
		otherSystemKeys := []string{
			"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", // OpenAI style
			"AKIAIOSFODNN7EXAMPLE",                                // AWS style
			"ghp_1234567890abcdefghijklmnopqrstuvwxyz",            // GitHub style
			"AIzaSyD1234567890abcdefghijklmnopqrstuvw",            // Google API style
			"xox-1234567890-1234567890-1234567890",                // Slack style
		}

		for _, key := range otherSystemKeys {
			result, err := manager.ValidateAPIKey(ctx, key)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Key from other system should be invalid: %s", key)
		}
	})

	t.Run("malformed keys", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		malformedKeys := []string{
			"sk-proj-",                       // Only prefix
			"sk-proj-short",                  // Too short
			"sk-proj-invalid@#$%^&*()",       // Special characters
			"sk-proj-\x00\x00\x00\x00",       // Null bytes
			"sk-proj-\n\r\t",                 // Control characters
			"sk-proj-unicode™®©😀",            // Unicode characters
			"sk-proj-with spaces in it",      // Spaces
			"sk-proj-ЁЯШШ emoji ЁЯШШ in key", // Emojis
		}

		for _, key := range malformedKeys {
			result, err := manager.ValidateAPIKey(ctx, key)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Malformed key should be invalid")
		}
	})

	t.Run("empty and whitespace keys", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		emptyKeys := []string{
			"",
			" ",
			"   ",
			"\t",
			"\n",
			"\r\n",
			"          ",
		}

		for _, key := range emptyKeys {
			result, err := manager.ValidateAPIKey(ctx, key)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Empty/whitespace key should be invalid")
		}
	})
}

// TestPenetration_ExpiredKeys тестирует работу с истекшими ключами
func TestPenetration_ExpiredKeys(t *testing.T) {
	t.Run("expired key validation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Пытаемся создать ключ с уже истекшим сроком действия
		expiredTime := time.Now().Add(-24 * time.Hour)
		req := models.CreateAPIKeyRequest{
			Name:        "Expired Key",
			Description: "Key that should expire",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			ExpiresAt:   &expiredTime, // Истек вчера
		}

		// Система должна отклонить создание уже истекшего ключа
		_, err := manager.CreateAPIKey(ctx, req)
		assert.Error(t, err, "Should not allow creating already expired key")
		assert.Contains(t, err.Error(), "past", "Error should mention that date is in the past")
	})

	t.Run("key expiring soon", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ, который истечет через 1 секунду
		soonExpireTime := time.Now().Add(1 * time.Second)
		req := models.CreateAPIKeyRequest{
			Name:        "Soon Expiring Key",
			Description: "Key that will expire soon",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			ExpiresAt:   &soonExpireTime,
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Ключ должен быть валиден сейчас
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.True(t, result.Valid, "Key should be valid before expiration")

		// Ждем истечения срока действия
		time.Sleep(2 * time.Second)

		// Теперь ключ должен быть невалиден
		result, err = manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.False(t, result.Valid, "Key should be invalid after expiration")
	})

	t.Run("never expiring key", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ без срока действия (nil)
		req := models.CreateAPIKeyRequest{
			Name:        "Never Expiring Key",
			Description: "Key without expiration",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			ExpiresAt:   nil, // nil - не истекает
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Ключ должен быть валиден
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.True(t, result.Valid, "Never expiring key should be valid")
	})
}

// TestPenetration_DisabledKeys тестирует работу с отключенными ключами
func TestPenetration_DisabledKeys(t *testing.T) {
	t.Run("disabled key validation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Key to be disabled",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Ключ должен быть валиден
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.True(t, result.Valid, "Key should be valid initially")

		// Отключаем ключ
		disabledStatus := models.APIKeyStatusDisabled
		updateReq := models.UpdateAPIKeyRequest{
			Status: &disabledStatus,
		}
		_, err = manager.UpdateAPIKey(ctx, key.APIKey.ID, updateReq)
		require.NoError(t, err)

		// Теперь ключ должен быть невалиден
		result, err = manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.False(t, result.Valid, "Disabled key should be invalid")
	})

	t.Run("revoked key validation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Key to be revoked",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Отзываем ключ
		revokedStatus := models.APIKeyStatusRevoked
		updateReq := models.UpdateAPIKeyRequest{
			Status: &revokedStatus,
		}
		_, err = manager.UpdateAPIKey(ctx, key.APIKey.ID, updateReq)
		require.NoError(t, err)

		// Отозванный ключ должен быть невалиден
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.False(t, result.Valid, "Revoked key should be invalid")
	})
}

// TestPenetration_ModelRestrictions тестирует обход ограничений на модели
func TestPenetration_ModelRestrictions(t *testing.T) {
	t.Run("restricted model access attempt", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ с ограничением на определенные модели
		req := models.CreateAPIKeyRequest{
			Name:        "Restricted Key",
			Description: "Key restricted to specific models",
			Models:      []string{"llama3.1", "qwen2.5-coder"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полную информацию о ключе для проверки модели
		fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
		require.NoError(t, err)

		// Проверяем доступ к разрешенным моделям
		allowedModels := []string{"llama3.1", "qwen2.5-coder"}
		for _, model := range allowedModels {
			hasAccess := false
			for _, allowedModel := range fullKey.Models {
				if allowedModel == model || allowedModel == "*" {
					hasAccess = true
					break
				}
			}
			assert.True(t, hasAccess, "Should have access to allowed model: %s", model)
		}

		// Проверяем попытки доступа к запрещенным моделям
		forbiddenModels := []string{"gpt-4", "claude-3", "mistral", "codellama"}
		for _, model := range forbiddenModels {
			hasAccess := false
			for _, allowedModel := range fullKey.Models {
				if allowedModel == model || allowedModel == "*" {
					hasAccess = true
					break
				}
			}
			assert.False(t, hasAccess, "Should not have access to forbidden model: %s", model)
		}
	})

	t.Run("wildcard model access", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ с доступом ко всем моделям
		req := models.CreateAPIKeyRequest{
			Name:        "Wildcard Key",
			Description: "Key with access to all models",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полную информацию о ключе
		fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
		require.NoError(t, err)

		// Проверяем что есть wildcard
		hasWildcard := slices.Contains(fullKey.Models, "*")
		assert.True(t, hasWildcard, "Key should have wildcard model access")
	})
}

// TestPenetration_PermissionEscalation тестирует попытки повышения привилегий
func TestPenetration_PermissionEscalation(t *testing.T) {
	t.Run("limited permissions", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ только с правом на chat
		req := models.CreateAPIKeyRequest{
			Name:        "Limited Key",
			Description: "Key with limited permissions",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полную информацию о ключе
		fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
		require.NoError(t, err)

		// Проверяем что есть только chat permission
		hasChat := false
		hasAdmin := false

		for _, perm := range fullKey.Permissions {
			if perm == "chat" {
				hasChat = true
			}
			if perm == "admin" || perm == "*" {
				hasAdmin = true
			}
		}

		assert.True(t, hasChat, "Should have chat permission")
		assert.False(t, hasAdmin, "Should not have admin permission")
	})

	t.Run("admin permission", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ с admin правами
		req := models.CreateAPIKeyRequest{
			Name:        "Admin Key",
			Description: "Key with admin permissions",
			Models:      []string{"*"},
			Permissions: []string{"*"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Получаем полную информацию о ключе
		fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
		require.NoError(t, err)

		// Проверяем наличие wildcard permission
		hasWildcard := slices.Contains(fullKey.Permissions, "*")
		assert.True(t, hasWildcard, "Admin key should have wildcard permission")
	})
}

// TestPenetration_KeyEnumeration тестирует попытки перебора ключей
func TestPenetration_KeyEnumeration(t *testing.T) {
	t.Run("sequential key id guessing", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем несколько ключей
		for i := range 5 {
			req := models.CreateAPIKeyRequest{
				Name:        fmt.Sprintf("Key %d", i),
				Description: "Test key",
				Models:      []string{"*"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		// Пытаемся подобрать ID ключей
		guessedIDs := []string{
			"ak_1",
			"ak_2",
			"ak_3",
			"ak_100",
			"ak_1000",
			"key_1",
			"key_2",
		}

		successfulGuesses := 0
		for _, guessedID := range guessedIDs {
			_, err := manager.GetAPIKey(ctx, guessedID)
			if err == nil {
				successfulGuesses++
			}
		}

		// ID ключей должны быть достаточно случайными, чтобы не угадать
		assert.Equal(t, 0, successfulGuesses, "Should not be able to guess key IDs")
	})

	t.Run("brute force key generation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем реальный ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Real Key",
			Description: "Real key",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		realKey, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся подобрать ключ перебором
		// Тестируем только префикс + несколько символов (полный перебор занял бы вечность)
		attempts := 100
		successfulAttempts := 0

		for i := range attempts {
			guessKey := fmt.Sprintf("sk-proj-%064x", i)
			result, err := manager.ValidateAPIKey(ctx, guessKey)
			require.NoError(t, err)
			if result.Valid {
				successfulAttempts++
			}
		}

		// Перебор не должен быть успешным (кроме крайне маловероятного случая)
		assert.Equal(t, 0, successfulAttempts, "Brute force should not succeed")

		// Но реальный ключ должен работать
		result, err := manager.ValidateAPIKey(ctx, realKey.PlainKey)
		require.NoError(t, err)
		assert.True(t, result.Valid, "Real key should be valid")
	})
}

// TestPenetration_InjectionAttacks тестирует попытки injection атак
func TestPenetration_InjectionAttacks(t *testing.T) {
	t.Run("sql injection in key validation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		sqlInjectionKeys := []string{
			"sk-proj-' OR '1'='1",
			"sk-proj-'; DROP TABLE api_keys; --",
			"sk-proj-' UNION SELECT * FROM users--",
			"sk-proj-admin'--",
			"sk-proj-' OR 1=1--",
		}

		for _, injKey := range sqlInjectionKeys {
			result, err := manager.ValidateAPIKey(ctx, injKey)
			// Не должно быть паники или критических ошибок
			require.NoError(t, err)
			assert.False(t, result.Valid, "SQL injection should not work")
		}
	})

	t.Run("command injection in key name", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		commandInjectionNames := []string{
			"; rm -rf /",
			"$(whoami)",
			"`cat /etc/passwd`",
			"| ls -la",
			"&& curl malicious.com",
		}

		for _, name := range commandInjectionNames {
			req := models.CreateAPIKeyRequest{
				Name:        name,
				Description: "Test",
				Models:      []string{"*"},
				Permissions: []string{"chat"},
			}
			// Создание ключа не должно выполнять команды
			key, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, key)

			// Имя должно быть экранировано или очищено
			fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
			require.NoError(t, err)
			assert.NotEmpty(t, fullKey.Name)
		}
	})

	t.Run("path traversal in key description", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		pathTraversalDesc := []string{
			"../../../../../../etc/passwd",
			"../../../config/secrets.yaml",
			"..\\..\\..\\windows\\system32\\config",
			"./../.env",
		}

		for _, desc := range pathTraversalDesc {
			req := models.CreateAPIKeyRequest{
				Name:        "Test Key",
				Description: desc,
				Models:      []string{"*"},
				Permissions: []string{"chat"},
			}
			key, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, key)

			// Описание не должно приводить к чтению файлов
			fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
			require.NoError(t, err)
			// Описание должно сохраниться как строка, не выполняясь
			assert.Equal(t, desc, fullKey.Description)
		}
	})
}

// TestPenetration_RateLimitBypass тестирует попытки обхода rate limiting
func TestPenetration_RateLimitBypass(t *testing.T) {
	t.Run("rapid sequential requests", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ с низким лимитом
		req := models.CreateAPIKeyRequest{
			Name:        "Limited Key",
			Description: "Key with low rate limit",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 5,
				RequestsPerDay:    100,
			},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся сделать много запросов подряд
		validations := 0
		for range 20 {
			result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
			require.NoError(t, err)
			if result.Valid {
				validations++
			}
		}

		// Все валидации должны пройти (rate limiting применяется на уровне middleware, а не в ValidateAPIKey)
		assert.Equal(t, 20, validations, "All validations should succeed")
	})
}

// TestPenetration_TimingAttacks тестирует защиту от timing attacks
func TestPenetration_TimingAttacks(t *testing.T) {
	t.Run("constant time key validation", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем реальный ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		realKey, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Создаем несколько близких к реальному ключей
		similarKeys := []string{
			realKey.PlainKey[:len(realKey.PlainKey)-1] + "a",
			realKey.PlainKey[:len(realKey.PlainKey)-5] + "aaaaa",
			realKey.PlainKey[:len(realKey.PlainKey)-10] + "aaaaaaaaaa",
		}

		// Измеряем время валидации
		var realKeyTime time.Duration
		start := time.Now()
		_, _ = manager.ValidateAPIKey(ctx, realKey.PlainKey)
		realKeyTime = time.Since(start)

		for _, similarKey := range similarKeys {
			start := time.Now()
			_, _ = manager.ValidateAPIKey(ctx, similarKey)
			similarKeyTime := time.Since(start)

			// Время должно быть примерно одинаковым (bcrypt constant-time)
			timeDiff := realKeyTime - similarKeyTime
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}

			// Разница не должна быть слишком большой (допускаем 100ms variance)
			assert.Less(t, timeDiff, 100*time.Millisecond,
				"Timing should be constant for similar keys")
		}
	})
}

// TestPenetration_ConcurrentAbuse тестирует злоупотребление concurrent доступом
func TestPenetration_ConcurrentAbuse(t *testing.T) {
	t.Run("concurrent key deletion and usage", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся одновременно использовать и удалить ключ
		done := make(chan bool, 2)

		// Горутина 1: постоянно валидирует ключ
		go func() {
			for range 10 {
				_, _ = manager.ValidateAPIKey(ctx, key.PlainKey)
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Горутина 2: удаляет ключ
		go func() {
			time.Sleep(50 * time.Millisecond)
			_ = manager.DeleteAPIKey(ctx, key.APIKey.ID)
			done <- true
		}()

		// Ждем завершения обеих горутин
		<-done
		<-done

		// После удаления ключ должен быть невалиден
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.False(t, result.Valid, "Deleted key should be invalid")
	})

	t.Run("concurrent key updates race condition", func(t *testing.T) {
		manager, ctx := setupPenetrationTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся одновременно обновить ключ несколько раз
		done := make(chan bool, 10)

		for i := range 10 {
			go func(iteration int) {
				desc := fmt.Sprintf("Updated description %d", iteration)
				updateReq := models.UpdateAPIKeyRequest{
					Description: &desc,
				}
				_, _ = manager.UpdateAPIKey(ctx, key.APIKey.ID, updateReq)
				done <- true
			}(i)
		}

		// Ждем завершения всех обновлений
		for range 10 {
			<-done
		}

		// Ключ должен остаться валидным после всех обновлений
		result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
		require.NoError(t, err)
		assert.True(t, result.Valid, "Key should remain valid after concurrent updates")

		// Проверяем что ключ был обновлен
		fullKey, err := manager.GetAPIKey(ctx, key.APIKey.ID)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(fullKey.Description, "Updated description"),
			"Key description should be updated")
	})
}
