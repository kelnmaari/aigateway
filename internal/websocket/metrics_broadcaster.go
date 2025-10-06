package websocket

import (
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/metrics"
)

// MetricsBroadcaster периодически отправляет обновления метрик
type MetricsBroadcaster struct {
	storage     *metrics.MetricsStorage
	broadcaster *EventBroadcaster
	logger      *logrus.Logger
	interval    time.Duration
	stopChan    chan struct{}
}

// NewMetricsBroadcaster создает новый metrics broadcaster
func NewMetricsBroadcaster(
	storage *metrics.MetricsStorage,
	broadcaster *EventBroadcaster,
	logger *logrus.Logger,
	interval time.Duration,
) *MetricsBroadcaster {
	if logger == nil {
		logger = logrus.New()
	}

	if interval == 0 {
		interval = 5 * time.Second // Default: обновления каждые 5 секунд
	}

	return &MetricsBroadcaster{
		storage:     storage,
		broadcaster: broadcaster,
		logger:      logger,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

// Start запускает периодическую отправку метрик
func (mb *MetricsBroadcaster) Start() {
	mb.logger.WithField("interval", mb.interval).Info("Metrics broadcaster started")

	go mb.run()
}

// Stop останавливает broadcaster
func (mb *MetricsBroadcaster) Stop() {
	mb.logger.Info("Stopping metrics broadcaster")
	close(mb.stopChan)
}

// run основной цикл отправки метрик
func (mb *MetricsBroadcaster) run() {
	ticker := time.NewTicker(mb.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mb.broadcastMetrics()

		case <-mb.stopChan:
			mb.logger.Info("Metrics broadcaster stopped")
			return
		}
	}
}

// broadcastMetrics собирает и отправляет текущие метрики
func (mb *MetricsBroadcaster) broadcastMetrics() {
	// Получаем статистику для всех метрик
	allStats := mb.storage.GetAllStats()

	// Конвертируем в map[string]interface{} для события
	metricsData := make(map[string]interface{})

	for metricType, stats := range allStats {
		metricsData[string(metricType)] = map[string]interface{}{
			"count":   stats.Count,
			"sum":     stats.Sum,
			"min":     stats.Min,
			"max":     stats.Max,
			"avg":     stats.Avg,
			"median":  stats.Median,
			"p95":     stats.P95,
			"p99":     stats.P99,
			"std_dev": stats.StdDev,
		}
	}

	// Добавляем информацию о buffers
	bufferInfo := mb.storage.GetBufferInfo()
	buffersData := make(map[string]interface{})

	for metricType, info := range bufferInfo {
		buffersData[string(metricType)] = map[string]interface{}{
			"size":     info.Size,
			"capacity": info.Capacity,
			"is_full":  info.IsFull,
		}
	}

	metricsData["buffers"] = buffersData

	// Отправляем событие
	err := mb.broadcaster.BroadcastMetricsUpdate(metricsData)
	if err != nil {
		mb.logger.WithError(err).Error("Failed to broadcast metrics")
	}
}

// BroadcastImmediate немедленно отправляет текущие метрики
func (mb *MetricsBroadcaster) BroadcastImmediate() {
	mb.broadcastMetrics()
}
