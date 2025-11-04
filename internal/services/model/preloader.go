// Package model provides model management services
// Version 1.12.1+: Model Preloading & Warming
package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"

	"github.com/sirupsen/logrus"
)

// ModelPreloader управляет preloading и warming моделей
// Устраняет cold start задержку при первом запросе к модели
type ModelPreloader struct {
	client *ollama.Client
	config *config.ModelPreloadConfig
	logger *logrus.Logger

	// loadedModels хранит время последнего использования каждой модели
	loadedModels map[string]time.Time
	mu           sync.RWMutex

	// cancelFunc для остановки health check loop
	cancelFunc context.CancelFunc
}

// NewModelPreloader создает новый ModelPreloader
func NewModelPreloader(
	client *ollama.Client,
	cfg *config.ModelPreloadConfig,
	logger *logrus.Logger,
) *ModelPreloader {
	return &ModelPreloader{
		client:       client,
		config:       cfg,
		logger:       logger,
		loadedModels: make(map[string]time.Time),
	}
}

// Start запускает preloading и health check loop
// Не блокирует выполнение если preload failed
func (p *ModelPreloader) Start(ctx context.Context) error {
	if !p.config.Enabled {
		p.logger.Info("Model preloading disabled")
		return nil
	}

	p.logger.WithFields(logrus.Fields{
		"models":       len(p.config.Models),
		"on_startup":   p.config.OnStartup,
		"keep_warm":    p.config.KeepWarm,
		"unload_after": p.config.UnloadAfter,
	}).Info("Starting model preloader")

	// Preload on startup (async, don't block)
	if p.config.OnStartup && len(p.config.Models) > 0 {
		go func() {
			if err := p.preloadModels(ctx); err != nil {
				p.logger.WithError(err).Warn("Some models failed to preload (non-fatal)")
			}
		}()
	}

	// Start health check loop
	if p.config.KeepWarm && p.config.HealthCheckInterval > 0 {
		loopCtx, cancel := context.WithCancel(ctx)
		p.cancelFunc = cancel
		go p.healthCheckLoop(loopCtx)
	}

	// Start unload loop (optional)
	if p.config.UnloadAfter > 0 {
		go p.unloadUnusedLoop(ctx)
	}

	return nil
}

// Stop останавливает health check loop
func (p *ModelPreloader) Stop() {
	if p.cancelFunc != nil {
		p.cancelFunc()
		p.logger.Info("Model preloader stopped")
	}
}

// preloadModels загружает все configured модели
func (p *ModelPreloader) preloadModels(ctx context.Context) error {
	p.logger.Infof("Preloading %d models", len(p.config.Models))

	var wg sync.WaitGroup
	errCh := make(chan error, len(p.config.Models))

	for _, modelName := range p.config.Models {
		model := modelName // Capture for closure
		wg.Go(func() {
			p.logger.Infof("Preloading model: %s", model)
			start := time.Now()

			if err := p.loadModel(ctx, model); err != nil {
				p.logger.WithError(err).Errorf("Failed to preload model: %s", model)
				errCh <- fmt.Errorf("model %s: %w", model, err)
				return
			}

			p.markLoaded(model)
			p.logger.WithField("duration", time.Since(start)).Infof("Model preloaded: %s", model)
		})
	}

	wg.Wait()
	close(errCh)

	// Collect errors (non-fatal)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%d models failed to preload", len(errs))
	}

	return nil
}

// loadModel загружает модель в память через dummy request
func (p *ModelPreloader) loadModel(ctx context.Context, modelName string) error {
	// Timeout для preload операции (configurable, default 5 minutes для больших моделей)
	timeout := p.config.LoadTimeout
	if timeout == 0 {
		timeout = 5 * time.Minute // Default 5 minutes вместо 60 seconds
	}
	loadCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Send dummy request с minimal token generation
	numPredict := 1
	req := &ollama.GenerateRequest{
		Model:  modelName,
		Prompt: p.config.WarmUpPrompt,
		Stream: false,
		Options: &ollama.GenerateOptions{
			NumPredict: &numPredict, // Generate only 1 token (fast)
		},
	}

	_, err := p.client.Generate(loadCtx, req)
	if err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}

	return nil
}

// healthCheckLoop периодически проверяет загруженные модели
func (p *ModelPreloader) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	p.logger.WithField("interval", p.config.HealthCheckInterval).Info("Model health check loop started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Health check loop stopped")
			return
		case <-ticker.C:
			p.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck проверяет все loaded models
func (p *ModelPreloader) performHealthCheck(ctx context.Context) {
	p.mu.RLock()
	models := make([]string, 0, len(p.loadedModels))
	for model := range p.loadedModels {
		models = append(models, model)
	}
	p.mu.RUnlock()

	if len(models) == 0 {
		return
	}

	p.logger.Debugf("Health check for %d models", len(models))

	for _, modelName := range models {
		checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		
		// Send keep-alive request
		if err := p.loadModel(checkCtx, modelName); err != nil {
			p.logger.WithError(err).Warnf("Health check failed for model: %s", modelName)
			p.markUnloaded(modelName)
		} else {
			p.markLoaded(modelName) // Update last used time
			p.logger.Debugf("Health check passed: %s", modelName)
		}

		cancel()
	}
}

// unloadUnusedLoop периодически выгружает неиспользуемые модели
func (p *ModelPreloader) unloadUnusedLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	p.logger.WithField("unload_after", p.config.UnloadAfter).Info("Model unload loop started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.unloadUnused()
		}
	}
}

// unloadUnused выгружает модели которые не использовались
func (p *ModelPreloader) unloadUnused() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	unloadedCount := 0

	for modelName, lastUsed := range p.loadedModels {
		if now.Sub(lastUsed) > p.config.UnloadAfter {
			p.logger.Infof("Unloading unused model: %s (last used: %s ago)", 
				modelName, now.Sub(lastUsed))
			
			// Ollama doesn't have explicit unload API
			// Model will be evicted by LRU when memory is needed
			delete(p.loadedModels, modelName)
			unloadedCount++
		}
	}

	if unloadedCount > 0 {
		p.logger.Infof("Unloaded %d unused models", unloadedCount)
	}
}

// markLoaded обновляет timestamp использования модели
func (p *ModelPreloader) markLoaded(modelName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.loadedModels[modelName] = time.Now()
}

// markUnloaded удаляет модель из tracked list
func (p *ModelPreloader) markUnloaded(modelName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.loadedModels, modelName)
}

// MarkUsed вызывается при каждом запросе к модели
// Обновляет timestamp для отслеживания активности
func (p *ModelPreloader) MarkUsed(modelName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if _, exists := p.loadedModels[modelName]; exists {
		p.loadedModels[modelName] = time.Now()
		p.logger.Debugf("Model marked as used: %s", modelName)
	}
}

// GetLoadedModels возвращает список loaded models
func (p *ModelPreloader) GetLoadedModels() []ModelStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	models := make([]ModelStatus, 0, len(p.loadedModels))
	now := time.Now()

	for model, lastUsed := range p.loadedModels {
		models = append(models, ModelStatus{
			Name:      model,
			Loaded:    true,
			LastUsed:  lastUsed,
			IdleTime:  now.Sub(lastUsed),
		})
	}

	return models
}

// PreloadModel вручную загружает модель (через API)
func (p *ModelPreloader) PreloadModel(ctx context.Context, modelName string) error {
	p.logger.Infof("Manual preload requested for model: %s", modelName)

	if err := p.loadModel(ctx, modelName); err != nil {
		return fmt.Errorf("failed to preload model: %w", err)
	}

	p.markLoaded(modelName)
	p.logger.Infof("Model manually preloaded: %s", modelName)

	return nil
}

// ModelStatus представляет статус модели
type ModelStatus struct {
	Name     string        `json:"name"`
	Loaded   bool          `json:"loaded"`
	LastUsed time.Time     `json:"last_used"`
	IdleTime time.Duration `json:"idle_time"`
}


