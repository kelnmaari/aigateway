# MODEL-02: Model Preloading & Warming

**Версия:** 1.10.0  
**Приоритет:** Medium  
**Сложность:** Low-Medium  
**Оценка:** 6-8 часов

## Описание

Механизм preloading и warming моделей для ускорения первого ответа. Модели загружаются в память заранее и остаются в горячем состоянии для быстрых ответов.

## Проблема

В текущей версии:

- Первый запрос к модели медленный (cold start)
- Модели выгружаются из памяти при неиспользовании
- Задержка 5-30 секунд при загрузке больших моделей
- Плохой UX для первого пользователя

## Решение

Model preloading strategy с health checks и keep-alive.

### Архитектура

```
Startup → [Preload Models] → Keep in Memory
              ↓
         Periodic Health Check
              ↓
         Warm-up Requests (optional)
```

## Технические детали

### Configuration

```yaml
# config.yaml
ollama:
  models:
    preload:
      enabled: true
      models:
        - "llama3.1:latest"
        - "gpt-oss:latest"
      
      # Preload strategy
      on_startup: true
      keep_warm: true
      
      # Health check
      health_check_interval: 5m
      warm_up_prompt: "Hello" # Test prompt to keep model loaded
      
      # Memory management
      max_loaded_models: 3
      unload_after: 30m # Unload if unused
```

### Preload Service

```go
// internal/services/model/preloader.go

type ModelPreloader struct {
    ollamaClient *ollama.Client
    config       *config.ModelPreloadConfig
    logger       *logrus.Logger
    
    loadedModels map[string]time.Time // model -> last_used
    mu           sync.RWMutex
}

func NewModelPreloader(client *ollama.Client, cfg *config.ModelPreloadConfig, logger *logrus.Logger) *ModelPreloader {
    return &ModelPreloader{
        ollamaClient: client,
        config:       cfg,
        logger:       logger,
        loadedModels: make(map[string]time.Time),
    }
}

func (p *ModelPreloader) Start(ctx context.Context) error {
    if !p.config.Enabled {
        p.logger.Info("Model preloading disabled")
        return nil
    }
    
    // Preload on startup
    if p.config.OnStartup {
        if err := p.preloadModels(ctx); err != nil {
            p.logger.WithError(err).Warn("Failed to preload some models")
        }
    }
    
    // Start health check loop
    if p.config.KeepWarm {
        go p.healthCheckLoop(ctx)
    }
    
    // Start unload loop (optional)
    if p.config.UnloadAfter > 0 {
        go p.unloadUnusedLoop(ctx)
    }
    
    return nil
}

func (p *ModelPreloader) preloadModels(ctx context.Context) error {
    p.logger.Infof("Preloading %d models", len(p.config.Models))
    
    for _, modelName := range p.config.Models {
        p.logger.Infof("Preloading model: %s", modelName)
        
        if err := p.loadModel(ctx, modelName); err != nil {
            p.logger.WithError(err).Errorf("Failed to preload model: %s", modelName)
            continue
        }
        
        p.markLoaded(modelName)
        p.logger.Infof("Model preloaded: %s", modelName)
    }
    
    return nil
}

func (p *ModelPreloader) loadModel(ctx context.Context, modelName string) error {
    // Send a dummy request to load model into memory
    req := &ollama.GenerateRequest{
        Model:  modelName,
        Prompt: p.config.WarmUpPrompt,
        Stream: false,
        Options: map[string]interface{}{
            "num_predict": 1, // Only generate 1 token (fast)
        },
    }
    
    _, err := p.ollamaClient.Generate(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to load model: %w", err)
    }
    
    return nil
}

func (p *ModelPreloader) healthCheckLoop(ctx context.Context) {
    ticker := time.NewTicker(p.config.HealthCheckInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.performHealthCheck(ctx)
        }
    }
}

func (p *ModelPreloader) performHealthCheck(ctx context.Context) {
    p.mu.RLock()
    models := make([]string, 0, len(p.loadedModels))
    for model := range p.loadedModels {
        models = append(models, model)
    }
    p.mu.RUnlock()
    
    for _, modelName := range models {
        p.logger.Debugf("Health check for model: %s", modelName)
        
        // Send keep-alive request
        if err := p.loadModel(ctx, modelName); err != nil {
            p.logger.WithError(err).Warnf("Health check failed for model: %s", modelName)
            p.markUnloaded(modelName)
        } else {
            p.markLoaded(modelName)
        }
    }
}

func (p *ModelPreloader) unloadUnusedLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.unloadUnused(ctx)
        }
    }
}

func (p *ModelPreloader) unloadUnused(ctx context.Context) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    now := time.Now()
    
    for modelName, lastUsed := range p.loadedModels {
        if now.Sub(lastUsed) > p.config.UnloadAfter {
            p.logger.Infof("Unloading unused model: %s (last used: %s ago)", modelName, now.Sub(lastUsed))
            
            // Ollama doesn't have explicit unload API, model will be evicted by LRU
            delete(p.loadedModels, modelName)
        }
    }
}

func (p *ModelPreloader) markLoaded(modelName string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    p.loadedModels[modelName] = time.Now()
}

func (p *ModelPreloader) markUnloaded(modelName string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    delete(p.loadedModels, modelName)
}

func (p *ModelPreloader) MarkUsed(modelName string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    if _, exists := p.loadedModels[modelName]; exists {
        p.loadedModels[modelName] = time.Now()
    }
}

func (p *ModelPreloader) GetLoadedModels() []string {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    models := make([]string, 0, len(p.loadedModels))
    for model := range p.loadedModels {
        models = append(models, model)
    }
    return models
}
```

### Integration with Chat Handler

```go
// internal/api/handlers/chat.go (update)

func (h *ChatHandler) ChatCompletions(c *gin.Context) {
    // ... (existing code)
    
    // Mark model as used for preloading tracking
    if h.modelPreloader != nil {
        h.modelPreloader.MarkUsed(req.Model)
    }
    
    // ... (continue with request)
}
```

### Admin UI

**Admin Panel → Models → Preloading Status:**

```html
<div id="model-preloading-tab" class="tab-pane">
  <h3>Model Preloading</h3>
  
  <label>
    <input type="checkbox" id="preload-enabled" />
    Enable Model Preloading
  </label>
  
  <h4>Currently Loaded Models</h4>
  <ul id="loaded-models-list">
    <li>llama3.1:latest <span class="badge badge-success">Loaded</span> (last used: 2 minutes ago)</li>
    <li>gpt-oss:latest <span class="badge badge-success">Loaded</span> (last used: 5 minutes ago)</li>
  </ul>
  
  <h4>Preload Configuration</h4>
  <div id="preload-models-list">
    <div class="preload-model-item">
      <input type="checkbox" checked /> llama3.1:latest
      <button onclick="preloadNow('llama3.1:latest')">Preload Now</button>
    </div>
    <div class="preload-model-item">
      <input type="checkbox" checked /> gpt-oss:latest
      <button onclick="preloadNow('gpt-oss:latest')">Preload Now</button>
    </div>
  </div>
  
  <button onclick="addPreloadModel()">Add Model to Preload</button>
  <button onclick="savePreloadConfig()">Save Configuration</button>
  <button onclick="preloadAllModels()">Preload All Now</button>
</div>
```

### API Endpoints

```go
// internal/api/handlers/model.go (extend)

// Get loaded models status
// GET /api/admin/models/loaded
func (h *ModelHandler) GetLoadedModels(c *gin.Context) {
    if h.preloader == nil {
        c.JSON(http.StatusOK, gin.H{"loaded_models": []string{}})
        return
    }
    
    models := h.preloader.GetLoadedModels()
    
    c.JSON(http.StatusOK, gin.H{
        "loaded_models": models,
        "count":         len(models),
    })
}

// Preload a model
// POST /api/admin/models/:name/preload
func (h *ModelHandler) PreloadModel(c *gin.Context) {
    modelName := c.Param("name")
    
    if h.preloader == nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "preloader not configured"})
        return
    }
    
    if err := h.preloader.loadModel(c.Request.Context(), modelName); err != nil {
        h.logger.WithError(err).Errorf("Failed to preload model: %s", modelName)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to preload model"})
        return
    }
    
    h.preloader.markLoaded(modelName)
    
    c.JSON(http.StatusOK, gin.H{
        "message": "model preloaded",
        "model":   modelName,
    })
}
```

### Startup Integration

```go
// cmd/server/main.go (update)

func main() {
    // ... (existing setup)
    
    // Initialize model preloader
    modelPreloader := model.NewModelPreloader(
        ollamaClient,
        cfg.Ollama.Models.Preload,
        logger,
    )
    
    // Start preloading
    if err := modelPreloader.Start(context.Background()); err != nil {
        logger.WithError(err).Fatal("Failed to start model preloader")
    }
    
    logger.Info("Model preloader started")
    
    // ... (continue with server)
}
```

## Требования

### Функциональные

1. ✅ Preload models on startup
2. ✅ Health check loop для keep-alive
3. ✅ Warm-up requests (dummy prompts)
4. ✅ Track loaded models
5. ✅ Unload unused models (optional)
6. ✅ Manual preload via Admin UI
7. ✅ Configuration через YAML
8. ✅ API endpoints для status/control
9. ✅ Integration с chat handler
10. ✅ Logging для preload events

### Нефункциональные

1. **Performance**
   - First request latency reduction (5-30s → <1s)
   - Minimal memory overhead

2. **Reliability**
   - Graceful handling если preload fails
   - Don't block startup если preload fails

## Acceptance Criteria

- [ ] Models preload on startup
- [ ] Loaded models respond faster (first request)
- [ ] Health check keeps models warm
- [ ] Unused models unload after configured time
- [ ] Admin UI отображает loaded models status
- [ ] Manual preload функция работает
- [ ] API endpoints для status/preload работают
- [ ] Configuration через YAML применяется
- [ ] Logging отражает preload events
- [ ] Server startup не блокируется при preload failure
- [ ] Unit tests для preloader logic
- [ ] Integration tests для preload flow

## Риски и зависимости

### Риски

1. **Memory exhaustion** - слишком много models
   - Mitigation: max_loaded_models limit, LRU eviction

2. **Startup delay** - долгий preload
   - Mitigation: Async preload, don't block startup

### Зависимости

1. Ollama client
2. Model list from config or DB

## Связанные задачи

- **MODEL-01**: Dynamic Parameters - использует configs для preload
- **METRICS-01**: Prometheus - экспорт preload metrics

## Примечания

- Ollama автоматически управляет памятью (LRU eviction)
- Explicit unload API отсутствует в Ollama
- Preloading полезен для production, но необязателен для dev

---

**Статус:** 📋 Planned for v1.10.0  
**Последнее обновление:** 2025-10-11
