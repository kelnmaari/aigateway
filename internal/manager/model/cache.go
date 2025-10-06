// Package model provides caching functionality for model manager
package model

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
)

// CacheManager управляет кешированием моделей
type CacheManager struct {
	*Manager
}

// updateCacheWithValidation обновляет кеш с валидацией
func (m *Manager) updateCacheWithValidation(models *ollama.ModelsResponse, success bool) {
	if models == nil {
		m.logger.Warn("Attempted to cache nil models response")
		return
	}

	// Валидация моделей
	validModels := make([]ollama.Model, 0, len(models.Models))
	hiddenModels := make(map[string]bool)

	// Создаем map скрытых моделей для быстрого поиска
	for _, hidden := range m.config.Models.Hidden {
		hiddenModels[hidden] = true
	}

	for _, model := range models.Models {
		// Пропускаем скрытые модели
		if hiddenModels[model.Name] {
			m.logger.WithField("model", model.Name).Debug("Skipping hidden model")
			continue
		}

		// Валидация модели
		if m.validateModel(model) {
			validModels = append(validModels, model)
		} else {
			m.logger.WithField("model", model.Name).Warn("Invalid model, skipping")
		}
	}

	// Обновляем кеш с валидными моделями
	validatedResponse := &ollama.ModelsResponse{
		Models: validModels,
	}

	m.updateCache(validatedResponse, success)

	m.logger.WithFields(logrus.Fields{
		"total_models":  len(models.Models),
		"valid_models":  len(validModels),
		"hidden_models": len(models.Models) - len(validModels),
	}).Debug("Updated cache with validated models")
}

// validateModel проверяет валидность модели
func (m *Manager) validateModel(model ollama.Model) bool {
	// Проверка обязательных полей
	if model.Name == "" {
		return false
	}

	if model.Size < 0 {
		return false
	}

	// Проверка формата даты
	if model.ModifiedAt.IsZero() {
		m.logger.WithField("model", model.Name).Debug("Model has zero modification time")
	}

	return true
}

// warmupCache предварительно загружает кеш
func (m *Manager) warmupCache(ctx context.Context) {
	m.logger.Info("Warming up models cache")

	if err := m.RefreshModels(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to warmup models cache")
	} else {
		m.logger.Info("Models cache warmed up successfully")
	}
}

// getCacheInfo возвращает информацию о состоянии кеша
func (m *Manager) getCacheInfo() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	info := map[string]interface{}{
		"enabled":          m.config.Models.Cache.Enabled,
		"ttl":              m.config.Models.Cache.TTL.String(),
		"refresh_interval": m.config.Models.Cache.RefreshInterval.String(),
	}

	if m.lastUpdate.IsZero() {
		info["status"] = "empty"
		info["age"] = "never_updated"
		info["models_count"] = 0
	} else {
		info["status"] = "populated"
		info["age"] = time.Since(m.lastUpdate).String()
		info["last_update"] = m.lastUpdate.Format(time.RFC3339)
		info["valid"] = m.isCacheValid()

		if m.cachedModels != nil {
			info["models_count"] = len(m.cachedModels.Models)
		}
	}

	return info
}

// evictExpiredCache проверяет и очищает устаревший кеш
func (m *Manager) evictExpiredCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.config.Models.Cache.Enabled {
		return
	}

	if m.cachedModels == nil || m.lastUpdate.IsZero() {
		return
	}

	// Проверяем TTL
	if m.config.Models.Cache.TTL > 0 && time.Since(m.lastUpdate) > m.config.Models.Cache.TTL {
		m.logger.Debug("Cache expired, clearing")
		m.cachedModels = nil
		m.lastUpdate = time.Time{}
	}
}

// PrecacheModels предварительно кеширует популярные модели
func (m *Manager) PrecacheModels(ctx context.Context, modelNames []string) {
	if len(modelNames) == 0 {
		return
	}

	m.logger.WithFields(logrus.Fields{
		"models_count": len(modelNames),
		"models":       modelNames,
	}).Info("Precaching popular models")

	for _, modelName := range modelNames {
		if err := m.EnsureModelAvailable(ctx, modelName); err != nil {
			m.logger.WithError(err).WithField("model", modelName).Warn("Failed to precache model")
		} else {
			m.logger.WithField("model", modelName).Debug("Model precached successfully")
		}
	}
}

// GetCacheMetrics возвращает метрики кеша
func (m *Manager) GetCacheMetrics() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	metrics := map[string]interface{}{
		"cache_hits":     m.stats.CacheHits,
		"cache_misses":   m.stats.CacheMisses,
		"total_requests": m.stats.CacheHits + m.stats.CacheMisses,
	}

	// Вычисляем hit rate
	totalRequests := m.stats.CacheHits + m.stats.CacheMisses
	if totalRequests > 0 {
		hitRate := float64(m.stats.CacheHits) / float64(totalRequests) * 100
		metrics["hit_rate_percent"] = hitRate
	} else {
		metrics["hit_rate_percent"] = 0.0
	}

	// Информация о кеше
	if m.cachedModels != nil {
		metrics["cached_models_count"] = len(m.cachedModels.Models)
		metrics["cache_age_seconds"] = time.Since(m.lastUpdate).Seconds()
		metrics["cache_valid"] = m.isCacheValid()
	} else {
		metrics["cached_models_count"] = 0
		metrics["cache_age_seconds"] = 0
		metrics["cache_valid"] = false
	}

	return metrics
}
