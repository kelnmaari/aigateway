package models

import (
	"encoding/json"
	"time"
)

// ModelProviderType представляет тип провайдера моделей
type ModelProviderType string

const (
	ProviderTypeVLLM      ModelProviderType = "vllm"
	ProviderTypeOpenAI    ModelProviderType = "openai"
	ProviderTypeAnthropic ModelProviderType = "anthropic"
	ProviderTypeGemini    ModelProviderType = "gemini"
	ProviderTypeDeepSeek  ModelProviderType = "deepseek"
	ProviderTypeCustom    ModelProviderType = "custom"
)

// ModelHealthStatus представляет health status модели или провайдера
type ModelHealthStatus string

const (
	HealthStatusHealthy   ModelHealthStatus = "healthy"
	HealthStatusUnhealthy ModelHealthStatus = "unhealthy"
	HealthStatusUnknown   ModelHealthStatus = "unknown"
)

// ModelStatus представляет текущий статус модели
type ModelStatus string

const (
	ModelStatusActive   ModelStatus = "active"
	ModelStatusInactive ModelStatus = "inactive"
	ModelStatusLoading  ModelStatus = "loading"
	ModelStatusError    ModelStatus = "error"
)

// ModelCapability представляет возможности модели
// Capabilities определяют роли модели согласно Continue.dev model roles:
// https://docs.continue.dev/customize/model-roles/00-intro
type ModelCapability string

const (
	// Chat model capabilities (text generation)
	CapabilityChat         ModelCapability = "chat"         // Chat conversations
	CapabilityAutocomplete ModelCapability = "autocomplete" // Code autocomplete suggestions
	CapabilityEdit         ModelCapability = "edit"         // Generate code based on edit prompts
	CapabilityApply        ModelCapability = "apply"        // Apply edits to files
	CapabilityVision       ModelCapability = "vision"       // Vision/image understanding

	// Embedding model capabilities
	CapabilityEmbeddings ModelCapability = "embeddings" // Vector embeddings for semantic search
	CapabilityRerank     ModelCapability = "rerank"     // Rerank vector search results

	// Additional capabilities
	CapabilityFunctionCalling ModelCapability = "function-calling" // Tool/function calling
	CapabilityCodeCompletion  ModelCapability = "code-completion"  // Code completion (legacy, use autocomplete)
)

// ChatModelCapabilities returns capabilities implied by chat capability.
// If a model can chat, it can also autocomplete, edit, and apply.
func ChatModelCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityChat,
		CapabilityAutocomplete,
		CapabilityEdit,
		CapabilityApply,
	}
}

// EmbeddingModelCapabilities returns capabilities implied by embeddings capability.
// Embedding models can also rerank.
func EmbeddingModelCapabilities() []ModelCapability {
	return []ModelCapability{
		CapabilityEmbeddings,
		CapabilityRerank,
	}
}

// ========================================
// Model Provider
// ========================================

// ModelProvider представляет конфигурацию для model provider (vLLM, OpenAI, Anthropic, Gemini, etc)
type ModelProvider struct {
	ID           string            `json:"id" db:"id"`
	Name         string            `json:"name" db:"name"`                   // "openai-main", "vllm-gpu1"
	ProviderType ModelProviderType `json:"provider_type" db:"provider_type"` // "openai", "vllm", "anthropic", "gemini"
	BaseURL      string            `json:"base_url" db:"base_url"`           // "https://api.openai.com"
	APIKey       string            `json:"-" db:"api_key"`                   // Encrypted, не возвращается в API

	// Configuration
	Enabled  bool                   `json:"enabled" db:"enabled"`
	Priority int                    `json:"priority" db:"priority"`       // Higher = preferred
	Config   map[string]interface{} `json:"config" db:"config"`           // Provider-specific config
	ConfigDB string                 `json:"-" db:"config_json,omitempty"` // For DB serialization

	// Health
	HealthStatus    ModelHealthStatus `json:"health_status" db:"health_status"`
	LastHealthCheck *time.Time        `json:"last_health_check,omitempty" db:"last_health_check"`
	ErrorMessage    string            `json:"error_message,omitempty" db:"error_message"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MarshalConfigToDB сериализует Config в JSON string для хранения в БД
func (p *ModelProvider) MarshalConfigToDB() error {
	if p.Config == nil {
		p.ConfigDB = "{}"
		return nil
	}
	data, err := json.Marshal(p.Config)
	if err != nil {
		return err
	}
	p.ConfigDB = string(data)
	return nil
}

// UnmarshalConfigFromDB десериализует Config из JSON string из БД
func (p *ModelProvider) UnmarshalConfigFromDB() error {
	if p.ConfigDB == "" {
		p.Config = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal([]byte(p.ConfigDB), &p.Config)
}

// CreateModelProviderRequest представляет запрос на создание provider
type CreateModelProviderRequest struct {
	Name         string                 `json:"name" binding:"required"`
	ProviderType ModelProviderType      `json:"provider_type" binding:"required"`
	BaseURL      string                 `json:"base_url" binding:"required"`
	APIKey       string                 `json:"api_key,omitempty"`
	Enabled      *bool                  `json:"enabled,omitempty"`
	Priority     *int                   `json:"priority,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// UpdateModelProviderRequest представляет запрос на обновление provider
type UpdateModelProviderRequest struct {
	Name     *string                `json:"name,omitempty"`
	BaseURL  *string                `json:"base_url,omitempty"`
	APIKey   *string                `json:"api_key,omitempty"`
	Enabled  *bool                  `json:"enabled,omitempty"`
	Priority *int                   `json:"priority,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

// ========================================
// Model Registry
// ========================================

// ModelRegistry представляет запись о модели в универсальном реестре
type ModelRegistry struct {
	ID         string `json:"id" db:"id"`
	ModelID    string `json:"model_id" db:"model_id"`       // "llama2:7b", "gpt-4"
	ModelName  string `json:"model_name" db:"model_name"`   // Display name
	ProviderID string `json:"provider_id" db:"provider_id"` // FK to model_providers

	// Capabilities & Parameters
	Capabilities   []ModelCapability      `json:"capabilities" db:"capabilities"`
	CapabilitiesDB string                 `json:"-" db:"capabilities_json,omitempty"` // For DB serialization
	Parameters     map[string]interface{} `json:"parameters" db:"parameters"`
	ParametersDB   string                 `json:"-" db:"parameters_json,omitempty"` // For DB serialization

	// Requirements
	RequiresGPU   bool `json:"requires_gpu" db:"requires_gpu"`
	MinVRAMGB     *int `json:"min_vram_gb,omitempty" db:"min_vram_gb"`
	ContextLength *int `json:"context_length,omitempty" db:"context_length"`

	// Status
	Status          ModelStatus       `json:"status" db:"status"`
	HealthStatus    ModelHealthStatus `json:"health_status" db:"health_status"`
	LastHealthCheck *time.Time        `json:"last_health_check,omitempty" db:"last_health_check"`

	// Metadata
	Description string   `json:"description,omitempty" db:"description"`
	Tags        []string `json:"tags,omitempty" db:"tags"`
	TagsDB      string   `json:"-" db:"tags_string,omitempty"` // For SQLite CSV storage

	// Performance Metrics
	AvgLatencyMs    *float64 `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	TokensPerSecond *float64 `json:"tokens_per_second,omitempty" db:"tokens_per_second"`
	TotalRequests   int64    `json:"total_requests" db:"total_requests"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined data (not in DB)
	Provider *ModelProvider `json:"provider,omitempty" db:"-"`
}

// MarshalToDB сериализует JSON поля для хранения в БД
func (m *ModelRegistry) MarshalToDB() error {
	// Capabilities
	if m.Capabilities == nil {
		m.CapabilitiesDB = "[]"
	} else {
		data, err := json.Marshal(m.Capabilities)
		if err != nil {
			return err
		}
		m.CapabilitiesDB = string(data)
	}

	// Parameters
	if m.Parameters == nil {
		m.ParametersDB = "{}"
	} else {
		data, err := json.Marshal(m.Parameters)
		if err != nil {
			return err
		}
		m.ParametersDB = string(data)
	}

	// Tags (для SQLite - CSV string, для PostgreSQL - array)
	if m.Tags != nil && len(m.Tags) > 0 {
		// SQLite будет использовать TagsDB (comma-separated)
		// PostgreSQL будет использовать Tags (array)
		// Обработка в storage layer
	}

	return nil
}

// UnmarshalFromDB десериализует JSON поля из БД
func (m *ModelRegistry) UnmarshalFromDB() error {
	// Capabilities
	if m.CapabilitiesDB != "" {
		var caps []ModelCapability
		if err := json.Unmarshal([]byte(m.CapabilitiesDB), &caps); err != nil {
			return err
		}
		m.Capabilities = caps
	} else {
		m.Capabilities = []ModelCapability{}
	}

	// Parameters
	if m.ParametersDB != "" {
		var params map[string]interface{}
		if err := json.Unmarshal([]byte(m.ParametersDB), &params); err != nil {
			return err
		}
		m.Parameters = params
	} else {
		m.Parameters = make(map[string]interface{})
	}

	// Tags parsing handled in storage layer (different for SQLite/PostgreSQL)

	return nil
}

// CreateModelRegistryRequest представляет запрос на регистрацию модели
type CreateModelRegistryRequest struct {
	ModelID       string                 `json:"model_id" binding:"required"`
	ModelName     string                 `json:"model_name" binding:"required"`
	ProviderID    string                 `json:"provider_id" binding:"required"`
	Capabilities  []ModelCapability      `json:"capabilities,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	RequiresGPU   *bool                  `json:"requires_gpu,omitempty"`
	MinVRAMGB     *int                   `json:"min_vram_gb,omitempty"`
	ContextLength *int                   `json:"context_length,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
}

// UpdateModelRegistryRequest представляет запрос на обновление модели
type UpdateModelRegistryRequest struct {
	ModelName     *string                `json:"model_name,omitempty"`
	ProviderID    *string                `json:"provider_id,omitempty"`
	Capabilities  []ModelCapability      `json:"capabilities,omitempty"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Status        *ModelStatus           `json:"status,omitempty"`
	RequiresGPU   *bool                  `json:"requires_gpu,omitempty"`
	MinVRAMGB     *int                   `json:"min_vram_gb,omitempty"`
	ContextLength *int                   `json:"context_length,omitempty"`
	Description   *string                `json:"description,omitempty"`
	Tags          []string               `json:"tags,omitempty"`
}

// ModelRegistryFilter представляет фильтры для поиска моделей
type ModelRegistryFilter struct {
	ProviderID   string            // Фильтр по provider
	ProviderType ModelProviderType // Фильтр по типу provider
	Capabilities []ModelCapability // Модели с этими capabilities
	Status       ModelStatus       // Фильтр по статусу
	HealthStatus ModelHealthStatus // Фильтр по health
	RequiresGPU  *bool             // Требуется ли GPU
	Tag          string            // Фильтр по тегу
	Limit        int               // Limit results
	Offset       int               // Pagination offset
}

// ModelRegistryStats представляет статистику registry
type ModelRegistryStats struct {
	TotalModels        int                     `json:"total_models"`
	ActiveModels       int                     `json:"active_models"`
	HealthyModels      int                     `json:"healthy_models"`
	TotalProviders     int                     `json:"total_providers"`
	EnabledProviders   int                     `json:"enabled_providers"`
	ModelsByProvider   map[string]int          `json:"models_by_provider"`   // provider_name -> count
	ModelsByCapability map[ModelCapability]int `json:"models_by_capability"` // capability -> count
}
