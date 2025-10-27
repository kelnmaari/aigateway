// Package auth provides performance tests for rate limiting and API key management
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
	"aigateway/internal/auth/ratelimit"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// setupPerformanceTestManager создает тестовый менеджер для performance тестов
func setupPerformanceTestManager(t *testing.T) (*apikey.Manager, context.Context) {
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

// setupRateLimiter создает rate limiter для тестов
func setupRateLimiter(t *testing.T) *ratelimit.Limiter {
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
	cfg.Auth.RateLimiting.Enabled = true

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	return ratelimit.NewLimiter(cfg, logger)
}

// TestPerformance_RateLimiting_MultipleKeys тестирует rate limiting с множественными ключами
func TestPerformance_RateLimiting_MultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("concurrent requests with different keys", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)
		limiter := setupRateLimiter(t)
		defer limiter.Stop()

		// Создаем несколько ключей с разными лимитами
		keys := make([]*models.CreateAPIKeyResponse, 5)
		for i := 0; i < 5; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        "Performance Test Key",
				Description: "Performance test",
				Models:      []string{"*"},
				Permissions: []string{"chat"},
				RateLimits: &models.RateLimits{
					RequestsPerMinute: 100,
					RequestsPerDay:    1000,
				},
			}
			key, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
			keys[i] = key
		}

		// Запускаем параллельные запросы с разными ключами
		var wg sync.WaitGroup
		totalRequests := 500
		requestsPerKey := totalRequests / len(keys)
		successCount := 0
		rateLimitedCount := 0
		var mu sync.Mutex

		start := time.Now()

		for i, key := range keys {
			wg.Add(1)
			go func(apiKey *models.APIKeyPublic, idx int) {
				defer wg.Done()

				for j := 0; j < requestsPerKey; j++ {
					result := limiter.CheckRateLimit(ctx, apiKey.ID, apiKey, 0)

					mu.Lock()
					if result.Allowed {
						successCount++
					} else {
						rateLimitedCount++
					}
					mu.Unlock()

					// Небольшая задержка между запросами
					time.Sleep(10 * time.Millisecond)
				}
			}(&key.APIKey, i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Total requests: %d", totalRequests)
		t.Logf("Successful requests: %d", successCount)
		t.Logf("Rate limited requests: %d", rateLimitedCount)
		t.Logf("Duration: %v", duration)
		t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

		// Проверяем что хотя бы часть запросов прошла
		assert.Greater(t, successCount, 0, "Some requests should succeed")
	})

	t.Run("burst traffic handling", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)
		limiter := setupRateLimiter(t)
		defer limiter.Stop()

		// Создаем ключ с низким лимитом
		req := models.CreateAPIKeyRequest{
			Name:        "Burst Test Key",
			Description: "Burst test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 10, // Низкий лимит для теста burst
				RequestsPerDay:    100,
			},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Отправляем burst запросов (больше чем лимит)
		burstSize := 20
		successCount := 0
		rateLimitedCount := 0

		start := time.Now()
		for i := 0; i < burstSize; i++ {
			result := limiter.CheckRateLimit(ctx, key.APIKey.ID, &key.APIKey, 0)
			if result.Allowed {
				successCount++
			} else {
				rateLimitedCount++
			}
		}
		duration := time.Since(start)

		t.Logf("Burst size: %d", burstSize)
		t.Logf("Successful requests: %d", successCount)
		t.Logf("Rate limited requests: %d", rateLimitedCount)
		t.Logf("Duration: %v", duration)

		// Проверяем что rate limiting работает
		assert.LessOrEqual(t, successCount, req.RateLimits.RequestsPerMinute,
			"Should not exceed rate limit")
		assert.Greater(t, rateLimitedCount, 0,
			"Some requests should be rate limited")
	})

	t.Run("sustained load testing", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)
		limiter := setupRateLimiter(t)
		defer limiter.Stop()

		// Создаем ключ с разумным лимитом
		req := models.CreateAPIKeyRequest{
			Name:        "Sustained Load Key",
			Description: "Sustained load test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 60, // 1 запрос в секунду
				RequestsPerDay:    1000,
			},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Отправляем запросы с постоянной скоростью (1 запрос в секунду)
		duration := 5 * time.Second
		interval := 100 * time.Millisecond // 10 запросов в секунду
		expectedRequests := int(duration / interval)

		successCount := 0
		rateLimitedCount := 0

		start := time.Now()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		done := time.After(duration)
		for {
			select {
			case <-ticker.C:
				result := limiter.CheckRateLimit(ctx, key.APIKey.ID, &key.APIKey, 0)
				if result.Allowed {
					successCount++
				} else {
					rateLimitedCount++
				}
			case <-done:
				goto finished
			}
		}

	finished:
		actualDuration := time.Since(start)

		t.Logf("Expected requests: %d", expectedRequests)
		t.Logf("Successful requests: %d", successCount)
		t.Logf("Rate limited requests: %d", rateLimitedCount)
		t.Logf("Duration: %v", actualDuration)
		t.Logf("Average rate: %.2f req/sec", float64(successCount)/actualDuration.Seconds())

		// Проверяем что rate limiting работает корректно
		// При 10 req/sec и лимите 60 req/min все должны пройти
		assert.Equal(t, expectedRequests, successCount+rateLimitedCount,
			"Total requests should match expected")
	})
}

// TestPerformance_RateLimiting_HighConcurrency тестирует высокую конкурентность
func TestPerformance_RateLimiting_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("high concurrency with single key", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)
		limiter := setupRateLimiter(t)
		defer limiter.Stop()

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "High Concurrency Key",
			Description: "High concurrency test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 1000,
				RequestsPerDay:    10000,
			},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Запускаем много параллельных горутин
		var wg sync.WaitGroup
		goroutines := 100
		requestsPerGoroutine := 10
		totalRequests := goroutines * requestsPerGoroutine

		successCount := 0
		rateLimitedCount := 0
		var mu sync.Mutex

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < requestsPerGoroutine; j++ {
					result := limiter.CheckRateLimit(ctx, key.APIKey.ID, &key.APIKey, 0)

					mu.Lock()
					if result.Allowed {
						successCount++
					} else {
						rateLimitedCount++
					}
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Goroutines: %d", goroutines)
		t.Logf("Total requests: %d", totalRequests)
		t.Logf("Successful requests: %d", successCount)
		t.Logf("Rate limited requests: %d", rateLimitedCount)
		t.Logf("Duration: %v", duration)
		t.Logf("Requests per second: %.2f", float64(totalRequests)/duration.Seconds())

		// Проверяем что все запросы обработаны
		assert.Equal(t, totalRequests, successCount+rateLimitedCount,
			"All requests should be processed")

		// При высоком лимите большинство запросов должны пройти
		assert.Greater(t, successCount, totalRequests/2,
			"Most requests should succeed with high rate limit")
	})

	t.Run("concurrent key validation and rate limiting", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)
		limiter := setupRateLimiter(t)
		defer limiter.Stop()

		// Создаем ключ
		req := models.CreateAPIKeyRequest{
			Name:        "Concurrent Validation Key",
			Description: "Concurrent validation test",
			Models:      []string{"*"},
			Permissions: []string{"chat"},
			RateLimits: &models.RateLimits{
				RequestsPerMinute: 500,
				RequestsPerDay:    5000,
			},
		}
		key, err := manager.CreateAPIKey(ctx, req)
		require.NoError(t, err)

		// Одновременно валидируем ключ и проверяем rate limiting
		var wg sync.WaitGroup
		goroutines := 50
		iterationsPerGoroutine := 5

		validationErrors := 0
		rateLimitErrors := 0
		successCount := 0
		var mu sync.Mutex

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < iterationsPerGoroutine; j++ {
					// Валидируем ключ
					result, err := manager.ValidateAPIKey(ctx, key.PlainKey)
					if err != nil || !result.Valid {
						mu.Lock()
						validationErrors++
						mu.Unlock()
						continue
					}

					// Проверяем rate limiting
					limitResult := limiter.CheckRateLimit(ctx, key.APIKey.ID, &key.APIKey, 0)

					mu.Lock()
					if limitResult.Allowed {
						successCount++
					} else {
						rateLimitErrors++
					}
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		totalOperations := goroutines * iterationsPerGoroutine

		t.Logf("Total operations: %d", totalOperations)
		t.Logf("Successful: %d", successCount)
		t.Logf("Validation errors: %d", validationErrors)
		t.Logf("Rate limit errors: %d", rateLimitErrors)
		t.Logf("Duration: %v", duration)
		t.Logf("Operations per second: %.2f", float64(totalOperations)/duration.Seconds())

		// Не должно быть ошибок валидации
		assert.Equal(t, 0, validationErrors,
			"Should not have validation errors")

		// Все запросы должны быть обработаны
		assert.Equal(t, totalOperations, successCount+rateLimitErrors,
			"All operations should be processed")
	})
}

// TestPerformance_APIKeyOperations тестирует производительность операций с API ключами
func TestPerformance_APIKeyOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("batch key creation", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)

		keyCount := 50
		start := time.Now()

		for i := 0; i < keyCount; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        "Batch Key",
				Description: "Batch creation test",
				Models:      []string{"*"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		duration := time.Since(start)

		t.Logf("Created %d keys in %v", keyCount, duration)
		t.Logf("Average time per key: %v", duration/time.Duration(keyCount))
		t.Logf("Keys per second: %.2f", float64(keyCount)/duration.Seconds())

		// Проверяем что ключи создаются достаточно быстро
		avgPerKey := duration / time.Duration(keyCount)
		assert.Less(t, avgPerKey, 200*time.Millisecond,
			"Key creation should be reasonably fast")
	})

	t.Run("concurrent key listing", func(t *testing.T) {
		manager, ctx := setupPerformanceTestManager(t)

		// Создаем несколько ключей
		for i := 0; i < 10; i++ {
			req := models.CreateAPIKeyRequest{
				Name:        "List Test Key",
				Description: "List test",
				Models:      []string{"*"},
				Permissions: []string{"chat"},
			}
			_, err := manager.CreateAPIKey(ctx, req)
			require.NoError(t, err)
		}

		// Параллельно запрашиваем список ключей
		var wg sync.WaitGroup
		goroutines := 20
		errors := 0
		var mu sync.Mutex

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				listReq := models.ListAPIKeysRequest{
					Limit:  100,
					Offset: 0,
				}
				_, err := manager.ListAPIKeys(ctx, listReq)
				if err != nil {
					mu.Lock()
					errors++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Executed %d concurrent list operations in %v", goroutines, duration)
		t.Logf("Errors: %d", errors)

		assert.Equal(t, 0, errors, "Should not have errors during concurrent listing")
	})
}

// BenchmarkRateLimiter_CheckRateLimit бенчмарк для rate limiter
func BenchmarkRateLimiter_CheckRateLimit(b *testing.B) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			Enabled: true,
		},
		Server: config.ServerConfig{Host: "localhost", Port: 8080},
		Ollama: config.OllamaConfig{URL: "http://localhost:11434"},
	}
	cfg.Auth.RateLimiting.Enabled = true
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	limiter := ratelimit.NewLimiter(cfg, logger)
	defer limiter.Stop()

	keyInfo := &models.APIKeyPublic{
		ID:   "test-key-123",
		Name: "Benchmark Key",
		RateLimits: models.RateLimits{
			RequestsPerMinute: 1000,
			RequestsPerDay:    10000,
		},
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.CheckRateLimit(ctx, keyInfo.ID, keyInfo, 0)
	}
}

// BenchmarkAPIKeyValidation_Parallel бенчмарк для параллельной валидации
func BenchmarkAPIKeyValidation_Parallel(b *testing.B) {
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

