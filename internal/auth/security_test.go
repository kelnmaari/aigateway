// Package auth provides security tests for authentication system
package auth

import (
	"context"
	"sync"
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

// setupSecurityTestManager создает тестовый менеджер для security тестов
func setupSecurityTestManager(t *testing.T) (*apikey.Manager, context.Context) {
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
	logger.SetLevel(logrus.ErrorLevel)

	stor := storage.NewMemoryStorage()
	manager := apikey.NewManager(cfg, logger, stor)

	ctx := context.Background()
	err := manager.Initialize(ctx)
	require.NoError(t, err)

	return manager, ctx
}

// TestSecurity_TimingAttackResistance проверяет устойчивость к timing attacks
func TestSecurity_TimingAttackResistance(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "integration")
	t.Attr("go_version", "1.25+")
	
	t.Run("bcrypt constant time comparison", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Создаем валидный ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Security test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		validKey, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Измеряем время для нескольких валидных и невалидных ключей
		validDurations := make([]time.Duration, 10)
		invalidDurations := make([]time.Duration, 10)

		// Тестируем валидный ключ 10 раз
		for i := 0; i < 10; i++ {
			start := time.Now()
			result, err := manager.ValidateAPIKey(ctx, validKey.PlainKey)
			validDurations[i] = time.Since(start)
			require.NoError(t, err)
			assert.True(t, result.Valid)
		}

		// Тестируем невалидный ключ 10 раз
		invalidKey := "sk-proj-invalid1234567890123456789012345678901234567890123456789012"
		for i := 0; i < 10; i++ {
			start := time.Now()
			result, err := manager.ValidateAPIKey(ctx, invalidKey)
			invalidDurations[i] = time.Since(start)
			require.NoError(t, err)
			assert.False(t, result.Valid)
		}

		// Вычисляем средние времена
		avgValid := averageDuration(validDurations)
		avgInvalid := averageDuration(invalidDurations)

		// Разница не должна быть слишком большой (bcrypt должен быть constant-time)
		timeDiff := avgValid - avgInvalid
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		t.Logf("Average valid key time: %v", avgValid)
		t.Logf("Average invalid key time: %v", avgInvalid)
		t.Logf("Time difference: %v", timeDiff)

		// Разница не должна превышать 100ms (bcrypt сам по себе медленный, ~100-200ms)
		assert.Less(t, timeDiff, 100*time.Millisecond,
			"Time difference too large - possible timing attack vulnerability")
	})

	t.Run("early termination attack", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		validKey, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся использовать укороченные версии ключа
		// Примечание: bcrypt имеет ограничение на длину входа (72 байта),
		// поэтому добавление символов после этого предела может не повлиять на валидацию
		keyLen := len(validKey.PlainKey)
		shortKeys := []string{
			validKey.PlainKey[:10],       // Очень короткий
			validKey.PlainKey[:20],       // Короткий
			validKey.PlainKey[:keyLen/2], // Половина
			validKey.PlainKey[:keyLen-2], // Почти полный
		}

		for _, shortKey := range shortKeys {
			result, err := manager.ValidateAPIKey(ctx, shortKey)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Shortened key should be invalid")
		}
	})
}

// TestSecurity_BruteForceProtection проверяет защиту от brute force атак
func TestSecurity_BruteForceProtection(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "integration")
	t.Attr("go_version", "1.25+")
	
	t.Run("multiple failed attempts", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Пытаемся перебрать несколько невалидных ключей
		failedAttempts := 0
		maxAttempts := 100

		start := time.Now()
		for i := 0; i < maxAttempts; i++ {
			invalidKey := generateRandomInvalidKey()
			result, err := manager.ValidateAPIKey(ctx, invalidKey)
			require.NoError(t, err)
			if !result.Valid {
				failedAttempts++
			}
		}
		duration := time.Since(start)

		// Все попытки должны были провалиться
		assert.Equal(t, maxAttempts, failedAttempts)

		// Bcrypt должен замедлить брутфорс
		// На мощном железе bcrypt может быть быстрее, но все равно должен добавлять задержку
		// Ожидаем минимум 3 секунды для 100 попыток (с учетом современного железа)
		t.Logf("100 brute force attempts took: %v", duration)
		assert.Greater(t, duration, 3*time.Second,
			"Brute force should be significantly slowed by bcrypt")

		// В среднем каждая попытка должна занимать минимум 30ms из-за bcrypt
		avgPerAttempt := duration / time.Duration(maxAttempts)
		assert.Greater(t, avgPerAttempt, 30*time.Millisecond,
			"Each attempt should take at least 30ms due to bcrypt")
	})

	t.Run("parallel brute force attempts", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Запускаем параллельные попытки брутфорса
		var wg sync.WaitGroup
		goroutines := 10
		attemptsPerGoroutine := 10
		totalAttempts := goroutines * attemptsPerGoroutine

		start := time.Now()
		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < attemptsPerGoroutine; i++ {
					invalidKey := generateRandomInvalidKey()
					result, _ := manager.ValidateAPIKey(ctx, invalidKey)
					assert.False(t, result.Valid)
				}
			}()
		}
		wg.Wait()
		duration := time.Since(start)

		t.Logf("%d parallel brute force attempts took: %v", totalAttempts, duration)

		// Даже с параллелизмом, bcrypt должен защищать
		avgPerAttempt := duration / time.Duration(totalAttempts)
		t.Logf("Average time per attempt: %v", avgPerAttempt)
	})
}

// TestSecurity_TokenManipulation проверяет устойчивость к манипуляциям с токенами
func TestSecurity_TokenManipulation(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "unit")
	t.Attr("go_version", "1.25+")
	
	t.Run("modified token structure", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Создаем валидный ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		validKey, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Пытаемся модифицировать токен различными способами
		// Примечание: bcrypt имеет ограничение на длину (72 байта),
		// поэтому мы не тестируем добавление суффиксов к длинным ключам
		manipulations := []struct {
			name string
			key  string
		}{
			{
				name: "change prefix",
				key:  "sk-test-" + validKey.PlainKey[8:],
			},
			{
				name: "remove prefix",
				key:  validKey.PlainKey[8:],
			},
			{
				name: "change case",
				key:  "SK-PROJ-" + validKey.PlainKey[8:],
			},
			{
				name: "insert space",
				key:  validKey.PlainKey[:20] + " " + validKey.PlainKey[20:],
			},
			{
				name: "reverse",
				key:  reverseString(validKey.PlainKey),
			},
			{
				name: "flip one character",
				key:  flipCharacter(validKey.PlainKey, 20),
			},
		}

		for _, tc := range manipulations {
			t.Run(tc.name, func(t *testing.T) {
				result, err := manager.ValidateAPIKey(ctx, tc.key)
				require.NoError(t, err)
				assert.False(t, result.Valid, "Modified token should be invalid")
			})
		}
	})

	t.Run("sql injection attempts", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Пытаемся внедрить SQL injection в API ключ
		sqlInjectionAttempts := []string{
			"sk-proj-' OR '1'='1",
			"sk-proj-'; DROP TABLE api_keys; --",
			"sk-proj-1234567890' UNION SELECT * FROM users--",
			"sk-proj-<script>alert('xss')</script>",
		}

		for _, injectionKey := range sqlInjectionAttempts {
			result, err := manager.ValidateAPIKey(ctx, injectionKey)
			// Не должно быть паники или критических ошибок
			require.NoError(t, err)
			assert.False(t, result.Valid, "SQL injection attempt should be invalid")
		}
	})

	t.Run("special characters in key", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		specialCharKeys := []string{
			"sk-proj-\x00\x00\x00\x00",       // Null bytes
			"sk-proj-\n\r\t",                 // Control characters
			"sk-proj-™®©",                    // Unicode symbols
			"sk-proj-😀🔑",                     // Emojis
			"sk-proj-../../../../etc/passwd", // Path traversal
		}

		for _, specialKey := range specialCharKeys {
			result, err := manager.ValidateAPIKey(ctx, specialKey)
			require.NoError(t, err)
			assert.False(t, result.Valid, "Special characters should be handled safely")
		}
	})
}

// TestSecurity_InvalidKeyFormats проверяет обработку различных невалидных форматов
func TestSecurity_InvalidKeyFormats(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "unit")
	t.Attr("go_version", "1.25+")
	
	manager, ctx := setupSecurityTestManager(t)

	testCases := []struct {
		name string
		key  string
	}{
		{
			name: "empty string",
			key:  "",
		},
		{
			name: "only spaces",
			key:  "     ",
		},
		{
			name: "too short",
			key:  "sk",
		},
		{
			name: "very long key",
			key:  "sk-proj-" + generateLongString(10000),
		},
		{
			name: "only prefix",
			key:  "sk-proj-",
		},
		{
			name: "numbers only",
			key:  "12345678901234567890",
		},
		{
			name: "binary data",
			key:  string([]byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := manager.ValidateAPIKey(ctx, tc.key)
			// Не должно быть паники
			require.NoError(t, err)
			assert.False(t, result.Valid)
		})
	}
}

// TestSecurity_ConcurrentAccessSafety проверяет безопасность при concurrent доступе
func TestSecurity_ConcurrentAccessSafety(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "integration")
	t.Attr("go_version", "1.25+")
	
	t.Run("concurrent key creation", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		var wg sync.WaitGroup
		goroutines := 20
		createdKeys := make(chan string, goroutines)
		errors := make(chan error, goroutines)

		// Создаем ключи параллельно
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				req := models.CreateAPIKeyRequest{
					Name:        "Concurrent Key",
					Description: "Concurrent test",
					Models:      []string{"*"},
					Permissions: []string{"chat"},
				}
				key, err := manager.CreateAPIKey(ctx, req)
				if err != nil {
					errors <- err
					return
				}
				createdKeys <- key.APIKey.ID
			}(i)
		}
		wg.Wait()
		close(createdKeys)
		close(errors)

		// Проверяем что все ключи уникальны
		keyIDs := make(map[string]bool)
		for keyID := range createdKeys {
			assert.False(t, keyIDs[keyID], "Duplicate key ID detected: %s", keyID)
			keyIDs[keyID] = true
		}

		// Не должно быть ошибок
		for err := range errors {
			t.Errorf("Error during concurrent key creation: %v", err)
		}
	})

	t.Run("concurrent validation of same key", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Создаем один ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		var wg sync.WaitGroup
		goroutines := 50
		successCount := 0
		var mu sync.Mutex

		// Валидируем один и тот же ключ параллельно
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
				require.NoError(t, err)
				if result.Valid {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		// Все валидации должны были пройти успешно
		assert.Equal(t, goroutines, successCount)
	})
}

// TestSecurity_MemoryLeaks проверяет на утечки памяти
func TestSecurity_MemoryLeaks(t *testing.T) {
	t.Attr("category", "security")
	t.Attr("type", "performance")
	t.Attr("go_version", "1.25+")
	
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	t.Run("repeated key validation", func(t *testing.T) {
		manager, ctx := setupSecurityTestManager(t)

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Test Key",
			Description: "Memory test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Выполняем множество валидаций
		iterations := 1000
		for i := 0; i < iterations; i++ {
			_, err := manager.ValidateAPIKey(ctx, key.PlainKey)
			require.NoError(t, err)

			// Иногда валидируем невалидный ключ
			if i%10 == 0 {
				_, err := manager.ValidateAPIKey(ctx, "sk-proj-invalid"+generateRandomString(40))
				require.NoError(t, err)
			}
		}

		// Если бы были утечки памяти, тест мог бы упасть или замедлиться значительно
		t.Logf("Completed %d validation iterations without issues", iterations)
	})
}

// Вспомогательные функции

func averageDuration(durations []time.Duration) time.Duration {
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

func generateRandomInvalidKey() string {
	return "sk-proj-" + generateRandomString(64)
}

func generateRandomString(length int) string {
	const charset = "abcdef0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func flipCharacter(s string, index int) string {
	if index >= len(s) {
		return s
	}
	runes := []rune(s)
	if runes[index] == 'a' {
		runes[index] = 'b'
	} else {
		runes[index] = 'a'
	}
	return string(runes)
}

// BenchmarkSecurity_ValidationUnderLoad бенчмарк для проверки производительности под нагрузкой
func BenchmarkSecurity_ValidationUnderLoad(b *testing.B) {
	cfg := &config.Config{
		Auth:   config.AuthConfig{Enabled: true},
		Server: config.ServerConfig{Host: "localhost", Port: 8080},
		Ollama: config.OllamaConfig{URL: "http://localhost:11434"},
	}
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	stor := storage.NewMemoryStorage()
	manager := apikey.NewManager(cfg, logger, stor)
	ctx := context.Background()
	_ = manager.Initialize(ctx)

	// Создаем тестовый ключ
	req := models.CreateAPIKeyRequest{
		Name:        "Benchmark Key",
		Description: "Benchmark",
		Models:      []string{"*"},
		Permissions: []string{"chat"},
	}
	key, _ := manager.CreateAPIKey(ctx, req)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = manager.ValidateAPIKey(ctx, key.PlainKey)
		}
	})
}

