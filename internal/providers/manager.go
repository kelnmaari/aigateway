package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"
	"github.com/sirupsen/logrus"
)

// ProviderManager управляет всеми зарегистрированными providers
type ProviderManager struct {
	db        storage.Database
	logger    *logrus.Logger
	providers map[string]Provider // provider_id -> Provider instance
	mu        sync.RWMutex
}

// NewProviderManager создает новый ProviderManager
func NewProviderManager(db storage.Database, logger *logrus.Logger) *ProviderManager {
	return &ProviderManager{
		db:        db,
		logger:    logger,
		providers: make(map[string]Provider),
	}
}

// RegisterProvider регистрирует provider в manager
func (pm *ProviderManager) RegisterProvider(providerID string, provider Provider) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.providers[providerID] = provider
	pm.logger.WithFields(logrus.Fields{
		"provider_id":   providerID,
		"provider_name": provider.GetName(),
		"provider_type": provider.GetType(),
	}).Info("Provider registered")
}

// GetProvider возвращает provider по ID
func (pm *ProviderManager) GetProvider(providerID string) (Provider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, exists := pm.providers[providerID]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", providerID)
	}

	return provider, nil
}

// ListProviders возвращает список всех зарегистрированных providers
func (pm *ProviderManager) ListProviders() []Provider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make([]Provider, 0, len(pm.providers))
	for _, p := range pm.providers {
		providers = append(providers, p)
	}

	return providers
}

// LoadProvidersFromDB загружает providers из БД и создает instances
func (pm *ProviderManager) LoadProvidersFromDB(ctx context.Context) error {
	// Получаем список enabled providers из БД
	providerConfigs, err := pm.db.ListModelProviders(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to load providers from DB: %w", err)
	}

	pm.logger.Infof("Loading %d providers from database", len(providerConfigs))

	for _, config := range providerConfigs {
		var provider Provider

		switch config.ProviderType {
		case models.ProviderTypeVLLM:
			provider = NewVLLMProvider(config.Name, config.BaseURL)
		case models.ProviderTypeOpenAI:
			provider = NewOpenAIProvider(config.Name, config.BaseURL, config.APIKey)
		case models.ProviderTypeAnthropic:
			provider = NewAnthropicProvider(config.Name, config.BaseURL, config.APIKey)
		case models.ProviderTypeGemini:
			provider = NewGeminiProvider(config.Name, config.BaseURL, config.APIKey)
		case models.ProviderTypeDeepSeek:
			provider = NewDeepSeekProvider(config.Name, config.BaseURL, config.APIKey)
		default:
			pm.logger.Warnf("Unsupported provider type: %s (provider: %s)", config.ProviderType, config.Name)
			continue
		}

		pm.RegisterProvider(config.ID, provider)
	}

	return nil
}

// HealthCheckAll проверяет health всех providers
func (pm *ProviderManager) HealthCheckAll(ctx context.Context) map[string]ProviderHealth {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	results := make(map[string]ProviderHealth)

	for providerID, provider := range pm.providers {
		startTime := time.Now()
		err := provider.HealthCheck(ctx)
		latency := time.Since(startTime)

		health := ProviderHealth{
			CheckedAt: time.Now(),
			Latency:   latency,
		}

		if err != nil {
			health.Status = models.HealthStatusUnhealthy
			health.ErrorMsg = err.Error()
		} else {
			health.Status = models.HealthStatusHealthy
		}

		results[providerID] = health

		// Update health status in DB
		go func(id string, h ProviderHealth) {
			updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := pm.db.UpdateModelProviderHealth(updateCtx, id, h.Status, h.ErrorMsg); err != nil {
				pm.logger.WithError(err).Errorf("Failed to update provider health: %s", id)
			}
		}(providerID, health)
	}

	return results
}

// DiscoverModels обнаруживает модели от всех providers и сохраняет в registry
func (pm *ProviderManager) DiscoverModels(ctx context.Context) (int, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	totalDiscovered := 0

	for providerID, provider := range pm.providers {
		pm.logger.Infof("Discovering models from provider: %s (%s)", provider.GetName(), provider.GetType())

		providerModels, err := provider.ListModels(ctx)
		if err != nil {
			pm.logger.WithError(err).Errorf("Failed to list models from provider: %s", providerID)
			continue
		}

		pm.logger.Infof("Found %d models from %s", len(providerModels), provider.GetName())

		// Сохраняем модели в registry
		for _, providerModel := range providerModels {
			// Проверяем существует ли модель уже
			existing, err := pm.db.GetModelRegistryByModelID(ctx, providerModel.ID)
			if err == nil && existing != nil {
				// Модель уже существует, пропускаем
				pm.logger.Debugf("Model already exists in registry: %s", providerModel.ID)
				continue
			}

		// Создаем новую запись в registry
		registryModel := &models.ModelRegistry{
			ModelID:       providerModel.ID,
			ModelName:     providerModel.Name,
			ProviderID:    providerID,
			Capabilities:  providerModel.Capabilities,
			Parameters:    providerModel.Parameters,
			RequiresGPU:   providerModel.RequiresGPU,
			MinVRAMGB:     providerModel.MinVRAMGB,
			ContextLength: providerModel.ContextLength,
			Description:   providerModel.Description,
			Tags:          providerModel.Tags,
			Status:        models.ModelStatusActive,
			HealthStatus:  models.HealthStatusUnknown,
		}
		
		// Ensure JSON fields are not nil for database insert
		if registryModel.Capabilities == nil {
			registryModel.Capabilities = []models.ModelCapability{}
		}
		if registryModel.Parameters == nil {
			registryModel.Parameters = make(map[string]interface{})
		}
		if registryModel.Tags == nil {
			registryModel.Tags = []string{}
		}

			if err := pm.db.CreateModelRegistry(ctx, registryModel); err != nil {
				pm.logger.WithError(err).Errorf("Failed to register model: %s", providerModel.ID)
				continue
			}

			totalDiscovered++
			pm.logger.Infof("Registered model: %s from %s", providerModel.ID, provider.GetName())
		}
	}

	pm.logger.Infof("Discovery complete: %d new models registered", totalDiscovered)
	return totalDiscovered, nil
}

// DiscoverModelsFromProvider обнаруживает модели от одного provider и сохраняет в registry.
// Возвращает количество новых моделей и список всех обнаруженных моделей.
func (pm *ProviderManager) DiscoverModelsFromProvider(ctx context.Context, providerID string) (int, []*models.ModelRegistry, error) {
	pm.mu.RLock()
	provider, exists := pm.providers[providerID]
	pm.mu.RUnlock()

	if !exists {
		return 0, nil, fmt.Errorf("provider not found: %s", providerID)
	}

	pm.logger.Infof("Discovering models from provider: %s (%s)", provider.GetName(), provider.GetType())

	providerModels, err := provider.ListModels(ctx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list models from provider %s: %w", providerID, err)
	}

	pm.logger.Infof("Found %d models from %s", len(providerModels), provider.GetName())

	var discovered int
	var allModels []*models.ModelRegistry

	for _, providerModel := range providerModels {
		// Проверяем существует ли модель уже
		existing, err := pm.db.GetModelRegistryByModelID(ctx, providerModel.ID)
		if err == nil && existing != nil {
			allModels = append(allModels, existing)
			continue
		}

		// Создаем новую запись в registry
		registryModel := &models.ModelRegistry{
			ModelID:      providerModel.ID,
			ModelName:    providerModel.Name,
			ProviderID:   providerID,
			Capabilities: []models.ModelCapability{},
			Parameters:   make(map[string]interface{}),
			Tags:         []string{},
			Status:       models.ModelStatusActive,
			HealthStatus: models.HealthStatusUnknown,
		}

		if err := pm.db.CreateModelRegistry(ctx, registryModel); err != nil {
			pm.logger.WithError(err).Errorf("Failed to register model: %s", providerModel.ID)
			continue
		}

		discovered++
		allModels = append(allModels, registryModel)
		pm.logger.Infof("Registered model: %s from %s", providerModel.ID, provider.GetName())
	}

	pm.logger.Infof("Provider discovery complete for %s: %d new models registered", providerID, discovered)
	return discovered, allModels, nil
}

// RunDiscoveryLoop запускает периодическое обнаружение моделей
func (pm *ProviderManager) RunDiscoveryLoop(ctx context.Context, interval time.Duration) {
	pm.logger.Infof("Starting model discovery loop with interval: %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pm.logger.Info("Model discovery loop stopped")
			return
		case <-ticker.C:
			pm.logger.Debug("Running scheduled model discovery...")
			discovered, err := pm.DiscoverModels(ctx)
			if err != nil {
				pm.logger.WithError(err).Warn("Scheduled model discovery failed")
			} else {
				pm.logger.Debugf("Scheduled model discovery complete: %d new models", discovered)
			}
		}
	}
}

// RunHealthCheckLoop запускает периодическую проверку здоровья providers
func (pm *ProviderManager) RunHealthCheckLoop(ctx context.Context, interval time.Duration) {
	pm.logger.Infof("Starting provider health check loop with interval: %s", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			pm.logger.Info("Provider health check loop stopped")
			return
		case <-ticker.C:
			pm.logger.Debug("Running scheduled provider health check...")
			healthResults := pm.HealthCheckAll(ctx)
			unhealthyCount := 0
			for _, health := range healthResults {
				if health.Status == models.HealthStatusUnhealthy {
					unhealthyCount++
				}
			}
			if unhealthyCount > 0 {
				pm.logger.Warnf("Health check complete: %d/%d providers unhealthy", unhealthyCount, len(healthResults))
			} else {
				pm.logger.Debugf("Health check complete: all %d providers healthy", len(healthResults))
			}
		}
	}
}

