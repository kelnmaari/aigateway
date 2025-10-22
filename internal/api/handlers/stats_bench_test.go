package handlers

import (
	"sync"
	"testing"
)

// BenchmarkStatsOriginal_Sequential тест оригинальной версии (без конкурентности)
func BenchmarkStatsOriginal_Sequential(b *testing.B) {
	stats := &Stats{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.IncrementTotalRequests()
	}
}

// BenchmarkStatsOptimized_Sequential тест оптимизированной версии (без конкурентности)
func BenchmarkStatsOptimized_Sequential(b *testing.B) {
	stats := NewStatsOptimized()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stats.IncrementTotalRequests()
	}
}

// BenchmarkStatsOriginal_Parallel тест false sharing (высокая конкурентность)
func BenchmarkStatsOriginal_Parallel(b *testing.B) {
	stats := &Stats{}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Эмулируем реальный HTTP request workflow
			stats.IncrementTotalRequests()
			stats.IncrementActiveRequests()
			stats.IncrementSuccessRequests()
			stats.DecrementActiveRequests()
		}
	})
}

// BenchmarkStatsOptimized_Parallel тест с padding (без false sharing)
func BenchmarkStatsOptimized_Parallel(b *testing.B) {
	stats := NewStatsOptimized()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			stats.IncrementTotalRequests()
			stats.IncrementActiveRequests()
			stats.IncrementSuccessRequests()
			stats.DecrementActiveRequests()
		}
	})
}

// BenchmarkStatsOriginal_Contention высокая contention на 4 счетчиках
func BenchmarkStatsOriginal_Contention(b *testing.B) {
	stats := &Stats{}

	// 4 goroutines обновляют 4 разных счетчика одновременно
	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		for pb.Next() {
			wg.Add(4)

			go func() {
				defer wg.Done()
				stats.IncrementTotalRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementActiveRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementSuccessRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementErrorRequests()
			}()

			wg.Wait()
		}
	})
}

// BenchmarkStatsOptimized_Contention с padding - нет false sharing
func BenchmarkStatsOptimized_Contention(b *testing.B) {
	stats := NewStatsOptimized()

	b.RunParallel(func(pb *testing.PB) {
		var wg sync.WaitGroup
		for pb.Next() {
			wg.Add(4)

			go func() {
				defer wg.Done()
				stats.IncrementTotalRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementActiveRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementSuccessRequests()
			}()

			go func() {
				defer wg.Done()
				stats.IncrementErrorRequests()
			}()

			wg.Wait()
		}
	})
}

// BenchmarkSnapshot_Original тест чтения снимка статистики
func BenchmarkSnapshot_Original(b *testing.B) {
	stats := &Stats{}

	// Подготовим данные
	stats.IncrementTotalRequests()
	stats.IncrementSuccessRequests()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stats.Snapshot()
	}
}

// BenchmarkSnapshot_Optimized тест чтения снимка
func BenchmarkSnapshot_Optimized(b *testing.B) {
	stats := NewStatsOptimized()

	stats.IncrementTotalRequests()
	stats.IncrementSuccessRequests()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stats.Snapshot()
	}
}

/*
Как запустить benchmarks:

# Последовательный тест (baseline)
go test -bench=Sequential -benchtime=10s

# Параллельный тест (демонстрирует false sharing)
go test -bench=Parallel -benchtime=10s -cpu=1,2,4,8,16

# Высокая contention (максимальная нагрузка)
go test -bench=Contention -benchtime=10s -cpu=16

# Все benchmarks с детальной статистикой
go test -bench=. -benchmem -benchtime=10s -cpu=1,2,4,8,16

Ожидаемые результаты (16 CPU cores):

Sequential (1 goroutine):
  Original:   ~2ns/op
  Optimized:  ~2ns/op
  Вывод: Без конкурентности overhead от padding незначительный

Parallel (RunParallel):
  Original:   ~45ns/op  (false sharing!)
  Optimized:  ~7ns/op
  Speedup: 6.4x

Contention (4 goroutines на 4 счетчика):
  Original:   ~120ns/op (worst case false sharing)
  Optimized:  ~12ns/op
  Speedup: 10x

Snapshot:
  Original:   ~80ns/op
  Optimized:  ~85ns/op
  Вывод: Незначительное замедление из-за padding

Memory:
  Original:   64 bytes/struct
  Optimized:  320 bytes/struct
  Overhead: 5x memory (acceptable для global singleton)
*/
