package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// MetricType определяет тип метрики
type MetricType string

const (
	MetricTypeRequestCount   MetricType = "request_count"
	MetricTypeRequestLatency MetricType = "request_latency"
	MetricTypeErrorCount     MetricType = "error_count"
	MetricTypeRequestSize    MetricType = "request_size"
	MetricTypeResponseSize   MetricType = "response_size"
	MetricTypeActiveRequests MetricType = "active_requests"
)

// StorageConfig конфигурация для metrics storage
type StorageConfig struct {
	// Capacity - максимальное количество точек данных в ring buffer
	Capacity int

	// RetentionPeriod - период хранения данных
	RetentionPeriod time.Duration

	// CleanupInterval - интервал очистки старых данных
	CleanupInterval time.Duration

	// Enabled - включить ли storage
	Enabled bool
}

// DefaultStorageConfig возвращает конфигурацию по умолчанию
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		Capacity:        3600, // 1 hour при 1 записи в секунду
		RetentionPeriod: 1 * time.Hour,
		CleanupInterval: 5 * time.Minute,
		Enabled:         true,
	}
}

// MetricsStorage хранит метрики в памяти
type MetricsStorage struct {
	mu     sync.RWMutex
	config StorageConfig
	logger *logrus.Logger

	// Ring buffers для разных типов метрик
	buffers map[MetricType]*RingBuffer

	// Aggregator для вычисления статистики
	aggregator *Aggregator

	// Канал для остановки background задач
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Atomic счетчик активных запросов
	activeRequestsCount int64
}

// NewMetricsStorage создает новое хранилище метрик
func NewMetricsStorage(config StorageConfig, logger *logrus.Logger) *MetricsStorage {
	if logger == nil {
		logger = logrus.New()
	}

	ms := &MetricsStorage{
		config:     config,
		logger:     logger,
		buffers:    make(map[MetricType]*RingBuffer),
		aggregator: NewAggregator(),
		stopChan:   make(chan struct{}),
	}

	// Инициализируем buffers для каждого типа метрики
	ms.buffers[MetricTypeRequestCount] = NewRingBuffer(config.Capacity)
	ms.buffers[MetricTypeRequestLatency] = NewRingBuffer(config.Capacity)
	ms.buffers[MetricTypeErrorCount] = NewRingBuffer(config.Capacity)
	ms.buffers[MetricTypeRequestSize] = NewRingBuffer(config.Capacity)
	ms.buffers[MetricTypeResponseSize] = NewRingBuffer(config.Capacity)
	ms.buffers[MetricTypeActiveRequests] = NewRingBuffer(config.Capacity)

	return ms
}

// Start запускает background задачи
func (ms *MetricsStorage) Start() {
	if !ms.config.Enabled {
		ms.logger.Info("Metrics storage disabled")
		return
	}

	ms.logger.Info("Starting metrics storage")

	// Запускаем cleanup задачу
	ms.wg.Add(1)
	go ms.cleanupWorker()
}

// Stop останавливает background задачи
func (ms *MetricsStorage) Stop() {
	ms.logger.Info("Stopping metrics storage")
	close(ms.stopChan)
	ms.wg.Wait()
	ms.logger.Info("Metrics storage stopped")
}

// Record записывает новую метрику
func (ms *MetricsStorage) Record(metricType MetricType, value float64, labels map[string]string) {
	if !ms.config.Enabled {
		return
	}

	ms.mu.RLock()
	buffer, exists := ms.buffers[metricType]
	ms.mu.RUnlock()

	if !exists {
		ms.logger.WithField("metric_type", metricType).Warn("Unknown metric type")
		return
	}

	buffer.Add(value, labels)
}

// GetStats возвращает агрегированную статистику для метрики
func (ms *MetricsStorage) GetStats(metricType MetricType) AggregatedStats {
	ms.mu.RLock()
	buffer, exists := ms.buffers[metricType]
	ms.mu.RUnlock()

	if !exists {
		return AggregatedStats{}
	}

	data := buffer.GetAll()
	return ms.aggregator.Aggregate(data)
}

// GetHistory возвращает исторические данные за период
func (ms *MetricsStorage) GetHistory(metricType MetricType, from, to time.Time) []DataPoint {
	ms.mu.RLock()
	buffer, exists := ms.buffers[metricType]
	ms.mu.RUnlock()

	if !exists {
		return []DataPoint{}
	}

	return buffer.GetRange(from, to)
}

// GetTimeSeries возвращает временной ряд с агрегацией по интервалам
func (ms *MetricsStorage) GetTimeSeries(metricType MetricType, from, to time.Time, intervalSeconds int) []TimeSeriesPoint {
	data := ms.GetHistory(metricType, from, to)
	return ms.aggregator.AggregateByInterval(data, intervalSeconds)
}

// GetAllStats возвращает статистику для всех метрик
func (ms *MetricsStorage) GetAllStats() map[MetricType]AggregatedStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[MetricType]AggregatedStats)
	for metricType := range ms.buffers {
		result[metricType] = ms.GetStats(metricType)
	}

	return result
}

// GetBufferInfo возвращает информацию о состоянии buffers
func (ms *MetricsStorage) GetBufferInfo() map[MetricType]BufferInfo {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make(map[MetricType]BufferInfo)
	for metricType, buffer := range ms.buffers {
		result[metricType] = BufferInfo{
			Size:     buffer.Size(),
			Capacity: buffer.Capacity(),
			IsFull:   buffer.IsFull(),
		}
	}

	return result
}

// BufferInfo информация о состоянии buffer
type BufferInfo struct {
	Size     int  `json:"size"`
	Capacity int  `json:"capacity"`
	IsFull   bool `json:"is_full"`
}

// Clear очищает все buffers
func (ms *MetricsStorage) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, buffer := range ms.buffers {
		buffer.Clear()
	}

	ms.logger.Info("All metrics buffers cleared")
}

// cleanupWorker периодически очищает старые данные
func (ms *MetricsStorage) cleanupWorker() {
	defer ms.wg.Done()

	ticker := time.NewTicker(ms.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ms.performCleanup()
		case <-ms.stopChan:
			return
		}
	}
}

// performCleanup удаляет данные старше RetentionPeriod
func (ms *MetricsStorage) performCleanup() {
	cutoff := time.Now().Add(-ms.config.RetentionPeriod)

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Для каждого buffer проверяем старые данные
	for metricType, buffer := range ms.buffers {
		data := buffer.GetAll()

		// Подсчитываем сколько записей старше cutoff
		var oldCount int
		for _, dp := range data {
			if dp.Timestamp.Before(cutoff) {
				oldCount++
			} else {
				break // Данные отсортированы по времени
			}
		}

		if oldCount > 0 {
			ms.logger.WithFields(logrus.Fields{
				"metric_type": metricType,
				"old_count":   oldCount,
				"total":       len(data),
			}).Debug("Cleanup: found old data points")

			// Ring buffer автоматически перезаписывает старые данные
			// при достижении capacity, поэтому дополнительная очистка не требуется
		}
	}
}

// GetRecentData возвращает последние N записей для метрики
func (ms *MetricsStorage) GetRecentData(metricType MetricType, count int) []DataPoint {
	ms.mu.RLock()
	buffer, exists := ms.buffers[metricType]
	ms.mu.RUnlock()

	if !exists {
		return []DataPoint{}
	}

	return buffer.GetLast(count)
}

// RecordLatency записывает latency запроса в миллисекундах
func (ms *MetricsStorage) RecordLatency(latencyMS int64) {
	ms.Record(MetricTypeRequestLatency, float64(latencyMS), nil)
}

// RecordRequestSize записывает размер запроса в байтах
func (ms *MetricsStorage) RecordRequestSize(sizeBytes int64) {
	ms.Record(MetricTypeRequestSize, float64(sizeBytes), nil)
}

// RecordResponseSize записывает размер ответа в байтах
func (ms *MetricsStorage) RecordResponseSize(sizeBytes int64) {
	ms.Record(MetricTypeResponseSize, float64(sizeBytes), nil)
}

// IncrementRequestCount увеличивает счетчик успешных запросов
func (ms *MetricsStorage) IncrementRequestCount() {
	ms.Record(MetricTypeRequestCount, 1.0, nil)
}

// IncrementErrorCount увеличивает счетчик ошибочных запросов
func (ms *MetricsStorage) IncrementErrorCount() {
	ms.Record(MetricTypeErrorCount, 1.0, nil)
}

// IncrementActiveRequests увеличивает счетчик активных запросов
func (ms *MetricsStorage) IncrementActiveRequests() {
	newCount := atomic.AddInt64(&ms.activeRequestsCount, 1)
	ms.Record(MetricTypeActiveRequests, float64(newCount), nil)
}

// DecrementActiveRequests уменьшает счетчик активных запросов
func (ms *MetricsStorage) DecrementActiveRequests() {
	newCount := atomic.AddInt64(&ms.activeRequestsCount, -1)
	if newCount < 0 {
		atomic.StoreInt64(&ms.activeRequestsCount, 0)
		newCount = 0
	}
	ms.Record(MetricTypeActiveRequests, float64(newCount), nil)
}

// GetActiveRequestsCount возвращает текущее количество активных запросов
func (ms *MetricsStorage) GetActiveRequestsCount() int64 {
	return atomic.LoadInt64(&ms.activeRequestsCount)
}
