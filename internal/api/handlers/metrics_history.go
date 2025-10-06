package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/metrics"
)

// MetricsHistoryHandler обрабатывает запросы к историческим метрикам
type MetricsHistoryHandler struct {
	config  *config.Config
	logger  *logrus.Logger
	storage *metrics.MetricsStorage
}

// NewMetricsHistoryHandler создает новый handler
func NewMetricsHistoryHandler(cfg *config.Config, logger *logrus.Logger, storage *metrics.MetricsStorage) *MetricsHistoryHandler {
	return &MetricsHistoryHandler{
		config:  cfg,
		logger:  logger,
		storage: storage,
	}
}

// GetHistory возвращает исторические данные метрик
// GET /api/metrics/history?type=request_latency&period=1h&interval=1m
func (h *MetricsHistoryHandler) GetHistory(c *gin.Context) {
	// Параметры запроса
	metricTypeStr := c.DefaultQuery("type", "request_latency")
	periodStr := c.DefaultQuery("period", "1h")
	intervalStr := c.DefaultQuery("interval", "1m")

	// Парсим metric type
	metricType := metrics.MetricType(metricTypeStr)

	// Парсим period
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid period format",
			"details": "use format like: 1h, 30m, 1d",
		})
		return
	}

	// Парсим interval
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid interval format",
			"details": "use format like: 1m, 5m, 1h",
		})
		return
	}

	// Вычисляем временной диапазон
	to := time.Now()
	from := to.Add(-period)

	// Получаем временной ряд
	timeSeries := h.storage.GetTimeSeries(metricType, from, to, int(interval.Seconds()))

	c.JSON(http.StatusOK, gin.H{
		"metric_type": metricTypeStr,
		"period":      periodStr,
		"interval":    intervalStr,
		"from":        from.Unix(),
		"to":          to.Unix(),
		"data_points": len(timeSeries),
		"time_series": timeSeries,
	})
}

// GetStats возвращает агрегированную статистику
// GET /api/metrics/stats?type=request_latency
func (h *MetricsHistoryHandler) GetStats(c *gin.Context) {
	metricTypeStr := c.DefaultQuery("type", "request_latency")
	metricType := metrics.MetricType(metricTypeStr)

	stats := h.storage.GetStats(metricType)

	c.JSON(http.StatusOK, gin.H{
		"metric_type": metricTypeStr,
		"stats":       stats,
	})
}

// GetAllStats возвращает статистику для всех метрик
// GET /api/metrics/stats/all
func (h *MetricsHistoryHandler) GetAllStats(c *gin.Context) {
	allStats := h.storage.GetAllStats()

	c.JSON(http.StatusOK, gin.H{
		"metrics": allStats,
	})
}

// GetBufferInfo возвращает информацию о состоянии buffers
// GET /api/metrics/buffers
func (h *MetricsHistoryHandler) GetBufferInfo(c *gin.Context) {
	bufferInfo := h.storage.GetBufferInfo()

	c.JSON(http.StatusOK, gin.H{
		"buffers": bufferInfo,
	})
}

// GetRecentData возвращает последние N записей
// GET /api/metrics/recent?type=request_latency&count=100
func (h *MetricsHistoryHandler) GetRecentData(c *gin.Context) {
	metricTypeStr := c.DefaultQuery("type", "request_latency")
	countStr := c.DefaultQuery("count", "100")

	metricType := metrics.MetricType(metricTypeStr)

	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		count = 100
	}
	if count > 1000 {
		count = 1000 // Max 1000 records
	}

	data := h.storage.GetRecentData(metricType, count)

	c.JSON(http.StatusOK, gin.H{
		"metric_type": metricTypeStr,
		"count":       len(data),
		"requested":   count,
		"data":        data,
	})
}

// ClearMetrics очищает все metrics buffers (admin only)
// POST /api/metrics/clear
func (h *MetricsHistoryHandler) ClearMetrics(c *gin.Context) {
	h.storage.Clear()

	c.JSON(http.StatusOK, gin.H{
		"message": "All metrics buffers cleared",
	})
}

// GetMetricTypes возвращает список доступных типов метрик
// GET /api/metrics/types
func (h *MetricsHistoryHandler) GetMetricTypes(c *gin.Context) {
	types := []string{
		string(metrics.MetricTypeRequestCount),
		string(metrics.MetricTypeRequestLatency),
		string(metrics.MetricTypeErrorCount),
		string(metrics.MetricTypeRequestSize),
		string(metrics.MetricTypeResponseSize),
		string(metrics.MetricTypeActiveRequests),
	}

	c.JSON(http.StatusOK, gin.H{
		"metric_types": types,
		"count":        len(types),
	})
}
