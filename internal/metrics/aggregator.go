package metrics

import (
	"math"
	"sort"
	"sync"
)

// AggregatedStats содержит агрегированную статистику
type AggregatedStats struct {
	Count  int64   `json:"count"`
	Sum    float64 `json:"sum"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"` // P50
	P95    float64 `json:"p95"`
	P99    float64 `json:"p99"`
	StdDev float64 `json:"std_dev"` // Стандартное отклонение
}

// Aggregator вычисляет статистику из DataPoint
type Aggregator struct {
	mu sync.RWMutex
}

// NewAggregator создает новый aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{}
}

// Aggregate вычисляет статистику для набора данных
func (a *Aggregator) Aggregate(data []DataPoint) AggregatedStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(data) == 0 {
		return AggregatedStats{}
	}

	// Извлекаем значения
	values := make([]float64, len(data))
	for i, dp := range data {
		values[i] = dp.Value
	}

	return a.calculateStats(values)
}

// AggregateValues вычисляет статистику для массива значений
func (a *Aggregator) AggregateValues(values []float64) AggregatedStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.calculateStats(values)
}

// calculateStats выполняет основные расчеты
func (a *Aggregator) calculateStats(values []float64) AggregatedStats {
	if len(values) == 0 {
		return AggregatedStats{}
	}

	stats := AggregatedStats{
		Count: int64(len(values)),
		Min:   math.MaxFloat64,
		Max:   -math.MaxFloat64,
	}

	// Первый проход: sum, min, max
	for _, v := range values {
		stats.Sum += v
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
	}

	// Average
	stats.Avg = stats.Sum / float64(len(values))

	// Для percentiles нужно отсортировать
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// Median (P50)
	stats.Median = percentile(sorted, 0.50)

	// P95
	stats.P95 = percentile(sorted, 0.95)

	// P99
	stats.P99 = percentile(sorted, 0.99)

	// Standard Deviation
	stats.StdDev = a.calculateStdDev(values, stats.Avg)

	return stats
}

// percentile вычисляет процентиль для отсортированного массива
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}

	// Линейная интерполяция
	rank := p * float64(len(sorted)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	if lowerIndex == upperIndex {
		return sorted[lowerIndex]
	}

	// Интерполяция между двумя значениями
	weight := rank - float64(lowerIndex)
	return sorted[lowerIndex]*(1-weight) + sorted[upperIndex]*weight
}

// calculateStdDev вычисляет стандартное отклонение
func (a *Aggregator) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	var sumSquaredDiff float64
	for _, v := range values {
		diff := v - mean
		sumSquaredDiff += diff * diff
	}

	variance := sumSquaredDiff / float64(len(values))
	return math.Sqrt(variance)
}

// GroupByInterval группирует DataPoint по временным интервалам
func (a *Aggregator) GroupByInterval(data []DataPoint, intervalSeconds int) map[int64][]DataPoint {
	a.mu.RLock()
	defer a.mu.RUnlock()

	grouped := make(map[int64][]DataPoint)

	for _, dp := range data {
		// Округляем timestamp до ближайшего интервала
		bucket := (dp.Timestamp.Unix() / int64(intervalSeconds)) * int64(intervalSeconds)
		grouped[bucket] = append(grouped[bucket], dp)
	}

	return grouped
}

// AggregateByInterval вычисляет статистику для каждого временного интервала
func (a *Aggregator) AggregateByInterval(data []DataPoint, intervalSeconds int) []TimeSeriesPoint {
	grouped := a.GroupByInterval(data, intervalSeconds)

	// Сортируем ключи (timestamps)
	timestamps := make([]int64, 0, len(grouped))
	for ts := range grouped {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})

	// Вычисляем статистику для каждого интервала
	result := make([]TimeSeriesPoint, 0, len(timestamps))
	for _, ts := range timestamps {
		points := grouped[ts]
		stats := a.Aggregate(points)

		result = append(result, TimeSeriesPoint{
			Timestamp: ts,
			Stats:     stats,
			Count:     len(points),
		})
	}

	return result
}

// TimeSeriesPoint представляет точку временного ряда
type TimeSeriesPoint struct {
	Timestamp int64           `json:"timestamp"`
	Stats     AggregatedStats `json:"stats"`
	Count     int             `json:"count"`
}
