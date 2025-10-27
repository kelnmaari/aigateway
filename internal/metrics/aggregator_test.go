package metrics

import (
	"math"
	"testing"
	"time"
)

func TestAggregator_CalculateStats(t *testing.T) {
	agg := NewAggregator()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}
	stats := agg.AggregateValues(values)

	// Проверяем основные метрики
	if stats.Count != 5 {
		t.Errorf("Expected count 5, got %d", stats.Count)
	}

	if stats.Sum != 15.0 {
		t.Errorf("Expected sum 15.0, got %f", stats.Sum)
	}

	if stats.Avg != 3.0 {
		t.Errorf("Expected avg 3.0, got %f", stats.Avg)
	}

	if stats.Min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", stats.Min)
	}

	if stats.Max != 5.0 {
		t.Errorf("Expected max 5.0, got %f", stats.Max)
	}

	if stats.Median != 3.0 {
		t.Errorf("Expected median 3.0, got %f", stats.Median)
	}
}

func TestAggregator_Percentiles(t *testing.T) {
	agg := NewAggregator()

	// Создаем набор данных от 1 до 100
	values := make([]float64, 100)
	for i := 0; i < 100; i++ {
		values[i] = float64(i + 1)
	}

	stats := agg.AggregateValues(values)

	// P50 (медиана) должна быть около 50.5
	if math.Abs(stats.Median-50.5) > 0.5 {
		t.Errorf("Expected median around 50.5, got %f", stats.Median)
	}

	// P95 должна быть около 95
	if math.Abs(stats.P95-95.0) > 1.0 {
		t.Errorf("Expected P95 around 95, got %f", stats.P95)
	}

	// P99 должна быть около 99
	if math.Abs(stats.P99-99.0) > 1.0 {
		t.Errorf("Expected P99 around 99, got %f", stats.P99)
	}
}

func TestAggregator_EmptyData(t *testing.T) {
	agg := NewAggregator()

	stats := agg.AggregateValues([]float64{})

	if stats.Count != 0 {
		t.Errorf("Expected count 0 for empty data, got %d", stats.Count)
	}

	if stats.Sum != 0 {
		t.Errorf("Expected sum 0 for empty data, got %f", stats.Sum)
	}
}

func TestAggregator_SingleValue(t *testing.T) {
	agg := NewAggregator()

	stats := agg.AggregateValues([]float64{42.0})

	if stats.Count != 1 {
		t.Errorf("Expected count 1, got %d", stats.Count)
	}

	if stats.Min != 42.0 || stats.Max != 42.0 || stats.Avg != 42.0 || stats.Median != 42.0 {
		t.Errorf("Expected all stats to be 42.0 for single value")
	}

	if stats.StdDev != 0 {
		t.Errorf("Expected std dev 0 for single value, got %f", stats.StdDev)
	}
}

func TestAggregator_StdDev(t *testing.T) {
	agg := NewAggregator()

	// Данные с известным стандартным отклонением
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	stats := agg.AggregateValues(values)

	// Среднее: 5.0
	// Дисперсия: ((2-5)² + (4-5)² + (4-5)² + (4-5)² + (5-5)² + (5-5)² + (7-5)² + (9-5)²) / 8
	//          = (9 + 1 + 1 + 1 + 0 + 0 + 4 + 16) / 8 = 32 / 8 = 4
	// Стандартное отклонение: √4 = 2.0

	if stats.Avg != 5.0 {
		t.Errorf("Expected avg 5.0, got %f", stats.Avg)
	}

	if math.Abs(stats.StdDev-2.0) > 0.01 {
		t.Errorf("Expected std dev 2.0, got %f", stats.StdDev)
	}
}

func TestAggregator_GroupByInterval(t *testing.T) {
	agg := NewAggregator()

	// Создаем данные с разными timestamps
	now := time.Now().Truncate(time.Minute)
	data := []DataPoint{
		{Timestamp: now, Value: 1.0},
		{Timestamp: now.Add(30 * time.Second), Value: 2.0},
		{Timestamp: now.Add(1 * time.Minute), Value: 3.0},
		{Timestamp: now.Add(90 * time.Second), Value: 4.0},
		{Timestamp: now.Add(2 * time.Minute), Value: 5.0},
	}

	// Группируем по 60 секунд
	grouped := agg.GroupByInterval(data, 60)

	// Должно быть 3 группы (0-60s, 60-120s, 120-180s)
	if len(grouped) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(grouped))
	}

	// Проверяем что в каждой группе правильное количество элементов
	firstBucket := now.Unix()
	if len(grouped[firstBucket]) != 2 {
		t.Errorf("Expected 2 data points in first bucket, got %d", len(grouped[firstBucket]))
	}

	secondBucket := now.Add(1 * time.Minute).Unix()
	if len(grouped[secondBucket]) != 2 {
		t.Errorf("Expected 2 data points in second bucket, got %d", len(grouped[secondBucket]))
	}

	thirdBucket := now.Add(2 * time.Minute).Unix()
	if len(grouped[thirdBucket]) != 1 {
		t.Errorf("Expected 1 data point in third bucket, got %d", len(grouped[thirdBucket]))
	}
}

func TestAggregator_AggregateByInterval(t *testing.T) {
	agg := NewAggregator()

	// Создаем данные
	now := time.Now().Truncate(time.Minute)
	data := []DataPoint{
		{Timestamp: now, Value: 10.0},
		{Timestamp: now.Add(30 * time.Second), Value: 20.0},
		{Timestamp: now.Add(1 * time.Minute), Value: 30.0},
	}

	// Агрегируем по 60 секунд
	timeSeries := agg.AggregateByInterval(data, 60)

	if len(timeSeries) != 2 {
		t.Errorf("Expected 2 time series points, got %d", len(timeSeries))
	}

	// Проверяем первый интервал
	if timeSeries[0].Count != 2 {
		t.Errorf("Expected 2 data points in first interval, got %d", timeSeries[0].Count)
	}

	if timeSeries[0].Stats.Avg != 15.0 {
		t.Errorf("Expected avg 15.0 in first interval, got %f", timeSeries[0].Stats.Avg)
	}

	// Проверяем второй интервал
	if timeSeries[1].Count != 1 {
		t.Errorf("Expected 1 data point in second interval, got %d", timeSeries[1].Count)
	}

	if timeSeries[1].Stats.Avg != 30.0 {
		t.Errorf("Expected avg 30.0 in second interval, got %f", timeSeries[1].Stats.Avg)
	}
}

func TestAggregator_AggregateDataPoints(t *testing.T) {
	agg := NewAggregator()

	data := []DataPoint{
		{Timestamp: time.Now(), Value: 1.0},
		{Timestamp: time.Now(), Value: 2.0},
		{Timestamp: time.Now(), Value: 3.0},
	}

	stats := agg.Aggregate(data)

	if stats.Count != 3 {
		t.Errorf("Expected count 3, got %d", stats.Count)
	}

	if stats.Sum != 6.0 {
		t.Errorf("Expected sum 6.0, got %f", stats.Sum)
	}

	if stats.Avg != 2.0 {
		t.Errorf("Expected avg 2.0, got %f", stats.Avg)
	}
}

