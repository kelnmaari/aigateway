# REGISTRY-01: Model Registry Core

**Version:** 2.2.1  
**Priority:** HIGH  
**Estimated Time:** 6-8 hours  
**Status:** 📋 Planned  
**Dependencies:** None (Foundation task)

---

## 🎯 Goal

Create a universal Model Registry system for centralized management of models across multiple providers (Ollama, vLLM, future: OpenAI, Anthropic).

---

## 📋 Requirements

### Functional
- Database schema для model metadata storage
- Model capabilities tracking (chat/embeddings/vision/function-calling)
- Provider abstraction layer
- Model registration API
- Model discovery endpoint
- Health status integration
- Auto-discovery для Ollama/vLLM models

### Non-Functional
- < 10ms latency для registry lookups
- Support для 1000+ models
- Concurrent-safe registry updates
- Graceful degradation если provider unavailable

---

## 🗄️ Database Schema

```sql
-- Model Registry Table
CREATE TABLE model_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Model identification
    model_id TEXT UNIQUE NOT NULL,          -- "llama2:7b", "mistral-7b-instruct"
    model_name TEXT NOT NULL,               -- Display name
    provider TEXT NOT NULL,                 -- "ollama", "vllm", "openai"
    provider_endpoint TEXT,                 -- "http://localhost:11434"
    
    -- Capabilities
    capabilities JSONB DEFAULT '[]',        -- ["chat", "embeddings", "vision"]
    parameters JSONB DEFAULT '{}',          -- Model-specific parameters
    
    -- Requirements
    requires_gpu BOOLEAN DEFAULT false,
    min_vram_gb INTEGER,
    context_length INTEGER,
    
    -- Status
    status TEXT DEFAULT 'active',           -- active, inactive, loading, error
    health_status TEXT DEFAULT 'unknown',   -- healthy, unhealthy, unknown
    last_health_check TIMESTAMP,
    
    -- Metadata
    description TEXT,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Performance metrics (optional)
    avg_latency_ms REAL,
    tokens_per_second REAL,
    total_requests BIGINT DEFAULT 0,
    
    CONSTRAINT valid_provider CHECK (provider IN ('ollama', 'vllm', 'openai', 'anthropic', 'custom'))
);

CREATE INDEX idx_model_registry_provider ON model_registry(provider);
CREATE INDEX idx_model_registry_status ON model_registry(status);
CREATE INDEX idx_model_registry_capabilities ON model_registry USING GIN (capabilities);

-- Model Provider Configuration
CREATE TABLE model_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,              -- "ollama", "vllm-primary"
    provider_type TEXT NOT NULL,            -- "ollama", "vllm"
    base_url TEXT NOT NULL,
    api_key TEXT,                           -- Encrypted
    
    -- Configuration
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100,           -- Higher = preferred
    config JSONB DEFAULT '{}',              -- Provider-specific config
    
    -- Health
    health_status TEXT DEFAULT 'unknown',
    last_health_check TIMESTAMP,
    error_message TEXT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

---

## 🏗️ Go Implementation

### Models

```go
// internal/models/registry/model.go
package registry

import (
    "time"
    "github.com/google/uuid"
)

type ModelProvider string

const (
    ProviderOllama    ModelProvider = "ollama"
    ProviderVLLM      ModelProvider = "vllm"
    ProviderOpenAI    ModelProvider = "openai"
    ProviderAnthropic ModelProvider = "anthropic"
    ProviderCustom    ModelProvider = "custom"
)

type ModelCapability string

const (
    CapabilityChat          ModelCapability = "chat"
    CapabilityEmbeddings    ModelCapability = "embeddings"
    CapabilityVision        ModelCapability = "vision"
    CapabilityFunctionCall  ModelCapability = "function-calling"
    CapabilityStreaming     ModelCapability = "streaming"
)

type ModelStatus string

const (
    ModelStatusActive   ModelStatus = "active"
    ModelStatusInactive ModelStatus = "inactive"
    ModelStatusLoading  ModelStatus = "loading"
    ModelStatusError    ModelStatus = "error"
)

type ModelInfo struct {
    ID               uuid.UUID         `json:"id" db:"id"`
    ModelID          string            `json:"model_id" db:"model_id"`
    ModelName        string            `json:"model_name" db:"model_name"`
    Provider         ModelProvider     `json:"provider" db:"provider"`
    ProviderEndpoint string            `json:"provider_endpoint" db:"provider_endpoint"`
    
    Capabilities     []ModelCapability `json:"capabilities" db:"capabilities"`
    Parameters       map[string]interface{} `json:"parameters" db:"parameters"`
    
    RequiresGPU      bool              `json:"requires_gpu" db:"requires_gpu"`
    MinVRAMGB        int               `json:"min_vram_gb,omitempty" db:"min_vram_gb"`
    ContextLength    int               `json:"context_length,omitempty" db:"context_length"`
    
    Status           ModelStatus       `json:"status" db:"status"`
    HealthStatus     string            `json:"health_status" db:"health_status"`
    LastHealthCheck  *time.Time        `json:"last_health_check,omitempty" db:"last_health_check"`
    
    Description      string            `json:"description,omitempty" db:"description"`
    Tags             []string          `json:"tags,omitempty" db:"tags"`
    
    CreatedAt        time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
    
    // Performance metrics
    AvgLatencyMS     float64           `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
    TokensPerSecond  float64           `json:"tokens_per_second,omitempty" db:"tokens_per_second"`
    TotalRequests    int64             `json:"total_requests" db:"total_requests"`
}

type ProviderConfig struct {
    ID              uuid.UUID         `json:"id" db:"id"`
    Name            string            `json:"name" db:"name"`
    ProviderType    ModelProvider     `json:"provider_type" db:"provider_type"`
    BaseURL         string            `json:"base_url" db:"base_url"`
    APIKey          string            `json:"-" db:"api_key"` // Never expose in JSON
    
    Enabled         bool              `json:"enabled" db:"enabled"`
    Priority        int               `json:"priority" db:"priority"`
    Config          map[string]interface{} `json:"config" db:"config"`
    
    HealthStatus    string            `json:"health_status" db:"health_status"`
    LastHealthCheck *time.Time        `json:"last_health_check,omitempty" db:"last_health_check"`
    ErrorMessage    string            `json:"error_message,omitempty" db:"error_message"`
    
    CreatedAt       time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}
```

### Registry Service

```go
// internal/services/registry/service.go
package registry

import (
    "context"
    "fmt"
    "aigateway/internal/models/registry"
    "aigateway/internal/storage"
)

type ModelRegistryService struct {
    db     storage.Database
    logger *logrus.Logger
}

func NewModelRegistryService(db storage.Database, logger *logrus.Logger) *ModelRegistryService {
    return &ModelRegistryService{
        db:     db,
        logger: logger,
    }
}

// RegisterModel adds or updates a model в registry
func (s *ModelRegistryService) RegisterModel(ctx context.Context, model *registry.ModelInfo) error {
    // Check if model already exists
    existing, err := s.GetModel(ctx, model.ModelID)
    if err == nil {
        // Update existing
        model.ID = existing.ID
        return s.UpdateModel(ctx, model)
    }
    
    // Create new
    return s.db.CreateModelRegistryEntry(ctx, model)
}

// GetModel retrieves model info by model_id
func (s *ModelRegistryService) GetModel(ctx context.Context, modelID string) (*registry.ModelInfo, error) {
    return s.db.GetModelRegistryEntry(ctx, modelID)
}

// ListModels returns all models with optional filtering
func (s *ModelRegistryService) ListModels(ctx context.Context, filter ModelFilter) ([]*registry.ModelInfo, error) {
    return s.db.ListModelRegistryEntries(ctx, filter)
}

// UpdateModel updates model info
func (s *ModelRegistryService) UpdateModel(ctx context.Context, model *registry.ModelInfo) error {
    return s.db.UpdateModelRegistryEntry(ctx, model)
}

// DeleteModel removes model from registry
func (s *ModelRegistryService) DeleteModel(ctx context.Context, modelID string) error {
    return s.db.DeleteModelRegistryEntry(ctx, modelID)
}

// DiscoverModels auto-discovers models from providers
func (s *ModelRegistryService) DiscoverModels(ctx context.Context, provider registry.ModelProvider) error {
    // Implementation depends on provider
    switch provider {
    case registry.ProviderOllama:
        return s.discoverOllamaModels(ctx)
    case registry.ProviderVLLM:
        return s.discoverVLLMModels(ctx)
    default:
        return fmt.Errorf("unsupported provider: %s", provider)
    }
}

// UpdateHealthStatus updates model health status
func (s *ModelRegistryService) UpdateHealthStatus(ctx context.Context, modelID string, status string) error {
    return s.db.UpdateModelHealthStatus(ctx, modelID, status)
}

type ModelFilter struct {
    Provider     *registry.ModelProvider
    Capabilities []registry.ModelCapability
    Status       *registry.ModelStatus
    Tags         []string
    RequiresGPU  *bool
}
```

---

## 🌐 API Endpoints

### POST /api/models/registry

Register a new model.

```go
// internal/api/handlers/model_registry.go
func (h *ModelRegistryHandler) RegisterModel(c *gin.Context) {
    var req registry.ModelInfo
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }
    
    // Validate
    if req.ModelID == "" || req.Provider == "" {
        c.JSON(400, gin.H{"error": "model_id and provider required"})
        return
    }
    
    // Register
    if err := h.service.RegisterModel(c.Request.Context(), &req); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, req)
}
```

### GET /api/models/registry

List all registered models.

```go
func (h *ModelRegistryHandler) ListModels(c *gin.Context) {
    // Parse filters
    filter := registry.ModelFilter{
        Provider: parseProvider(c.Query("provider")),
        Status:   parseStatus(c.Query("status")),
    }
    
    models, err := h.service.ListModels(c.Request.Context(), filter)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "models": models,
        "count":  len(models),
    })
}
```

### GET /api/models/registry/:model_id

Get specific model info.

### PUT /api/models/registry/:model_id

Update model info.

### DELETE /api/models/registry/:model_id

Unregister model.

### POST /api/models/registry/discover

Auto-discover models from providers.

```go
func (h *ModelRegistryHandler) DiscoverModels(c *gin.Context) {
    var req struct {
        Provider registry.ModelProvider `json:"provider"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "provider required"})
        return
    }
    
    if err := h.service.DiscoverModels(c.Request.Context(), req.Provider); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "discovery completed"})
}
```

---

## ✅ Acceptance Criteria

- [ ] Database schema created and migrated
- [ ] ModelRegistryService implemented with full CRUD
- [ ] API endpoints functional and tested
- [ ] Auto-discovery works for Ollama models
- [ ] Auto-discovery works for vLLM models
- [ ] Health status tracking implemented
- [ ] Performance metrics collection works
- [ ] Concurrent access safe (no race conditions)

---

## 🧪 Testing

```go
// internal/services/registry/service_test.go
func TestRegisterModel(t *testing.T) {
    service := setupTestService(t)
    
    model := &registry.ModelInfo{
        ModelID:      "llama2:7b",
        ModelName:    "Llama 2 7B",
        Provider:     registry.ProviderOllama,
        Capabilities: []registry.ModelCapability{registry.CapabilityChat},
    }
    
    err := service.RegisterModel(context.Background(), model)
    assert.NoError(t, err)
    
    // Verify registration
    retrieved, err := service.GetModel(context.Background(), "llama2:7b")
    assert.NoError(t, err)
    assert.Equal(t, "Llama 2 7B", retrieved.ModelName)
}
```

---

**Next Steps:**
- REGISTRY-02: Smart Router integration
- VLLM-01: vLLM provider implementation
- REGISTRY-03: WebUI для registry management


