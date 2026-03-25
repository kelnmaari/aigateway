// Package redis provides Redis background synchronization workers
// Version: v3.0.6+ - Background Cache Sync
package redis

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// BackgroundSyncManager manages all background sync workers
type BackgroundSyncManager struct {
	manager *Manager
	logger  *logrus.Logger

	// Workers (exported for router access, v3.0.6+)
	StatsWorker     *StatsWorker
	ModelListWorker *ModelListWorker
	APIKeyWorker    *APIKeyWorker

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BackgroundWorkerConfig configuration for background workers
type BackgroundWorkerConfig struct {
	StatsInterval     time.Duration // Default: 10s
	ModelListInterval time.Duration // Default: 30s
	APIKeyTTL         time.Duration // Default: 15m
	Enabled           bool          // Enable/disable background sync
}

// NewBackgroundSyncManager creates a new background sync manager
func NewBackgroundSyncManager(manager *Manager, logger *logrus.Logger, config BackgroundWorkerConfig) *BackgroundSyncManager {
	if config.StatsInterval == 0 {
		config.StatsInterval = 10 * time.Second
	}
	if config.ModelListInterval == 0 {
		config.ModelListInterval = 30 * time.Second
	}
	if config.APIKeyTTL == 0 {
		config.APIKeyTTL = 15 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BackgroundSyncManager{
		manager: manager,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts all background workers
func (m *BackgroundSyncManager) Start() {
	m.logger.Info("🚀 Starting Redis background sync workers...")

	// Stats worker будет запущен при регистрации провайдера
	// Model list worker будет запущен при регистрации провайдера
	// API key worker работает on-demand (cache-through pattern)

	m.logger.Info("✅ Redis background sync manager started")
}

// Stop stops all background workers gracefully
func (m *BackgroundSyncManager) Stop() {
	m.logger.Info("⏹️  Stopping Redis background sync workers...")

	// Cancel context для всех воркеров
	m.cancel()

	// Ждем завершения всех воркеров
	m.wg.Wait()

	m.logger.Info("✅ All background workers stopped")
}

// ─────────────────────────────────────────────────────────────────────────────
// Stats Worker - Background stats aggregation
// ─────────────────────────────────────────────────────────────────────────────

// StatsProvider interface for collecting stats from application
type StatsProvider interface {
	CollectStats(ctx context.Context) (any, error)
}

// StatsWorker periodically updates stats in Redis
type StatsWorker struct {
	manager  *Manager
	logger   *logrus.Logger
	provider StatsProvider
	interval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewStatsWorker creates a new stats worker
func NewStatsWorker(manager *Manager, logger *logrus.Logger, provider StatsProvider, interval time.Duration) *StatsWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &StatsWorker{
		manager:  manager,
		logger:   logger,
		provider: provider,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the stats worker
func (w *StatsWorker) Start() {
	w.wg.Add(1)
	go w.run()
	w.logger.WithField("interval", w.interval).Info("📊 Stats worker started")
}

// Stop stops the stats worker
func (w *StatsWorker) Stop() {
	w.cancel()
	w.wg.Wait()
	w.logger.Info("📊 Stats worker stopped")
}

// run main worker loop
func (w *StatsWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Сразу обновляем при старте
	w.updateStats()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.updateStats()
		}
	}
}

// updateStats collects and caches stats
func (w *StatsWorker) updateStats() {
	ctx, cancel := context.WithTimeout(w.ctx, 5*time.Second)
	defer cancel()

	stats, err := w.provider.CollectStats(ctx)
	if err != nil {
		w.logger.WithError(err).Warn("Failed to collect stats")
		return
	}

	// Cache stats in Redis
	key := "stats:global"
	if err := w.manager.Cache.SetJSON(ctx, key, stats, w.interval*2); err != nil {
		w.logger.WithError(err).Warn("Failed to cache stats in Redis")
		return
	}

	w.logger.Debug("✅ Stats cached in Redis")
}

// GetCachedStats retrieves cached stats from Redis
func (w *StatsWorker) GetCachedStats(ctx context.Context) (any, error) {
	key := "stats:global"

	var stats any
	if err := w.manager.Cache.GetJSON(ctx, key, &stats); err != nil {
		// Fallback: collect fresh stats
		w.logger.Debug("Stats cache miss, collecting fresh stats")
		return w.provider.CollectStats(ctx)
	}

	return stats, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Model List Worker - Background model list sync
// ─────────────────────────────────────────────────────────────────────────────

// ModelListProvider interface for listing models
type ModelListProvider interface {
	ListModels(ctx context.Context) (any, error)
}

// ModelListWorker periodically updates model list in Redis
type ModelListWorker struct {
	manager  *Manager
	logger   *logrus.Logger
	provider ModelListProvider
	interval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewModelListWorker creates a new model list worker
func NewModelListWorker(manager *Manager, logger *logrus.Logger, provider ModelListProvider, interval time.Duration) *ModelListWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ModelListWorker{
		manager:  manager,
		logger:   logger,
		provider: provider,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the model list worker
func (w *ModelListWorker) Start() {
	w.wg.Add(1)
	go w.run()
	w.logger.WithField("interval", w.interval).Info("🤖 Model list worker started")
}

// Stop stops the model list worker
func (w *ModelListWorker) Stop() {
	w.cancel()
	w.wg.Wait()
	w.logger.Info("🤖 Model list worker stopped")
}

// run main worker loop
func (w *ModelListWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Сразу обновляем при старте
	w.updateModelList()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.updateModelList()
		}
	}
}

// updateModelList collects and caches model list
func (w *ModelListWorker) updateModelList() {
	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()

	models, err := w.provider.ListModels(ctx)
	if err != nil {
		w.logger.WithError(err).Warn("Failed to list models")
		return
	}

	// Cache models in Redis
	key := "models:list:inference"
	if err := w.manager.Cache.SetJSON(ctx, key, models, w.interval*2); err != nil {
		w.logger.WithError(err).Warn("Failed to cache model list in Redis")
		return
	}

	w.logger.Debug("✅ Model list cached in Redis")
}

// GetCachedModelList retrieves cached model list from Redis
func (w *ModelListWorker) GetCachedModelList(ctx context.Context) (any, error) {
	key := "models:list:inference"

	var models any
	if err := w.manager.Cache.GetJSON(ctx, key, &models); err != nil {
		// Fallback: list fresh models
		w.logger.Debug("Model list cache miss, listing fresh models")
		return w.provider.ListModels(ctx)
	}

	return models, nil
}

// InvalidateModelListCache invalidates model list cache (call on load/unload)
func (w *ModelListWorker) InvalidateModelListCache(ctx context.Context) error {
	key := "models:list:inference"
	if err := w.manager.Cache.Delete(ctx, key); err != nil {
		w.logger.WithError(err).Warn("Failed to invalidate model list cache")
		return err
	}

	// Сразу обновляем свежий список
	w.updateModelList()

	w.logger.Debug("✅ Model list cache invalidated and refreshed")
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// API Key Worker - Cache-through pattern for API keys
// ─────────────────────────────────────────────────────────────────────────────

// APIKeyProvider interface for loading API keys
type APIKeyProvider interface {
	GetAPIKey(ctx context.Context, keyID string) (any, error)
}

// APIKeyWorker provides cache-through access to API keys
type APIKeyWorker struct {
	manager  *Manager
	logger   *logrus.Logger
	provider APIKeyProvider
	ttl      time.Duration
}

// NewAPIKeyWorker creates a new API key worker
func NewAPIKeyWorker(manager *Manager, logger *logrus.Logger, provider APIKeyProvider, ttl time.Duration) *APIKeyWorker {
	return &APIKeyWorker{
		manager:  manager,
		logger:   logger,
		provider: provider,
		ttl:      ttl,
	}
}

// GetAPIKey retrieves API key from cache or DB (cache-through)
func (w *APIKeyWorker) GetAPIKey(ctx context.Context, keyID string) (any, error) {
	// Try cache first
	cacheKey := "apikey:" + keyID

	var apiKey any
	if err := w.manager.Cache.GetJSON(ctx, cacheKey, &apiKey); err == nil {
		w.logger.WithField("key_id", keyID).Debug("✅ API key cache HIT")
		return apiKey, nil
	}

	// Cache miss - load from DB
	w.logger.WithField("key_id", keyID).Debug("⚠️  API key cache MISS, loading from DB")

	apiKey, err := w.provider.GetAPIKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	if err := w.manager.Cache.SetJSON(ctx, cacheKey, apiKey, w.ttl); err != nil {
		w.logger.WithError(err).Warn("Failed to cache API key")
		// Не фейлим запрос если кэширование не удалось
	}

	w.logger.WithField("key_id", keyID).Debug("✅ API key loaded from DB and cached")
	return apiKey, nil
}

// InvalidateAPIKey invalidates specific API key cache
func (w *APIKeyWorker) InvalidateAPIKey(ctx context.Context, keyID string) error {
	cacheKey := "apikey:" + keyID
	if err := w.manager.Cache.Delete(ctx, cacheKey); err != nil {
		w.logger.WithError(err).Warn("Failed to invalidate API key cache")
		return err
	}

	w.logger.WithField("key_id", keyID).Debug("✅ API key cache invalidated")
	return nil
}

// InvalidateAllAPIKeys invalidates all API key caches
func (w *APIKeyWorker) InvalidateAllAPIKeys(ctx context.Context) error {
	// Pattern: apikey:*
	keys, err := w.manager.Client.Keys(ctx, "apikey:*")
	if err != nil {
		w.logger.WithError(err).Warn("Failed to list API key cache keys")
		return err
	}

	for _, key := range keys {
		if err := w.manager.Cache.Delete(ctx, key); err != nil {
			w.logger.WithError(err).WithField("key", key).Warn("Failed to delete cache key")
		}
	}

	w.logger.WithField("count", len(keys)).Info("✅ All API key caches invalidated")
	return nil
}
