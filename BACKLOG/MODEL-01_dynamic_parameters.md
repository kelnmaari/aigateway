# MODEL-01: Dynamic Model Parameters Configuration

**Версия:** 1.10.0  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 8-12 часов

## Описание

UI для настройки параметров моделей (temperature, top_p, num_ctx, и др.) через Admin Panel. Per-tenant и per-user конфигурации с валидацией через Ollama API.

## Проблема

В текущей версии:
- Параметры моделей hardcoded или default
- Невозможно настроить temperature, top_p через UI
- Нет per-tenant/per-user model configs
- Админы не могут ограничить параметры для пользователей

## Решение

UI для динамической настройки model parameters с validation и storage.

### Архитектура

```
Admin/User → Model Config UI → Save to DB
                                    ↓
Chat Request → Load Config → Apply to Ollama API
```

## Технические детали

### Model Parameters

Based on official Ollama API (см. `ollama-lib/`):

**Основные параметры:**
- `temperature` (0.0-2.0) - creativity level
- `top_p` (0.0-1.0) - nucleus sampling
- `top_k` (1-100) - top-k sampling
- `num_ctx` (128-131072) - context window size
- `num_predict` (-1, 1-4096) - max tokens to generate
- `repeat_penalty` (0.0-2.0) - repetition penalty
- `seed` (0-2147483647) - random seed
- `stop` (array of strings) - stop sequences

**Advanced параметры:**
- `num_gpu` (0-N) - number of GPUs to use
- `num_thread` (1-N) - number of CPU threads
- `mirostat` (0, 1, 2) - Mirostat sampling
- `mirostat_tau` (0.0-10.0)
- `mirostat_eta` (0.0-1.0)

### Data Model

```go
// internal/models/model_config.go

type ModelConfig struct {
    ID        string                 `json:"id" db:"id"`
    ModelName string                 `json:"model_name" db:"model_name"` // e.g., "llama3.1:latest"
    Scope     string                 `json:"scope" db:"scope"`           // "global", "tenant", "user"
    TenantID  *string                `json:"tenant_id,omitempty" db:"tenant_id"`
    UserID    *string                `json:"user_id,omitempty" db:"user_id"`
    
    // Model parameters
    Parameters ModelParameters `json:"parameters" db:"parameters"` // JSON
    
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type ModelParameters struct {
    // Generation parameters
    Temperature   *float64 `json:"temperature,omitempty"`
    TopP          *float64 `json:"top_p,omitempty"`
    TopK          *int     `json:"top_k,omitempty"`
    NumCtx        *int     `json:"num_ctx,omitempty"`
    NumPredict    *int     `json:"num_predict,omitempty"`
    RepeatPenalty *float64 `json:"repeat_penalty,omitempty"`
    Seed          *int     `json:"seed,omitempty"`
    Stop          []string `json:"stop,omitempty"`
    
    // Advanced parameters
    NumGPU       *int     `json:"num_gpu,omitempty"`
    NumThread    *int     `json:"num_thread,omitempty"`
    Mirostat     *int     `json:"mirostat,omitempty"`
    MirostatTau  *float64 `json:"mirostat_tau,omitempty"`
    MirostatEta  *float64 `json:"mirostat_eta,omitempty"`
}
```

### Database Schema

```sql
CREATE TABLE model_configs (
    id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'global', -- 'global', 'tenant', 'user'
    tenant_id TEXT,
    user_id TEXT,
    parameters TEXT NOT NULL, -- JSON
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(model_name, tenant_id, user_id) -- One config per model per scope
);

CREATE INDEX idx_model_configs_model_name ON model_configs(model_name);
CREATE INDEX idx_model_configs_tenant_id ON model_configs(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX idx_model_configs_user_id ON model_configs(user_id) WHERE user_id IS NOT NULL;
```

### Config Resolution (Priority)

```
User-specific config → Tenant-specific config → Global config → Ollama defaults
```

```go
// internal/services/model/config.go

type ModelConfigService struct {
    db storage.Database
}

func (s *ModelConfigService) GetEffectiveConfig(
    ctx context.Context,
    modelName string,
    userID *string,
    tenantID *string,
) (*ModelParameters, error) {
    // Priority: user → tenant → global → defaults
    
    // 1. User-specific config
    if userID != nil {
        config, err := s.db.GetModelConfig(ctx, modelName, "user", nil, userID)
        if err == nil {
            return &config.Parameters, nil
        }
    }
    
    // 2. Tenant-specific config
    if tenantID != nil {
        config, err := s.db.GetModelConfig(ctx, modelName, "tenant", tenantID, nil)
        if err == nil {
            return &config.Parameters, nil
        }
    }
    
    // 3. Global config
    config, err := s.db.GetModelConfig(ctx, modelName, "global", nil, nil)
    if err == nil {
        return &config.Parameters, nil
    }
    
    // 4. Return defaults
    return getDefaultParameters(), nil
}

func getDefaultParameters() *ModelParameters {
    return &ModelParameters{
        Temperature: float64Ptr(0.7),
        TopP:        float64Ptr(0.9),
        TopK:        intPtr(40),
        NumCtx:      intPtr(2048),
    }
}
```

### Chat Integration

```go
// internal/api/handlers/chat.go (modify existing)

func (h *ChatHandler) ChatCompletions(c *gin.Context) {
    // ... (existing code)
    
    // Get user and tenant IDs
    userID, _ := c.Get("user_id")
    tenantID, _ := c.Get("tenant_id")
    
    // Get effective model config
    modelConfig, err := h.modelConfigService.GetEffectiveConfig(
        c.Request.Context(),
        req.Model,
        stringPtr(userID.(string)),
        stringPtrOrNil(tenantID),
    )
    if err != nil {
        h.logger.WithError(err).Warn("Failed to get model config, using defaults")
        modelConfig = getDefaultParameters()
    }
    
    // Merge with user-provided parameters (user params override config)
    effectiveParams := mergeParameters(modelConfig, req.Options)
    
    // Validate parameters
    if err := validateParameters(effectiveParams); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    
    // Build Ollama request with effective params
    ollamaReq := &ollama.ChatRequest{
        Model:    req.Model,
        Messages: req.Messages,
        Options:  effectiveParams,
        Stream:   req.Stream,
    }
    
    // ... (send to Ollama)
}

func mergeParameters(config *ModelParameters, userParams map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{})
    
    // Start with config
    if config.Temperature != nil {
        result["temperature"] = *config.Temperature
    }
    // ... (rest of params)
    
    // Override with user params
    for k, v := range userParams {
        result[k] = v
    }
    
    return result
}
```

### Validation Service

```go
// internal/services/model/validator.go

type ParameterValidator struct {
    ollamaClient *ollama.Client
}

func (v *ParameterValidator) Validate(params *ModelParameters) error {
    if params.Temperature != nil {
        if *params.Temperature < 0.0 || *params.Temperature > 2.0 {
            return errors.New("temperature must be between 0.0 and 2.0")
        }
    }
    
    if params.TopP != nil {
        if *params.TopP < 0.0 || *params.TopP > 1.0 {
            return errors.New("top_p must be between 0.0 and 1.0")
        }
    }
    
    if params.NumCtx != nil {
        if *params.NumCtx < 128 || *params.NumCtx > 131072 {
            return errors.New("num_ctx must be between 128 and 131072")
        }
    }
    
    // Validate against Ollama capabilities (optional)
    // if err := v.validateWithOllama(params); err != nil {
    //     return err
    // }
    
    return nil
}
```

### Admin UI

**Admin Panel → Models → Configuration:**

```html
<div id="model-config-tab" class="tab-pane">
  <h3>Model Configuration</h3>
  
  <!-- Model Selector -->
  <select id="config-model-select">
    <option value="llama3.1:latest">llama3.1:latest</option>
    <option value="gpt-oss:latest">gpt-oss:latest</option>
    <!-- ... -->
  </select>
  
  <!-- Scope Selector -->
  <select id="config-scope">
    <option value="global">Global (all users)</option>
    <option value="tenant">Tenant-specific</option>
    <option value="user">User-specific</option>
  </select>
  
  <div id="tenant-selector" style="display:none;">
    <select id="config-tenant"><!-- populated --></select>
  </div>
  
  <div id="user-selector" style="display:none;">
    <select id="config-user"><!-- populated --></select>
  </div>
  
  <!-- Parameter Configuration -->
  <div class="model-parameters">
    <h4>Generation Parameters</h4>
    
    <label>
      Temperature (0.0-2.0)
      <input type="range" id="param-temperature" min="0" max="2" step="0.1" value="0.7" />
      <span id="temperature-value">0.7</span>
    </label>
    
    <label>
      Top P (0.0-1.0)
      <input type="range" id="param-top-p" min="0" max="1" step="0.05" value="0.9" />
      <span id="top-p-value">0.9</span>
    </label>
    
    <label>
      Context Window (num_ctx)
      <input type="number" id="param-num-ctx" min="128" max="131072" value="2048" />
    </label>
    
    <label>
      Max Tokens (num_predict)
      <input type="number" id="param-num-predict" min="-1" max="4096" value="-1" />
      <small>-1 = unlimited</small>
    </label>
    
    <h4>Advanced Parameters</h4>
    
    <label>
      Number of GPUs
      <input type="number" id="param-num-gpu" min="0" max="8" value="1" />
    </label>
    
    <label>
      CPU Threads
      <input type="number" id="param-num-thread" min="1" max="64" value="4" />
    </label>
    
    <!-- ... more params -->
  </div>
  
  <button onclick="saveModelConfig()">Save Configuration</button>
  <button onclick="testModelConfig()">Test Configuration</button>
  <button onclick="resetToDefaults()">Reset to Defaults</button>
</div>
```

### User UI (Profile Settings)

```html
<!-- web/profile.html → Model Preferences -->
<div class="user-model-prefs">
  <h3>My Model Preferences</h3>
  
  <p>Override default model settings for your chat sessions.</p>
  
  <select id="user-pref-model">
    <option value="llama3.1:latest">llama3.1:latest</option>
  </select>
  
  <label>
    Temperature: <input type="range" id="user-temp" min="0" max="2" step="0.1" />
  </label>
  
  <label>
    Max Tokens: <input type="number" id="user-max-tokens" />
  </label>
  
  <button onclick="saveUserModelPrefs()">Save My Preferences</button>
</div>
```

## Требования

### Функциональные

1. ✅ ModelConfig model с parameters
2. ✅ Per-scope configs (global, tenant, user)
3. ✅ Config resolution с priority
4. ✅ Parameter validation
5. ✅ Admin UI для model configuration
6. ✅ User UI для personal preferences
7. ✅ Test configuration function
8. ✅ Reset to defaults
9. ✅ Apply config в chat requests
10. ✅ Templates для popular configs (preset buttons)

### Нефункциональные

1. **Usability**
   - Intuitive sliders для parameters
   - Real-time validation feedback
   - Tooltips с explanations

2. **Performance**
   - Config lookup < 10ms
   - Caching effective configs

## Acceptance Criteria

- [ ] Model configs сохраняются в БД
- [ ] Global, tenant, user scopes работают
- [ ] Config resolution правильно применяет priority
- [ ] Parameter validation блокирует invalid values
- [ ] Admin UI позволяет настроить model parameters
- [ ] User UI позволяет установить personal preferences
- [ ] Test configuration проверяет валидность
- [ ] Chat requests используют effective config
- [ ] User-provided params override config params
- [ ] Reset to defaults восстанавливает default values
- [ ] Unit tests для config resolution, validation
- [ ] Integration tests для chat with custom params

## Риски и зависимости

### Риски

1. **Invalid parameters** - crash Ollama
   - Mitigation: Strict validation, safe defaults

2. **Performance impact** - неоптимальные параметры
   - Mitigation: Sane defaults, warnings для extreme values

### Зависимости

1. Ollama API documentation (см. `ollama-lib/`)
2. Existing chat handler

## Связанные задачи

- **MODEL-02**: Model Preloading - использует model configs
- **QUOTA-01**: Usage Quotas - ограничения на num_ctx, num_predict

## Примечания

- Parameters не все модели поддерживают одинаково
- Validation основана на Ollama API v1
- Advanced params (num_gpu, num_thread) требуют admin rights

---

**Статус:** 📋 Planned for v1.10.0  
**Последнее обновление:** 2025-10-11

