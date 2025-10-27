package models

import (
	"sync"
	"testing"
)

// TestAPIKeyUsage_RaceCondition демонстрирует race condition в оригинальной реализации
//
// Запустить с: go test -race -run=TestAPIKeyUsage_RaceCondition
//
// Ожидаемый результат: WARNING: DATA RACE
func TestAPIKeyUsage_RaceCondition(t *testing.T) {
	t.Skip("Skipped by default: demonstrates race condition in original code")

	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name: "test",
	})

	// Запускаем 100 goroutines которые конкурентно обновляют счетчики
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				// Эта функция НЕ thread-safe!
				key.IncrementUsage("gpt-4", "/v1/chat/completions", 1000, true)
			}
		}()
	}

	wg.Wait()

	// Результаты будут некорректны из-за race condition
	t.Logf("Total requests: %d (should be 100000, but will be less due to race)",
		key.Usage.TotalRequests)
}

// TestAPIKeyUsageHot_NoRaceCondition демонстрирует отсутствие race в оптимизированной версии
//
// Запустить с: go test -race -run=TestAPIKeyUsageHot_NoRaceCondition
//
// Ожидаемый результат: PASS (no race detected)
func TestAPIKeyUsageHot_NoRaceCondition(t *testing.T) {
	usage := &APIKeyUsageHot{
		ColdUsage: &APIKeyUsageCold{
			ModelUsage:    make(map[string]int64),
			EndpointUsage: make(map[string]int64),
			DailyUsage:    make(map[string]DayUsage),
		},
	}

	// Запускаем 100 goroutines которые конкурентно обновляют счетчики
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				// Thread-safe благодаря atomic operations
				usage.IncrementUsage(1000, true)
			}
		}()
	}

	wg.Wait()

	snapshot := usage.GetSnapshot()

	// Проверяем корректность результатов
	expected := int64(100 * 1000)
	if snapshot.TotalRequests != expected {
		t.Errorf("TotalRequests = %d, want %d", snapshot.TotalRequests, expected)
	}

	if snapshot.SuccessfulRequests != expected {
		t.Errorf("SuccessfulRequests = %d, want %d", snapshot.SuccessfulRequests, expected)
	}

	expectedTokens := expected * 1000
	if snapshot.TotalTokens != expectedTokens {
		t.Errorf("TotalTokens = %d, want %d", snapshot.TotalTokens, expectedTokens)
	}

	t.Logf("✅ All counters correct: %d requests, %d tokens",
		snapshot.TotalRequests, snapshot.TotalTokens)
}

// TestAPIKeyUsage_Correctness проверяет корректность счетчиков при конкурентном доступе
func TestAPIKeyUsage_Correctness(t *testing.T) {
	tests := []struct {
		name       string
		goroutines int
		iterations int
		success    bool
	}{
		{"Low_Concurrency", 10, 100, true},
		{"Medium_Concurrency", 50, 1000, true},
		{"High_Concurrency", 100, 10000, true},
		{"With_Failures", 50, 1000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &APIKeyUsageHot{
				ColdUsage: &APIKeyUsageCold{
					ModelUsage:    make(map[string]int64),
					EndpointUsage: make(map[string]int64),
					DailyUsage:    make(map[string]DayUsage),
				},
			}

			var wg sync.WaitGroup
			for i := 0; i < tt.goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < tt.iterations; j++ {
						usage.IncrementUsage(1000, tt.success)
					}
				}()
			}

			wg.Wait()

			snapshot := usage.GetSnapshot()
			expected := int64(tt.goroutines * tt.iterations)

			if snapshot.TotalRequests != expected {
				t.Errorf("TotalRequests = %d, want %d", snapshot.TotalRequests, expected)
			}

			if tt.success {
				if snapshot.SuccessfulRequests != expected {
					t.Errorf("SuccessfulRequests = %d, want %d",
						snapshot.SuccessfulRequests, expected)
				}
				if snapshot.FailedRequests != 0 {
					t.Errorf("FailedRequests = %d, want 0", snapshot.FailedRequests)
				}
			} else {
				if snapshot.FailedRequests != expected {
					t.Errorf("FailedRequests = %d, want %d",
						snapshot.FailedRequests, expected)
				}
				if snapshot.SuccessfulRequests != 0 {
					t.Errorf("SuccessfulRequests = %d, want 0",
						snapshot.SuccessfulRequests)
				}
			}

			expectedTokens := expected * 1000
			if snapshot.TotalTokens != expectedTokens {
				t.Errorf("TotalTokens = %d, want %d", snapshot.TotalTokens, expectedTokens)
			}
		})
	}
}

// BenchmarkAPIKeyUsage_RaceVsNoRace сравнивает производительность с/без race condition
func BenchmarkAPIKeyUsage_RaceVsNoRace(b *testing.B) {
	b.Run("Original_Sequential", func(b *testing.B) {
		key, _, _ := NewAPIKey(CreateAPIKeyRequest{Name: "test"})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key.IncrementUsage("gpt-4", "/v1/chat/completions", 1000, true)
		}
	})

	b.Run("Optimized_Sequential", func(b *testing.B) {
		usage := &APIKeyUsageHot{
			ColdUsage: &APIKeyUsageCold{
				ModelUsage:    make(map[string]int64),
				EndpointUsage: make(map[string]int64),
				DailyUsage:    make(map[string]DayUsage),
			},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			usage.IncrementUsage(1000, true)
		}
	})

	b.Run("Optimized_Parallel", func(b *testing.B) {
		usage := &APIKeyUsageHot{
			ColdUsage: &APIKeyUsageCold{
				ModelUsage:    make(map[string]int64),
				EndpointUsage: make(map[string]int64),
				DailyUsage:    make(map[string]DayUsage),
			},
		}

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				usage.IncrementUsage(1000, true)
			}
		})
	})
}

/*
Как запустить:

# Демонстрация race condition (покажет WARNING)
go test -race -run=TestAPIKeyUsage_RaceCondition -v

# Проверка отсутствия race в оптимизированной версии
go test -race -run=TestAPIKeyUsageHot_NoRaceCondition -v

# Проверка корректности счетчиков
go test -run=TestAPIKeyUsage_Correctness -v

# Benchmark сравнение
go test -bench=BenchmarkAPIKeyUsage_RaceVsNoRace -benchtime=10s

Ожидаемые результаты:

TestAPIKeyUsage_RaceCondition:
  ⚠️  WARNING: DATA RACE
  ❌ TotalRequests < 100000 (lost updates)

TestAPIKeyUsageHot_NoRaceCondition:
  ✅ PASS (no race detected)
  ✅ TotalRequests = 100000 (correct)

Benchmark:
  Original_Sequential:  ~150ns/op (maps overhead)
  Optimized_Sequential: ~8ns/op   (only atomic ops)
  Optimized_Parallel:   ~15ns/op  (with padding, no false sharing)
*/

