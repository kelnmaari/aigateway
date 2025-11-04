package router

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/providers"
	"aigateway/internal/storage"

	"github.com/sirupsen/logrus"
)

// ModelRouter обеспечивает smart routing для model requests
type ModelRouter struct {
	db              storage.Database
	providerManager *providers.ProviderManager
	logger          *logrus.Logger

	// Cache для быстрого поиска provider по model_id
	modelProviderCache map[string]string // model_id -> provider_id
	cacheMu            sync.RWMutex
	cacheExpiry        time.Time
}

// RouteConfig конфигурация маршрутизации для конкретного запроса
type RouteConfig struct {
	ModelID        string
	RequireHealthy bool                     // Требовать только healthy providers
	AllowFallback  bool                     // Разрешить fallback на другие providers
	PreferredType  models.ModelProviderType // Предпочитаемый тип provider (ollama/vllm)
	Timeout        time.Duration
}

// RouteResult результат маршрутизации
type RouteResult struct {
	Provider           providers.Provider
	ProviderID         string
	Model              *models.ModelRegistry
	IsFallback         bool
	FallbackReason     string
	AttemptedProviders []string // Список попыток для отладки
}

// NewModelRouter создает новый ModelRouter
func NewModelRouter(db storage.Database, providerManager *providers.ProviderManager, logger *logrus.Logger) *ModelRouter {
	return &ModelRouter{
		db:                 db,
		providerManager:    providerManager,
		logger:             logger,
		modelProviderCache: make(map[string]string),
		cacheExpiry:        time.Now().Add(5 * time.Minute),
	}
}

// Route выбирает оптимальный provider для обработки запроса
func (r *ModelRouter) Route(ctx context.Context, config RouteConfig) (*RouteResult, error) {
	r.logger.WithFields(logrus.Fields{
		"model_id":        config.ModelID,
		"require_healthy": config.RequireHealthy,
		"allow_fallback":  config.AllowFallback,
	}).Debug("Routing model request")

	// 1. Поиск модели в registry
	model, err := r.getModelFromRegistry(ctx, config.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found in registry: %w", err)
	}

	result := &RouteResult{
		Model:              model,
		AttemptedProviders: []string{},
	}

	// 2. Попытка маршрутизации к primary provider
	provider, providerID, err := r.routeToPrimaryProvider(ctx, model, config)
	if err == nil && provider != nil {
		result.Provider = provider
		result.ProviderID = providerID
		result.IsFallback = false
		r.logger.WithField("provider_id", providerID).Info("Routed to primary provider")
		return result, nil
	}

	result.AttemptedProviders = append(result.AttemptedProviders, model.ProviderID)
	r.logger.WithError(err).Warn("Primary provider unavailable")

	// 3. Fallback если разрешено
	if config.AllowFallback {
		provider, providerID, fallbackErr := r.findFallbackProvider(ctx, model, config, result)
		if fallbackErr == nil && provider != nil {
			result.Provider = provider
			result.ProviderID = providerID
			result.IsFallback = true
			result.FallbackReason = fmt.Sprintf("primary provider unavailable: %v", err)
			r.logger.WithFields(logrus.Fields{
				"fallback_provider": providerID,
				"reason":            result.FallbackReason,
			}).Info("Routed to fallback provider")
			return result, nil
		}
	}

	// 4. Не удалось найти доступный provider
	return nil, fmt.Errorf("no available provider for model %s (attempted: %v)",
		config.ModelID, result.AttemptedProviders)
}

// getModelFromRegistry получает модель из registry с кэшированием
func (r *ModelRouter) getModelFromRegistry(ctx context.Context, modelID string) (*models.ModelRegistry, error) {
	// Проверяем кэш provider mapping
	r.cacheMu.RLock()
	cacheExpired := time.Now().After(r.cacheExpiry)
	r.cacheMu.RUnlock()

	if cacheExpired {
		r.invalidateCache()
	}

	// Получаем модель из БД
	model, err := r.db.GetModelRegistryByModelID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model from registry: %w", err)
	}

	// Обновляем кэш
	r.cacheMu.Lock()
	r.modelProviderCache[modelID] = model.ProviderID
	r.cacheMu.Unlock()

	return model, nil
}

// routeToPrimaryProvider пытается маршрутизировать к primary provider модели
func (r *ModelRouter) routeToPrimaryProvider(ctx context.Context, model *models.ModelRegistry, config RouteConfig) (providers.Provider, string, error) {
	// Получаем provider из manager
	provider, err := r.providerManager.GetProvider(model.ProviderID)
	if err != nil {
		return nil, "", fmt.Errorf("provider not found: %w", err)
	}

	// Проверяем health если требуется
	if config.RequireHealthy {
		if err := r.checkProviderHealth(ctx, provider); err != nil {
			return nil, "", fmt.Errorf("provider unhealthy: %w", err)
		}
	}

	return provider, model.ProviderID, nil
}

// findFallbackProvider ищет альтернативный provider с load balancing
func (r *ModelRouter) findFallbackProvider(ctx context.Context, model *models.ModelRegistry, config RouteConfig, result *RouteResult) (providers.Provider, string, error) {
	// Получаем список всех providers с моделью такого же типа
	filter := &models.ModelRegistryFilter{
		ProviderType: config.PreferredType,
		Status:       models.ModelStatusActive,
	}

	if config.RequireHealthy {
		filter.HealthStatus = models.HealthStatusHealthy
	}

	// Ищем модели с теми же capabilities
	candidateModels, err := r.db.ListModelRegistry(ctx, filter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list models for fallback: %w", err)
	}

	// Load Balancing: сортируем кандидатов по priority и метрикам
	sortedCandidates := r.sortByLoadBalancing(candidateModels, model.Capabilities)

	// Пытаемся найти модель с теми же capabilities
	for _, candidateModel := range sortedCandidates {
		// Пропускаем уже попробованные providers
		alreadyAttempted := slices.Contains(result.AttemptedProviders, candidateModel.ProviderID)
		if alreadyAttempted {
			continue
		}

		// Проверяем совпадение capabilities
		if !r.capabilitiesMatch(model.Capabilities, candidateModel.Capabilities) {
			continue
		}

		// Пытаемся получить provider
		provider, err := r.providerManager.GetProvider(candidateModel.ProviderID)
		if err != nil {
			result.AttemptedProviders = append(result.AttemptedProviders, candidateModel.ProviderID)
			r.logger.WithError(err).Warnf("Fallback provider %s not available", candidateModel.ProviderID)
			continue
		}

		// Проверяем health
		if config.RequireHealthy {
			if err := r.checkProviderHealth(ctx, provider); err != nil {
				result.AttemptedProviders = append(result.AttemptedProviders, candidateModel.ProviderID)
				r.logger.WithError(err).Warnf("Fallback provider %s unhealthy", candidateModel.ProviderID)
				continue
			}
		}

		// Нашли подходящий fallback provider!
		return provider, candidateModel.ProviderID, nil
	}

	return nil, "", fmt.Errorf("no fallback providers available")
}

// checkProviderHealth проверяет health provider
func (r *ModelRouter) checkProviderHealth(ctx context.Context, provider providers.Provider) error {
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return provider.HealthCheck(healthCtx)
}

// capabilitiesMatch проверяет совпадение capabilities двух моделей
func (r *ModelRouter) capabilitiesMatch(required, available []models.ModelCapability) bool {
	if len(required) == 0 {
		return true // Нет требований
	}

	// Создаем map для быстрой проверки
	availableMap := make(map[models.ModelCapability]bool)
	for _, cap := range available {
		availableMap[cap] = true
	}

	// Проверяем что все required capabilities присутствуют
	for _, cap := range required {
		if !availableMap[cap] {
			return false
		}
	}

	return true
}

// invalidateCache очищает кэш маршрутизации
func (r *ModelRouter) invalidateCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.modelProviderCache = make(map[string]string)
	r.cacheExpiry = time.Now().Add(5 * time.Minute)
	r.logger.Debug("Model-provider cache invalidated")
}

// sortByLoadBalancing сортирует candidates для load balancing
// Приоритет:
// 1. Модели с совпадающими capabilities
// 2. Лучшие performance metrics (latency, throughput)
// 3. Меньшее количество total requests (load распределение)
func (r *ModelRouter) sortByLoadBalancing(candidates []*models.ModelRegistry, requiredCaps []models.ModelCapability) []*models.ModelRegistry {
	// Фильтруем только модели с подходящими capabilities
	filtered := make([]*models.ModelRegistry, 0, len(candidates))
	for _, candidate := range candidates {
		if r.capabilitiesMatch(requiredCaps, candidate.Capabilities) {
			filtered = append(filtered, candidate)
		}
	}

	// Сортируем по метрикам (простая эвристика)
	// TODO: Можно улучшить с weighted scoring
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			// Сравниваем по latency (меньше = лучше)
			iLatency := float64(1000000) // Большое значение если нет данных
			if filtered[i].AvgLatencyMs != nil {
				iLatency = *filtered[i].AvgLatencyMs
			}

			jLatency := float64(1000000)
			if filtered[j].AvgLatencyMs != nil {
				jLatency = *filtered[j].AvgLatencyMs
			}

			// Если latency похожи, сравниваем по load (total requests)
			if jLatency < iLatency || (jLatency == iLatency && filtered[j].TotalRequests < filtered[i].TotalRequests) {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	return filtered
}

// GetCachedProvider возвращает кэшированный provider_id для model_id (для отладки)
func (r *ModelRouter) GetCachedProvider(modelID string) (string, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	providerID, exists := r.modelProviderCache[modelID]
	return providerID, exists
}
