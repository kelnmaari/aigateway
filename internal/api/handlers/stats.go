// Package handlers provides HTTP handlers for Ollama-OpenAI Proxy
package handlers

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/metrics"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// GlobalStats хранит глобальную статистику приложения
var GlobalStats = &Stats{
	StartTime: time.Now(),
}

// Stats содержит статистику работы прокси
type Stats struct {
	StartTime time.Time

	// Счетчики запросов (используем atomic для thread-safety)
	TotalRequests   int64
	ActiveRequests  int64
	SuccessRequests int64
	ErrorRequests   int64

	// Метрики производительности
	TotalDuration   time.Duration
	AverageDuration time.Duration
}

// IncrementTotalRequests увеличивает счетчик всех запросов
func (s *Stats) IncrementTotalRequests() {
	atomic.AddInt64(&s.TotalRequests, 1)
}

// IncrementActiveRequests увеличивает счетчик активных запросов
func (s *Stats) IncrementActiveRequests() {
	atomic.AddInt64(&s.ActiveRequests, 1)
}

// DecrementActiveRequests уменьшает счетчик активных запросов
func (s *Stats) DecrementActiveRequests() {
	atomic.AddInt64(&s.ActiveRequests, -1)
}

// IncrementSuccessRequests увеличивает счетчик успешных запросов
func (s *Stats) IncrementSuccessRequests() {
	atomic.AddInt64(&s.SuccessRequests, 1)
}

// IncrementErrorRequests увеличивает счетчик ошибочных запросов
func (s *Stats) IncrementErrorRequests() {
	atomic.AddInt64(&s.ErrorRequests, 1)
}

// GetUptime возвращает время работы сервера
func (s *Stats) GetUptime() time.Duration {
	return time.Since(s.StartTime)
}

// Snapshot возвращает текущий снимок статистики
func (s *Stats) Snapshot() gin.H {
	total := atomic.LoadInt64(&s.TotalRequests)
	active := atomic.LoadInt64(&s.ActiveRequests)
	success := atomic.LoadInt64(&s.SuccessRequests)
	errors := atomic.LoadInt64(&s.ErrorRequests)

	avgDuration := "N/A"
	if success > 0 {
		avgDuration = (s.TotalDuration / time.Duration(success)).String()
	}

	return gin.H{
		"uptime_seconds":   s.GetUptime().Seconds(),
		"uptime":           s.GetUptime().String(),
		"total_requests":   total,
		"active_requests":  active,
		"success_requests": success,
		"error_requests":   errors,
		"average_duration": avgDuration,
	}
}

// APIKeyManager определяет интерфейс для работы с API ключами
type APIKeyManager interface {
	ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) (*models.ListAPIKeysResponse, error)
}

// MetricsStorageInterface определяет интерфейс для получения метрик
type MetricsStorageInterface interface {
	GetStats(metricType metrics.MetricType) metrics.AggregatedStats
}

// StatsHandler обрабатывает эндпоинт статистики для TUI
type StatsHandler struct {
	config         *config.Config
	logger         *logrus.Logger
	ollamaClient   OllamaClientInterface
	keyManager     APIKeyManager    // Legacy JSON storage (deprecated)
	db             storage.Database // Database for API keys (Version 1.3.0+)
	stats          *Stats
	version        string                  // Версия сервера
	metricsStorage MetricsStorageInterface // Для latency данных
}

// NewStatsHandler создает новый stats handler
func NewStatsHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface, keyMgr APIKeyManager, version string, metricsStorage MetricsStorageInterface, db storage.Database) *StatsHandler {
	return &StatsHandler{
		config:         cfg,
		logger:         logger,
		ollamaClient:   ollamaClient,
		keyManager:     keyMgr,
		db:             db,
		stats:          GlobalStats,
		version:        version,
		metricsStorage: metricsStorage,
	}
}

// GetStats обрабатывает GET /api/stats
func (h *StatsHandler) GetStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	// Проверяем подключение к Ollama
	ollamaConnected := false
	if err := h.ollamaClient.Health(ctx); err == nil {
		ollamaConnected = true
	}

	// Получаем список моделей от Ollama
	var modelsCount int
	var modelsList []string
	if ollamaConnected {
		if models, err := h.ollamaClient.GetModels(ctx); err == nil {
			modelsCount = len(models.Models)
			for _, model := range models.Models {
				modelsList = append(modelsList, model.Name)
			}
		}
	}

	// Получаем информацию об API ключах из БД (Version 1.3.0+) или legacy JSON storage
	apiKeysInfo := gin.H{
		"enabled": h.config.Auth.Enabled,
		"count":   0,
		"keys":    []gin.H{},
	}

	// Приоритет: Database → Legacy JSON storage
	if h.db != nil {
		// Используем БД (Version 1.3.0+)
		dbKeys, err := h.db.ListAPIKeys(ctx)
		if err == nil && dbKeys != nil {
			apiKeysInfo["count"] = len(dbKeys)

			// Формируем упрощенный список для TUI
			var keysList []gin.H
			for _, key := range dbKeys {
				keysList = append(keysList, gin.H{
					"id":          key.ID,
					"name":        key.Name,
					"status":      key.Status,
					"permissions": key.Permissions,
					"models":      key.Models,
					"created_at":  key.CreatedAt.Format("2006-01-02 15:04"),
					"last_used":   formatLastUsed(key.LastUsedAt),
				})
			}
			apiKeysInfo["keys"] = keysList
		}
	} else if h.keyManager != nil {
		// Fallback на legacy JSON storage
		req := models.ListAPIKeysRequest{
			Limit:  100,
			Offset: 0,
		}
		response, err := h.keyManager.ListAPIKeys(ctx, req)
		if err == nil && response != nil {
			apiKeysInfo["count"] = response.Total

			// Формируем упрощенный список для TUI
			var keysList []gin.H
			for _, key := range response.APIKeys {
				keysList = append(keysList, gin.H{
					"id":          key.ID,
					"name":        key.Name,
					"status":      key.Status,
					"permissions": key.Permissions,
					"models":      key.Models,
					"created_at":  key.CreatedAt.Format("2006-01-02 15:04"),
					"last_used":   formatLastUsed(key.LastUsedAt),
					"usage": gin.H{
						"total_requests": key.Usage.TotalRequests,
						"success":        key.Usage.SuccessfulRequests,
						"failed":         key.Usage.FailedRequests,
					},
				})
			}
			apiKeysInfo["keys"] = keysList
		}
	}

	// Получаем latency метрики если доступны
	latencyInfo := gin.H{}
	if h.metricsStorage != nil {
		latencyStats := h.metricsStorage.GetStats(metrics.MetricTypeRequestLatency)
		latencyInfo["avg"] = int(latencyStats.Avg)
		latencyInfo["median"] = int(latencyStats.Median)
		latencyInfo["p95"] = int(latencyStats.P95)
		latencyInfo["p99"] = int(latencyStats.P99)
	}

	response := gin.H{
		"server": gin.H{
			"status":  "running",
			"host":    h.config.Server.Host,
			"port":    h.config.Server.Port,
			"address": h.config.GetServerAddr(),
			"version": h.version,
		},
		"ollama": gin.H{
			"connected":    ollamaConnected,
			"url":          h.config.Ollama.URL,
			"models_count": modelsCount,
			"models":       modelsList,
		},
		"stats":    h.stats.Snapshot(),
		"latency":  latencyInfo,
		"api_keys": apiKeysInfo,
	}

	h.logger.Debug("Stats requested by TUI")
	c.JSON(http.StatusOK, response)
}

// formatLastUsed форматирует время последнего использования
func formatLastUsed(t *time.Time) string {
	if t == nil {
		return "Never"
	}
	return t.Format("2006-01-02 15:04")
}
