package models

import "time"

// LoadedModel represents a yzma model that should be loaded on server startup (v3.0.6+)
type LoadedModel struct {
	ID          string    `json:"id" db:"id"`
	ModelPath   string    `json:"model_path" db:"model_path"`
	Alias       string    `json:"alias" db:"alias"`
	LoadedAt    time.Time `json:"loaded_at" db:"loaded_at"`
	AutoLoad    bool      `json:"auto_load" db:"auto_load"`
	ContextSize *int      `json:"context_size,omitempty" db:"context_size"`
	BatchSize   *int      `json:"batch_size,omitempty" db:"batch_size"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

