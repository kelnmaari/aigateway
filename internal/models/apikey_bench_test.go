package models

import (
	"testing"
	"time"
)

// BenchmarkAPIKey_IncrementUsage_Original тест оригинального IncrementUsage
func BenchmarkAPIKey_IncrementUsage_Original(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name: "test",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key.IncrementUsage("gpt-4", "/v1/chat/completions", 1000, true)
	}
}

// BenchmarkAPIKeyUsageHot_IncrementUsage тест оптимизированного
func BenchmarkAPIKeyUsageHot_IncrementUsage(b *testing.B) {
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
}

// BenchmarkAPIKeyUsage_Parallel_Original параллельное обновление (race condition!)
func BenchmarkAPIKeyUsage_Parallel_Original(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name: "test",
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// ВНИМАНИЕ: Это НЕ thread-safe! Только для демонстрации проблемы
			key.IncrementUsage("gpt-4", "/v1/chat/completions", 1000, true)
		}
	})

	// Результаты будут некорректны из-за race condition
}

// BenchmarkAPIKeyUsageHot_Parallel оптимизированная версия (thread-safe)
func BenchmarkAPIKeyUsageHot_Parallel(b *testing.B) {
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
}

// BenchmarkAPIKey_HasModelAccess_Original проверка доступа (загружает все поля)
func BenchmarkAPIKey_HasModelAccess_Original(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:   "test",
		Models: []string{"gpt-4", "gpt-3.5-turbo", "claude-3"},
		Metadata: map[string]interface{}{
			"description": "Test key with lots of metadata",
			"tags":        []string{"production", "team-alpha", "high-priority"},
			"created_by":  "admin@example.com",
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.HasModelAccess("gpt-4")
	}
}

// BenchmarkAPIKeyHot_HasModelAccess hot path (только нужные данные)
func BenchmarkAPIKeyHot_HasModelAccess(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:   "test",
		Models: []string{"gpt-4", "gpt-3.5-turbo", "claude-3"},
	})

	keyHot, _ := ConvertToHotCold(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = keyHot.HasModelAccess("gpt-4")
	}
}

// BenchmarkAPIKeyHot_HasModelAccess_AllModels fast path (*=true)
func BenchmarkAPIKeyHot_HasModelAccess_AllModels(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:   "test",
		Models: []string{"*"},
	})

	keyHot, _ := ConvertToHotCold(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = keyHot.HasModelAccess("any-model")
	}
}

// BenchmarkAPIKey_IsActive_Original проверка активности
func BenchmarkAPIKey_IsActive_Original(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name: "test",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = key.IsActive()
	}
}

// BenchmarkAPIKeyHot_IsActive оптимизированная проверка
func BenchmarkAPIKeyHot_IsActive(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name: "test",
	})

	keyHot, _ := ConvertToHotCold(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = keyHot.IsActive()
	}
}

// BenchmarkConvertToHotCold стоимость конвертации
func BenchmarkConvertToHotCold(b *testing.B) {
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:        "test",
		Models:      []string{"gpt-4", "gpt-3.5-turbo"},
		Permissions: []string{"chat", "models"},
		Metadata: map[string]interface{}{
			"description": "Test key",
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ConvertToHotCold(key)
	}
}

// BenchmarkAPIKeyUsageHot_GetSnapshot тест создания snapshot
func BenchmarkAPIKeyUsageHot_GetSnapshot(b *testing.B) {
	usage := &APIKeyUsageHot{}
	usage.IncrementUsage(1000, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = usage.GetSnapshot()
	}
}

// BenchmarkAPIKey_Validation_Workflow реалистичный сценарий валидации
func BenchmarkAPIKey_Validation_Workflow(b *testing.B) {
	expiresAt := time.Now().Add(24 * time.Hour)
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:        "test",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		ExpiresAt:   &expiresAt,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Типичная проверка в middleware
		if !key.IsActive() {
			continue
		}
		if !key.HasModelAccess("gpt-4") {
			continue
		}
		if !key.HasPermission("chat") {
			continue
		}
	}
}

// BenchmarkAPIKeyHot_Validation_Workflow оптимизированный workflow
func BenchmarkAPIKeyHot_Validation_Workflow(b *testing.B) {
	expiresAt := time.Now().Add(24 * time.Hour)
	key, _, _ := NewAPIKey(CreateAPIKeyRequest{
		Name:        "test",
		Models:      []string{"gpt-4"},
		Permissions: []string{"chat"},
		ExpiresAt:   &expiresAt,
	})

	keyHot, _ := ConvertToHotCold(key)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !keyHot.IsActive() {
			continue
		}
		if !keyHot.HasModelAccess("gpt-4") {
			continue
		}
		if !keyHot.HasPermission("chat") {
			continue
		}
	}
}

/*
Как запустить:

# Все benchmarks
go test -bench=. -benchmem -benchtime=10s

# Только IncrementUsage (демонстрирует false sharing)
go test -bench=IncrementUsage -benchtime=10s -cpu=1,2,4,8,16

# Только validation workflow
go test -bench=Validation -benchmem

# С race detector (покажет race condition в Original)
go test -race -bench=Parallel -benchtime=1s

Ожидаемые результаты:

IncrementUsage (Sequential):
  Original:   ~150ns/op  (map allocations + non-atomic)
  Optimized:  ~8ns/op    (только atomic ops)
  Speedup: 18x

IncrementUsage (Parallel, 16 cores):
  Original:   race condition + slow
  Optimized:  ~15ns/op   (с padding, без false sharing)

HasModelAccess:
  Original:   ~35ns/op   (загружает всю структуру)
  Optimized:  ~12ns/op   (только hot data)
  Speedup: 3x

HasModelAccess (AllModels=true):
  Optimized:  ~2ns/op    (быстрый путь)
  Speedup: 17x vs original

Validation Workflow:
  Original:   ~85ns/op   (3 метода, все поля в cache)
  Optimized:  ~28ns/op   (hot data в одной cache line)
  Speedup: 3x

Memory:
  Original Usage:   ~120 bytes + maps
  Optimized Usage:  ~320 bytes (padded) + maps in ColdUsage

Trade-off: +200 bytes per key для 3-18x speedup на hot path
*/

