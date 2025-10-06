// Package model provides model management for Ollama-OpenAI Proxy
package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/converter"
	"ollama-openai-proxy/internal/models"
)

// Manager управляет доступными моделями и их кешированием
type Manager struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
	modelManager converter.ModelManager

	// Кеш моделей
	mutex        sync.RWMutex
	cachedModels *ollama.ModelsResponse
	lastUpdate   time.Time
	updateTicker *time.Ticker
	stopChannel  chan struct{}

	// Статистика
	stats ManagerStats
}

// ManagerStats содержит статистику работы model manager
type ManagerStats struct {
	TotalRefreshes     int64     `json:"total_refreshes"`
	LastRefreshTime    time.Time `json:"last_refresh_time"`
	LastRefreshSuccess bool      `json:"last_refresh_success"`
	LastError          string    `json:"last_error,omitempty"`
	CacheHits          int64     `json:"cache_hits"`
	CacheMisses        int64     `json:"cache_misses"`
	ModelsCount        int       `json:"models_count"`
	AvailableModels    []string  `json:"available_models,omitempty"`
}

// NewManager создает новый model manager
func NewManager(cfg *config.Config, logger *logrus.Logger, ollamaClient *ollama.Client) *Manager {
	modelManager := converter.NewDefaultModelManager(cfg, logger)

	m := &Manager{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
		modelManager: modelManager,
		stopChannel:  make(chan struct{}),
		stats: ManagerStats{
			LastRefreshTime: time.Time{},
		},
	}

	// Запускаем периодическое обновление если включено
	if cfg.Models.Cache.Enabled && cfg.Models.Cache.RefreshInterval > 0 {
		m.startPeriodicRefresh()
	}

	return m
}

// NewManagerWithCircuitBreaker создает model manager с circuit breaker клиентом
func NewManagerWithCircuitBreaker(cfg *config.Config, logger *logrus.Logger, ollamaClient interface{}) *Manager {
	// Адаптер для circuit breaker клиента
	var client OllamaClientInterface

	if circuitClient, ok := ollamaClient.(*ollama.ClientWithCircuitBreaker); ok {
		client = circuitClient
	} else if regularClient, ok := ollamaClient.(*ollama.Client); ok {
		client = regularClient
	} else {
		// Fallback на обычный клиент
		logger.Warn("Unknown client type, creating new client")
		regularClient, err := ollama.NewClient(cfg, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to create fallback client")
			return nil
		}
		client = regularClient
	}

	modelManager := converter.NewDefaultModelManager(cfg, logger)

	m := &Manager{
		config:       cfg,
		logger:       logger,
		ollamaClient: client,
		modelManager: modelManager,
		stopChannel:  make(chan struct{}),
		stats: ManagerStats{
			LastRefreshTime: time.Time{},
		},
	}

	// Запускаем периодическое обновление если включено
	if cfg.Models.Cache.Enabled && cfg.Models.Cache.RefreshInterval > 0 {
		m.startPeriodicRefresh()
	}

	return m
}

// GetModels возвращает список доступных моделей (с кешированием)
func (m *Manager) GetModels(ctx context.Context) (*models.ModelsResponse, error) {
	m.mutex.RLock()

	// Проверяем актуальность кеша
	if m.isCacheValid() {
		m.stats.CacheHits++
		cachedModels := m.cachedModels
		m.mutex.RUnlock()

		m.logger.Debug("Returning cached models")

		// Конвертируем в OpenAI формат через converter
		return m.convertModelsToOpenAI(cachedModels)
	}

	m.stats.CacheMisses++
	m.mutex.RUnlock()

	// Кеш устарел или отключен, получаем свежие данные
	return m.refreshAndGetModels(ctx)
}

// RefreshModels принудительно обновляет список моделей
func (m *Manager) RefreshModels(ctx context.Context) error {
	m.logger.Info("Manually refreshing models list")

	models, err := m.fetchModelsFromOllama(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh models: %w", err)
	}

	m.updateCache(models, true)
	return nil
}

// IsModelAvailable проверяет доступность конкретной модели
func (m *Manager) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	m.logger.WithField("model", modelName).Debug("Checking model availability")

	// Сначала пытаемся найти в кеше
	m.mutex.RLock()
	if m.isCacheValid() && m.cachedModels != nil {
		for _, model := range m.cachedModels.Models {
			if model.Name == modelName {
				m.mutex.RUnlock()
				m.logger.WithField("model", modelName).Debug("Model found in cache")
				return true, nil
			}
		}
		m.mutex.RUnlock()
	} else {
		m.mutex.RUnlock()
	}

	// Если не найдено в кеше, запрашиваем напрямую у Ollama
	return m.ollamaClient.IsModelAvailable(ctx, modelName)
}

// GetModelInfo возвращает детальную информацию о модели
func (m *Manager) GetModelInfo(ctx context.Context, modelName string) (*ollama.ShowResponse, error) {
	m.logger.WithField("model", modelName).Debug("Getting model information")

	return m.ollamaClient.ShowModel(ctx, modelName)
}

// GetSupportedModels возвращает список поддерживаемых моделей для OpenAI API
func (m *Manager) GetSupportedModels() []string {
	return m.modelManager.ListSupportedModels(models.APITypeOpenAI)
}

// GetModelManager возвращает converter model manager
func (m *Manager) GetModelManager() converter.ModelManager {
	return m.modelManager
}

// GetStats возвращает статистику работы manager'а
func (m *Manager) GetStats() ManagerStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := m.stats

	// Обновляем список доступных моделей
	if m.cachedModels != nil {
		stats.ModelsCount = len(m.cachedModels.Models)
		stats.AvailableModels = make([]string, len(m.cachedModels.Models))
		for i, model := range m.cachedModels.Models {
			stats.AvailableModels[i] = model.Name
		}
	}

	return stats
}

// Stop останавливает model manager и очищает ресурсы
func (m *Manager) Stop() {
	m.logger.Info("Stopping model manager")

	if m.updateTicker != nil {
		m.updateTicker.Stop()
	}

	close(m.stopChannel)

	m.logger.Debug("Model manager stopped")
}

// Private methods

// refreshAndGetModels обновляет кеш и возвращает модели
func (m *Manager) refreshAndGetModels(ctx context.Context) (*models.ModelsResponse, error) {
	m.logger.Debug("Refreshing models cache")

	models, err := m.fetchModelsFromOllama(ctx)
	if err != nil {
		return nil, err
	}

	m.updateCache(models, true)

	// Конвертируем в OpenAI формат
	return m.convertModelsToOpenAI(models)
}

// fetchModelsFromOllama получает список моделей из Ollama
func (m *Manager) fetchModelsFromOllama(ctx context.Context) (*ollama.ModelsResponse, error) {
	start := time.Now()

	models, err := m.ollamaClient.GetModels(ctx)
	if err != nil {
		m.updateStats(false, err)
		return nil, fmt.Errorf("failed to fetch models from Ollama: %w", err)
	}

	duration := time.Since(start)
	m.logger.WithFields(logrus.Fields{
		"models_count": len(models.Models),
		"fetch_time":   duration,
	}).Debug("Successfully fetched models from Ollama")

	m.updateStats(true, nil)
	return models, nil
}

// updateCache обновляет кеш моделей
func (m *Manager) updateCache(models *ollama.ModelsResponse, success bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if success {
		m.cachedModels = models
		m.lastUpdate = time.Now()

		m.logger.WithFields(logrus.Fields{
			"models_count": len(models.Models),
			"cache_ttl":    m.config.Models.Cache.TTL,
		}).Debug("Updated models cache")
	}
}

// updateStats обновляет статистику
func (m *Manager) updateStats(success bool, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.stats.TotalRefreshes++
	m.stats.LastRefreshTime = time.Now()
	m.stats.LastRefreshSuccess = success

	if err != nil {
		m.stats.LastError = err.Error()
	} else {
		m.stats.LastError = ""
	}
}

// isCacheValid проверяет актуальность кеша
func (m *Manager) isCacheValid() bool {
	if !m.config.Models.Cache.Enabled {
		return false
	}

	if m.cachedModels == nil {
		return false
	}

	if m.config.Models.Cache.TTL > 0 {
		return time.Since(m.lastUpdate) < m.config.Models.Cache.TTL
	}

	// Если TTL не задан, кеш всегда валиден
	return true
}

// convertModelsToOpenAI конвертирует модели Ollama в OpenAI формат
func (m *Manager) convertModelsToOpenAI(ollamaModels *ollama.ModelsResponse) (*models.ModelsResponse, error) {
	// Создаем converter для конвертации
	conv := converter.NewConverter(m.config, m.logger)

	return conv.ConvertModelsResponse(ollamaModels)
}

// startPeriodicRefresh запускает периодическое обновление моделей
func (m *Manager) startPeriodicRefresh() {
	interval := m.config.Models.Cache.RefreshInterval
	if interval <= 0 {
		return
	}

	m.logger.WithField("interval", interval).Info("Starting periodic models refresh")

	m.updateTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-m.updateTicker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

				if err := m.RefreshModels(ctx); err != nil {
					m.logger.WithError(err).Error("Periodic models refresh failed")
				} else {
					m.logger.Debug("Periodic models refresh completed")
				}

				cancel()

			case <-m.stopChannel:
				m.logger.Debug("Stopping periodic models refresh")
				return
			}
		}
	}()
}

// GetModelByName возвращает информацию о модели по имени
func (m *Manager) GetModelByName(modelName string) (*ollama.Model, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.cachedModels == nil {
		return nil, fmt.Errorf("models cache is empty")
	}

	for _, model := range m.cachedModels.Models {
		if model.Name == modelName {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model '%s' not found", modelName)
}

// FilterModels возвращает модели отфильтрованные по критериям
func (m *Manager) FilterModels(filter ModelFilter) []ollama.Model {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.cachedModels == nil {
		return []ollama.Model{}
	}

	var filtered []ollama.Model

	for _, model := range m.cachedModels.Models {
		if filter.Matches(model) {
			filtered = append(filtered, model)
		}
	}

	return filtered
}

// ModelFilter интерфейс для фильтрации моделей
type ModelFilter interface {
	Matches(model ollama.Model) bool
}

// NameFilter фильтр по имени модели
type NameFilter struct {
	Pattern string
}

func (f NameFilter) Matches(model ollama.Model) bool {
	return model.Name == f.Pattern
}

// FamilyFilter фильтр по семейству модели
type FamilyFilter struct {
	Family string
}

func (f FamilyFilter) Matches(model ollama.Model) bool {
	return models.ExtractModelFamily(model.Name) == f.Family
}

// SizeFilter фильтр по размеру модели
type SizeFilter struct {
	MinSize int64
	MaxSize int64
}

func (f SizeFilter) Matches(model ollama.Model) bool {
	if f.MinSize > 0 && model.Size < f.MinSize {
		return false
	}
	if f.MaxSize > 0 && model.Size > f.MaxSize {
		return false
	}
	return true
}

// EnsureModelAvailable убеждается что модель доступна (загружена в Ollama)
func (m *Manager) EnsureModelAvailable(ctx context.Context, modelName string) error {
	// Проверяем доступность модели
	available, err := m.IsModelAvailable(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to check model availability: %w", err)
	}

	if available {
		m.logger.WithField("model", modelName).Debug("Model is already available")
		return nil
	}

	// Если модель недоступна, пытаемся её загрузить
	m.logger.WithField("model", modelName).Info("Model not available, attempting to pull")

	pullResp, err := m.ollamaClient.PullModel(ctx, modelName)
	if err != nil {
		return fmt.Errorf("failed to pull model '%s': %w", modelName, err)
	}

	m.logger.WithFields(logrus.Fields{
		"model":  modelName,
		"status": pullResp.Status,
	}).Info("Model pull completed")

	// Обновляем кеш после загрузки модели
	if err := m.RefreshModels(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to refresh models cache after pull")
	}

	return nil
}

// ValidateAndMapModel проверяет и маппит модель
func (m *Manager) ValidateAndMapModel(modelName string) (string, error) {
	// Попытка маппинга OpenAI модели в Ollama
	ollamaModel, err := m.modelManager.MapOpenAIToOllama(modelName)
	if err != nil {
		// Если маппинг не найден, используем исходное имя
		m.logger.WithField("model", modelName).Debug("No mapping found, using original model name")
		return modelName, nil
	}

	m.logger.WithFields(logrus.Fields{
		"original_model": modelName,
		"mapped_model":   ollamaModel,
	}).Debug("Successfully mapped model")

	return ollamaModel, nil
}

// GetCachedModelsCount возвращает количество кешированных моделей
func (m *Manager) GetCachedModelsCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.cachedModels == nil {
		return 0
	}

	return len(m.cachedModels.Models)
}

// IsCacheEnabled проверяет включено ли кеширование
func (m *Manager) IsCacheEnabled() bool {
	return m.config.Models.Cache.Enabled
}

// GetCacheAge возвращает возраст кеша
func (m *Manager) GetCacheAge() time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.lastUpdate.IsZero() {
		return 0
	}

	return time.Since(m.lastUpdate)
}

// ClearCache очищает кеш моделей
func (m *Manager) ClearCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.cachedModels = nil
	m.lastUpdate = time.Time{}

	m.logger.Info("Models cache cleared")
}

// GetAvailableModelNames возвращает только имена доступных моделей
func (m *Manager) GetAvailableModelNames(ctx context.Context) ([]string, error) {
	models, err := m.GetModels(ctx)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(models.Data))
	for i, model := range models.Data {
		names[i] = model.ID
	}

	return names, nil
}

// GetModelCapabilities возвращает возможности модели
func (m *Manager) GetModelCapabilities(modelName string) (*models.ModelMappingConfig, error) {
	return m.modelManager.GetModelCapabilities(modelName)
}
