# MODEL-01: Dynamic Model Parameters Configuration

**Версия:** 1.9.1  
**Приоритет:** Medium  
**Сложность:** Medium  
**Оценка:** 10-14 часов (расширенная с Chat UI)

## Описание

Динамическая настройка параметров моделей (temperature, top_p, num_ctx, и др.) с тремя уровнями управления:
1. **Chat UI** (Primary) - красивая панель над строкой ввода для per-chat настроек с localStorage persistence
2. **Admin Panel** - глобальные, tenant, и user конфигурации
3. **Profile** (Secondary) - permanent user preferences

Выбранная модель и параметры запоминаются в localStorage и автоматически восстанавливаются при перезагрузке.

## Проблема

В текущей версии:
- Параметры моделей hardcoded или default
- Невозможно настроить temperature, top_p через UI
- Нет per-tenant/per-user model configs
- Админы не могут ограничить параметры для пользователей
- **Выбор модели расположен неудобно (в верхней части чата)**
- **Нет возможности быстро менять параметры без перезагрузки**
- **Не сохраняется последняя использованная модель**
- **Нет интуитивного UX для настройки параметров в чате**

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

### Chat UI - Model Selector Panel (Primary Interface)

**Расположение:** Над строкой ввода в web/chat.html

```html
<!-- web/chat.html → Model Control Panel (above input) -->
<div class="chat-model-panel">
  
  <!-- Model Selector (красивый dropdown с иконками) -->
  <div class="model-selector-wrapper">
    <label class="model-label">
      <i class="bi bi-robot"></i> Model
    </label>
    <select id="chat-model-select" class="model-select">
      <optgroup label="Recommended">
        <option value="qwen2.5-coder-tuned:7b" data-icon="⚡">Qwen 2.5 Coder (Fast)</option>
        <option value="qwen3-coder-tuned:30b" data-icon="🎯">Qwen 3 Coder (Accurate)</option>
      </optgroup>
      <optgroup label="All Models">
        <option value="devstral-tuned:latest" data-icon="🔬">Devstral</option>
        <!-- Динамически загружается с /v1/models -->
      </optgroup>
    </select>
  </div>
  
  <!-- Parameters Panel (collapsible) -->
  <div class="model-params-toggle">
    <button id="params-toggle" class="btn-params" aria-expanded="false">
      <i class="bi bi-sliders"></i> Parameters
      <i class="bi bi-chevron-down chevron-icon"></i>
    </button>
  </div>
  
  <!-- Expandable Parameters Panel -->
  <div id="params-panel" class="params-collapse" hidden>
    <div class="params-grid">
      
      <!-- Temperature -->
      <div class="param-control">
        <label for="chat-temperature">
          Temperature 
          <span class="param-info" data-tooltip="Creativity level (0.0 = focused, 2.0 = creative)">
            <i class="bi bi-info-circle"></i>
          </span>
        </label>
        <div class="param-slider-group">
          <input type="range" id="chat-temperature" min="0" max="2" step="0.1" value="0.7" />
          <input type="number" id="chat-temperature-value" min="0" max="2" step="0.1" value="0.7" class="param-number" />
        </div>
      </div>
      
      <!-- Top P -->
      <div class="param-control">
        <label for="chat-top-p">
          Top P
          <span class="param-info" data-tooltip="Nucleus sampling (0.0-1.0)">
            <i class="bi bi-info-circle"></i>
          </span>
        </label>
        <div class="param-slider-group">
          <input type="range" id="chat-top-p" min="0" max="1" step="0.05" value="0.9" />
          <input type="number" id="chat-top-p-value" min="0" max="1" step="0.05" value="0.9" class="param-number" />
        </div>
      </div>
      
      <!-- Max Tokens -->
      <div class="param-control">
        <label for="chat-max-tokens">
          Max Tokens
          <span class="param-info" data-tooltip="Maximum response length (-1 = unlimited)">
            <i class="bi bi-info-circle"></i>
          </span>
        </label>
        <input type="number" id="chat-max-tokens" min="-1" max="4096" value="-1" class="param-input" />
      </div>
      
      <!-- Context Window -->
      <div class="param-control">
        <label for="chat-num-ctx">
          Context Window
          <span class="param-info" data-tooltip="Number of context tokens (128-131072)">
            <i class="bi bi-info-circle"></i>
          </span>
        </label>
        <input type="number" id="chat-num-ctx" min="128" max="131072" value="4096" class="param-input" />
      </div>
      
    </div>
    
    <!-- Quick Presets -->
    <div class="params-presets">
      <button class="preset-btn" onclick="applyPreset('creative')">
        <i class="bi bi-palette"></i> Creative
      </button>
      <button class="preset-btn" onclick="applyPreset('balanced')">
        <i class="bi bi-balance-scale"></i> Balanced
      </button>
      <button class="preset-btn" onclick="applyPreset('precise')">
        <i class="bi bi-bullseye"></i> Precise
      </button>
      <button class="preset-btn" onclick="applyPreset('coding')">
        <i class="bi bi-code-slash"></i> Coding
      </button>
    </div>
    
    <!-- Reset Button -->
    <button class="btn-reset" onclick="resetChatParams()">
      <i class="bi bi-arrow-counterclockwise"></i> Reset to Defaults
    </button>
  </div>
  
</div>

<!-- Styles для Chat Model Panel -->
<style>
.chat-model-panel {
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 12px;
  padding: 16px;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.model-selector-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.model-label {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--bs-primary);
}

.model-select {
  flex: 1;
  padding: 10px 16px;
  border-radius: 8px;
  border: 2px solid var(--bs-border-color);
  background: var(--bs-body-bg);
  font-size: 16px;
  transition: all 0.2s;
}

.model-select:focus {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(13, 110, 253, 0.25);
  outline: none;
}

.btn-params {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--bs-secondary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-params:hover {
  background: var(--bs-tertiary-bg);
}

.btn-params[aria-expanded="true"] .chevron-icon {
  transform: rotate(180deg);
}

.params-collapse {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--bs-border-color);
  animation: slideDown 0.3s ease-out;
}

@keyframes slideDown {
  from { opacity: 0; transform: translateY(-10px); }
  to { opacity: 1; transform: translateY(0); }
}

.params-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.param-control {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.param-control label {
  font-weight: 500;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.param-info {
  cursor: help;
  color: var(--bs-secondary);
}

.param-slider-group {
  display: flex;
  gap: 8px;
  align-items: center;
}

.param-slider-group input[type="range"] {
  flex: 1;
}

.param-number, .param-input {
  width: 80px;
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid var(--bs-border-color);
}

.params-presets {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.preset-btn {
  padding: 8px 16px;
  border-radius: 6px;
  border: 1px solid var(--bs-border-color);
  background: var(--bs-body-bg);
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 6px;
}

.preset-btn:hover {
  background: var(--bs-primary);
  color: white;
  border-color: var(--bs-primary);
}

.btn-reset {
  padding: 8px 16px;
  border-radius: 6px;
  border: 1px solid var(--bs-border-color);
  background: var(--bs-secondary-bg);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-reset:hover {
  background: var(--bs-danger);
  color: white;
  border-color: var(--bs-danger);
}
</style>
```

### Chat UI - JavaScript Logic

```javascript
// web/js/chat.js

// Constants
const STORAGE_KEY_MODEL = 'chat_selected_model';
const STORAGE_KEY_PARAMS = 'chat_model_params';

// Presets
const PRESETS = {
  creative: {
    temperature: 1.2,
    top_p: 0.95,
    num_predict: -1,
    num_ctx: 4096
  },
  balanced: {
    temperature: 0.7,
    top_p: 0.9,
    num_predict: -1,
    num_ctx: 4096
  },
  precise: {
    temperature: 0.3,
    top_p: 0.8,
    num_predict: 2048,
    num_ctx: 2048
  },
  coding: {
    temperature: 0.2,
    top_p: 0.95,
    num_predict: 4096,
    num_ctx: 8192
  }
};

// Load saved model and params on page load
document.addEventListener('DOMContentLoaded', () => {
  loadModelsIntoSelector();
  restoreSavedModelAndParams();
  setupParamsPanelListeners();
  setupModelChangeListener();
});

// Load models from API
async function loadModelsIntoSelector() {
  try {
    const response = await fetch('/v1/models', {
      headers: { 'Authorization': `Bearer ${getAuthToken()}` }
    });
    const data = await response.json();
    
    const selector = document.getElementById('chat-model-select');
    const allModelsGroup = selector.querySelector('optgroup[label="All Models"]');
    
    // Clear existing
    allModelsGroup.innerHTML = '';
    
    // Populate with models
    data.data.forEach(model => {
      const option = document.createElement('option');
      option.value = model.id;
      option.textContent = model.id;
      option.setAttribute('data-icon', getModelIcon(model.id));
      allModelsGroup.appendChild(option);
    });
  } catch (error) {
    console.error('Failed to load models:', error);
  }
}

function getModelIcon(modelName) {
  if (modelName.includes('coder')) return '💻';
  if (modelName.includes('vision')) return '👁️';
  if (modelName.includes('llama')) return '🦙';
  return '🤖';
}

// Restore saved model and params from localStorage
function restoreSavedModelAndParams() {
  // Restore model selection
  const savedModel = localStorage.getItem(STORAGE_KEY_MODEL);
  if (savedModel) {
    const modelSelect = document.getElementById('chat-model-select');
    if (modelSelect.querySelector(`option[value="${savedModel}"]`)) {
      modelSelect.value = savedModel;
    }
  }
  
  // Restore parameters
  const savedParams = localStorage.getItem(STORAGE_KEY_PARAMS);
  if (savedParams) {
    const params = JSON.parse(savedParams);
    applyParamsToUI(params);
  }
}

// Setup listeners for params panel
function setupParamsPanelListeners() {
  // Toggle panel
  const toggleBtn = document.getElementById('params-toggle');
  const panel = document.getElementById('params-panel');
  
  toggleBtn.addEventListener('click', () => {
    const isExpanded = toggleBtn.getAttribute('aria-expanded') === 'true';
    toggleBtn.setAttribute('aria-expanded', !isExpanded);
    panel.hidden = isExpanded;
  });
  
  // Sync sliders with number inputs
  syncSliderWithInput('chat-temperature', 'chat-temperature-value');
  syncSliderWithInput('chat-top-p', 'chat-top-p-value');
  
  // Save params on change
  document.querySelectorAll('.param-control input').forEach(input => {
    input.addEventListener('change', saveCurrentParams);
  });
}

function syncSliderWithInput(sliderId, inputId) {
  const slider = document.getElementById(sliderId);
  const input = document.getElementById(inputId);
  
  slider.addEventListener('input', () => {
    input.value = slider.value;
  });
  
  input.addEventListener('input', () => {
    slider.value = input.value;
  });
}

// Setup model change listener
function setupModelChangeListener() {
  const modelSelect = document.getElementById('chat-model-select');
  modelSelect.addEventListener('change', () => {
    localStorage.setItem(STORAGE_KEY_MODEL, modelSelect.value);
    console.log(`Model changed to: ${modelSelect.value}`);
  });
}

// Apply preset
function applyPreset(presetName) {
  const preset = PRESETS[presetName];
  if (!preset) return;
  
  applyParamsToUI(preset);
  saveCurrentParams();
  
  // Visual feedback
  showToast(`Applied "${presetName}" preset`, 'success');
}

function applyParamsToUI(params) {
  if (params.temperature !== undefined) {
    document.getElementById('chat-temperature').value = params.temperature;
    document.getElementById('chat-temperature-value').value = params.temperature;
  }
  if (params.top_p !== undefined) {
    document.getElementById('chat-top-p').value = params.top_p;
    document.getElementById('chat-top-p-value').value = params.top_p;
  }
  if (params.num_predict !== undefined) {
    document.getElementById('chat-max-tokens').value = params.num_predict;
  }
  if (params.num_ctx !== undefined) {
    document.getElementById('chat-num-ctx').value = params.num_ctx;
  }
}

// Save current params to localStorage
function saveCurrentParams() {
  const params = {
    temperature: parseFloat(document.getElementById('chat-temperature').value),
    top_p: parseFloat(document.getElementById('chat-top-p').value),
    num_predict: parseInt(document.getElementById('chat-max-tokens').value),
    num_ctx: parseInt(document.getElementById('chat-num-ctx').value)
  };
  
  localStorage.setItem(STORAGE_KEY_PARAMS, JSON.stringify(params));
}

// Reset to defaults
function resetChatParams() {
  applyParamsToUI(PRESETS.balanced);
  saveCurrentParams();
  showToast('Reset to balanced defaults', 'info');
}

// Get current params for chat request
function getCurrentModelParams() {
  return {
    model: document.getElementById('chat-model-select').value,
    temperature: parseFloat(document.getElementById('chat-temperature').value),
    top_p: parseFloat(document.getElementById('chat-top-p').value),
    num_predict: parseInt(document.getElementById('chat-max-tokens').value),
    num_ctx: parseInt(document.getElementById('chat-num-ctx').value)
  };
}

// Modify sendMessage to include params
async function sendMessage(message) {
  const params = getCurrentModelParams();
  
  const requestBody = {
    model: params.model,
    messages: [...chatHistory, { role: 'user', content: message }],
    temperature: params.temperature,
    top_p: params.top_p,
    max_tokens: params.num_predict === -1 ? null : params.num_predict,
    // Ollama uses 'options' for num_ctx
    options: {
      num_ctx: params.num_ctx
    },
    stream: true
  };
  
  // ... (rest of sendMessage logic)
}
```

### User UI (Profile Settings - Secondary)

```html
<!-- web/profile.html → Model Preferences (для permanent settings) -->
<div class="user-model-prefs">
  <h3>Default Model Preferences</h3>
  
  <p>Set your default model settings. These are overridden by per-chat settings.</p>
  
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
6. ✅ **Chat UI - Model Selector Panel** (Primary UX)
   - Красивый dropdown над строкой ввода
   - Collapsible parameters panel
   - Real-time sliders с синхронизацией
   - Quick presets (Creative, Balanced, Precise, Coding)
   - Tooltips с объяснениями параметров
7. ✅ **localStorage persistence**
   - Запоминание выбранной модели
   - Запоминание параметров
   - Восстановление при перезагрузке страницы
8. ✅ User UI для permanent preferences (Profile)
9. ✅ Test configuration function
10. ✅ Reset to defaults
11. ✅ Apply config в chat requests
12. ✅ Dynamic model loading из API

### Нефункциональные

1. **Usability**
   - Intuitive sliders для parameters
   - Real-time validation feedback
   - Tooltips с explanations

2. **Performance**
   - Config lookup < 10ms
   - Caching effective configs

## Acceptance Criteria

### Backend
- [ ] Model configs сохраняются в БД
- [ ] Global, tenant, user scopes работают
- [ ] Config resolution правильно применяет priority
- [ ] Parameter validation блокирует invalid values
- [ ] Chat requests используют effective config
- [ ] User-provided params override config params
- [ ] Unit tests для config resolution, validation
- [ ] Integration tests для chat with custom params

### Admin UI
- [ ] Admin UI позволяет настроить model parameters
- [ ] Test configuration проверяет валидность
- [ ] Reset to defaults восстанавливает default values

### Chat UI (Primary)
- [ ] Model selector расположен над строкой ввода
- [ ] Dropdown загружает модели динамически из `/v1/models`
- [ ] Parameters panel collapsible/expandable
- [ ] Sliders синхронизируются с number inputs
- [ ] Tooltips объясняют каждый параметр
- [ ] Quick presets применяются одним кликом
- [ ] **Выбранная модель сохраняется в localStorage**
- [ ] **Параметры сохраняются в localStorage**
- [ ] **Модель и параметры восстанавливаются при перезагрузке**
- [ ] Reset button возвращает к balanced preset
- [ ] Chat requests используют выбранную модель и параметры
- [ ] Visual feedback (toast) при изменениях
- [ ] Responsive design (работает на мобильных)

### Profile UI (Secondary)
- [ ] User UI позволяет установить permanent preferences
- [ ] Defaults применяются если нет chat-specific настроек

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
- **localStorage** используется для chat preferences (не требует backend)
- **Chat UI** - primary interface, Profile/Admin - secondary
- Presets подобраны на основе best practices для разных use cases

---

**Статус:** 📋 Planned for v1.9.1 (приоритет перед VISION-01)  
**Последнее обновление:** 2025-10-13

