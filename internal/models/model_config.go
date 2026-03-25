// Package models provides data models for the application
package models

import (
	"encoding/json"
	"time"
)

// ModelConfig представляет конфигурацию параметров модели
type ModelConfig struct {
	ID        string    `json:"id" db:"id"`
	ModelName string    `json:"model_name" db:"model_name"` // e.g., "llama3.1:latest"
	Scope     string    `json:"scope" db:"scope"`           // "global", "tenant", "user"
	TenantID  *string   `json:"tenant_id,omitempty" db:"tenant_id"`
	UserID    *string   `json:"user_id,omitempty" db:"user_id"`
	CreatedBy string    `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Parameters stored as JSON in DB
	Parameters ModelParameters `json:"parameters" db:"parameters"`
}

// ModelParameters представляет параметры модели для inference
type ModelParameters struct {
	// ========================================
	// Predict Options (Runtime) - api.Options
	// ========================================

	// NumKeep - number of tokens to keep from initial prompt
	NumKeep *int `json:"num_keep,omitempty"`

	// Seed - random seed for generation (-1 for random)
	Seed *int `json:"seed,omitempty"`

	// NumPredict - maximum number of tokens to predict (-1 for unlimited)
	NumPredict *int `json:"num_predict,omitempty"`

	// TopK - top-k sampling (1-100, 0 = disabled)
	TopK *int `json:"top_k,omitempty"`

	// TopP - nucleus sampling probability (0.0-1.0)
	TopP *float32 `json:"top_p,omitempty"`

	// MinP - minimum probability threshold (0.0-1.0)
	MinP *float32 `json:"min_p,omitempty"`

	// TypicalP - typical probability (0.0-1.0)
	TypicalP *float32 `json:"typical_p,omitempty"`

	// RepeatLastN - number of tokens to consider for repeat penalty
	RepeatLastN *int `json:"repeat_last_n,omitempty"`

	// Temperature - sampling temperature (0.0-2.0)
	// 0.0 = deterministic, higher = more creative
	Temperature *float32 `json:"temperature,omitempty"`

	// RepeatPenalty - penalty for repeated tokens (0.0-2.0)
	RepeatPenalty *float32 `json:"repeat_penalty,omitempty"`

	// PresencePenalty - penalty for token presence (-2.0 to 2.0)
	PresencePenalty *float32 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty - penalty for token frequency (-2.0 to 2.0)
	FrequencyPenalty *float32 `json:"frequency_penalty,omitempty"`

	// Stop - stop sequences for generation
	Stop []string `json:"stop,omitempty"`

	// ========================================
	// Runner Options (Model Load Time) - api.Runner
	// ========================================

	// NumCtx - context window size (128-131072)
	NumCtx *int `json:"num_ctx,omitempty"`

	// NumBatch - batch size for prompt processing
	NumBatch *int `json:"num_batch,omitempty"`

	// NumGPU - number of layers to offload to GPU (-1 = all, 0 = none)
	NumGPU *int `json:"num_gpu,omitempty"`

	// MainGPU - main GPU index for split models
	MainGPU *int `json:"main_gpu,omitempty"`

	// UseMMap - use memory mapping for model loading
	UseMMap *bool `json:"use_mmap,omitempty"`

	// NumThread - number of CPU threads to use
	NumThread *int `json:"num_thread,omitempty"`
}

// DefaultParameters возвращает значения по умолчанию для параметров модели
func DefaultParameters() ModelParameters {
	return ModelParameters{
		// Predict options defaults
		Temperature:      Float32Ptr(0.7),
		TopP:             Float32Ptr(0.9),
		TopK:             new(40),
		NumPredict:       new(-1), // unlimited
		RepeatPenalty:    Float32Ptr(1.1),
		PresencePenalty:  Float32Ptr(0.0),
		FrequencyPenalty: Float32Ptr(0.0),

		// Runner options defaults
		NumCtx: new(4096),
	}
}

// PresetCreative возвращает preset для творческой генерации
func PresetCreative() ModelParameters {
	return ModelParameters{
		Temperature:   Float32Ptr(1.2),
		TopP:          Float32Ptr(0.95),
		TopK:          new(50),
		NumPredict:    new(-1),
		RepeatPenalty: Float32Ptr(1.0),
		NumCtx:        new(4096),
	}
}

// PresetBalanced возвращает сбалансированный preset (default)
func PresetBalanced() ModelParameters {
	return DefaultParameters()
}

// PresetPrecise возвращает preset для точной генерации
func PresetPrecise() ModelParameters {
	return ModelParameters{
		Temperature:   Float32Ptr(0.3),
		TopP:          Float32Ptr(0.8),
		TopK:          new(20),
		NumPredict:    new(2048),
		RepeatPenalty: Float32Ptr(1.15),
		NumCtx:        new(2048),
	}
}

// PresetCoding возвращает preset для генерации кода
func PresetCoding() ModelParameters {
	return ModelParameters{
		Temperature:   Float32Ptr(0.2),
		TopP:          Float32Ptr(0.95),
		TopK:          new(40),
		NumPredict:    new(4096),
		RepeatPenalty: Float32Ptr(1.05),
		NumCtx:        new(8192),
	}
}

// MergeWith объединяет параметры с приоритетом (override имеет приоритет)
func (p *ModelParameters) MergeWith(override *ModelParameters) *ModelParameters {
	if override == nil {
		return p
	}

	merged := *p // copy

	// Predict options
	if override.NumKeep != nil {
		merged.NumKeep = override.NumKeep
	}
	if override.Seed != nil {
		merged.Seed = override.Seed
	}
	if override.NumPredict != nil {
		merged.NumPredict = override.NumPredict
	}
	if override.TopK != nil {
		merged.TopK = override.TopK
	}
	if override.TopP != nil {
		merged.TopP = override.TopP
	}
	if override.MinP != nil {
		merged.MinP = override.MinP
	}
	if override.TypicalP != nil {
		merged.TypicalP = override.TypicalP
	}
	if override.RepeatLastN != nil {
		merged.RepeatLastN = override.RepeatLastN
	}
	if override.Temperature != nil {
		merged.Temperature = override.Temperature
	}
	if override.RepeatPenalty != nil {
		merged.RepeatPenalty = override.RepeatPenalty
	}
	if override.PresencePenalty != nil {
		merged.PresencePenalty = override.PresencePenalty
	}
	if override.FrequencyPenalty != nil {
		merged.FrequencyPenalty = override.FrequencyPenalty
	}
	if len(override.Stop) > 0 {
		merged.Stop = override.Stop
	}

	// Runner options
	if override.NumCtx != nil {
		merged.NumCtx = override.NumCtx
	}
	if override.NumBatch != nil {
		merged.NumBatch = override.NumBatch
	}
	if override.NumGPU != nil {
		merged.NumGPU = override.NumGPU
	}
	if override.MainGPU != nil {
		merged.MainGPU = override.MainGPU
	}
	if override.UseMMap != nil {
		merged.UseMMap = override.UseMMap
	}
	if override.NumThread != nil {
		merged.NumThread = override.NumThread
	}

	return &merged
}

// Scan реализует sql.Scanner для чтения JSON из БД
func (p *ModelParameters) Scan(value any) error {
	if value == nil {
		*p = DefaultParameters()
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		bytes = []byte(value.(string))
	}

	return json.Unmarshal(bytes, p)
}

// Helper functions для создания указателей
//
//go:fix inline
func IntPtr(v int) *int {
	return new(v)
}

//go:fix inline
func Float32Ptr(v float32) *float32 {
	return new(v)
}

//go:fix inline
func BoolPtr(v bool) *bool {
	return new(v)
}
