// Package models provides data models for RAG system.
package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SourceType определяет тип источника данных
type SourceType string

const (
	SourceTypeFile     SourceType = "file"
	SourceTypeAPI      SourceType = "api"
	SourceTypeDatabase SourceType = "database"
	SourceTypeWeb      SourceType = "web"
)

// SourceStatus определяет статус источника данных
type SourceStatus string

const (
	SourceStatusActive   SourceStatus = "active"
	SourceStatusInactive SourceStatus = "inactive"
	SourceStatusError    SourceStatus = "error"
	SourceStatusSyncing  SourceStatus = "syncing"
)

// SyncStatus определяет статус последней синхронизации
type SyncStatus string

const (
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusPartial SyncStatus = "partial"
)

// RAGDataSource представляет источник данных для RAG
type RAGDataSource struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	UserID      uuid.UUID    `json:"user_id" db:"user_id"`
	TenantID    *uuid.UUID   `json:"tenant_id,omitempty" db:"tenant_id"`
	
	// Основная информация
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description,omitempty" db:"description"`
	SourceType  SourceType   `json:"source_type" db:"source_type"`
	
	// Конфигурация (JSON для гибкости)
	Config      SourceConfig `json:"config" db:"config"`
	
	// Credentials (зашифрованные AES-256)
	CredentialsEncrypted string `json:"-" db:"credentials_encrypted"` // не отдаем в API
	
	// Статус и метрики
	Status           SourceStatus `json:"status" db:"status"`
	LastSyncAt       *time.Time   `json:"last_sync_at,omitempty" db:"last_sync_at"`
	LastSyncStatus   *SyncStatus  `json:"last_sync_status,omitempty" db:"last_sync_status"`
	LastError        string       `json:"last_error,omitempty" db:"last_error"`
	SyncFrequency    *string      `json:"sync_frequency,omitempty" db:"sync_frequency"` // PostgreSQL interval
	
	// Настройки индексации
	IndexingConfig IndexingConfig `json:"indexing_config,omitempty" db:"indexing_config"`
	
	// Статистика
	TotalChunks     int   `json:"total_chunks" db:"total_chunks"`
	TotalTokens     int64 `json:"total_tokens" db:"total_tokens"`
	LastChunkCount  int   `json:"last_chunk_count,omitempty" db:"last_chunk_count"`
	
	// Метаданные
	Tags     []string  `json:"tags,omitempty" db:"tags"`
	IsShared bool      `json:"is_shared" db:"is_shared"`
	
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// SourceConfig конфигурация источника (varies by type)
type SourceConfig map[string]interface{}

// Scan реализует sql.Scanner для SourceConfig
func (c *SourceConfig) Scan(value interface{}) error {
	if value == nil {
		*c = make(SourceConfig)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan SourceConfig: expected []byte, got %T", value)
	}
	
	return json.Unmarshal(bytes, c)
}

// Value реализует driver.Valuer для SourceConfig
func (c SourceConfig) Value() (driver.Value, error) {
	if c == nil {
		return json.Marshal(make(SourceConfig))
	}
	return json.Marshal(c)
}

// IndexingConfig настройки индексации для источника
type IndexingConfig struct {
	ChunkStrategy  string   `json:"chunk_strategy,omitempty"`   // "row_based", "page_based", "semantic"
	EmbedColumns   []string `json:"embed_columns,omitempty"`    // для database sources
	EmbedFields    []string `json:"embed_fields,omitempty"`     // для API sources
	MaxChunkSize   int      `json:"max_chunk_size,omitempty"`   // override global
	ChunkOverlap   int      `json:"chunk_overlap,omitempty"`    // override global
}

// Scan реализует sql.Scanner для IndexingConfig
func (ic *IndexingConfig) Scan(value interface{}) error {
	if value == nil {
		*ic = IndexingConfig{}
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan IndexingConfig: expected []byte, got %T", value)
	}
	
	return json.Unmarshal(bytes, ic)
}

// Value реализует driver.Valuer для IndexingConfig
func (ic IndexingConfig) Value() (driver.Value, error) {
	return json.Marshal(ic)
}

// CreateRAGDataSourceRequest запрос на создание источника
type CreateRAGDataSourceRequest struct {
	Name           string         `json:"name" binding:"required"`
	Description    string         `json:"description"`
	SourceType     SourceType     `json:"source_type" binding:"required"`
	Config         SourceConfig   `json:"config" binding:"required"`
	Credentials    map[string]string `json:"credentials,omitempty"` // будет зашифровано
	SyncFrequency  string         `json:"sync_frequency,omitempty"` // "6h", "daily", "weekly"
	IndexingConfig IndexingConfig `json:"indexing_config,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	IsShared       bool           `json:"is_shared"`
}

// UpdateRAGDataSourceRequest запрос на обновление источника
type UpdateRAGDataSourceRequest struct {
	Name           *string         `json:"name,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Config         *SourceConfig   `json:"config,omitempty"`
	Credentials    map[string]string `json:"credentials,omitempty"`
	SyncFrequency  *string         `json:"sync_frequency,omitempty"`
	IndexingConfig *IndexingConfig `json:"indexing_config,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	IsShared       *bool           `json:"is_shared,omitempty"`
	Status         *SourceStatus   `json:"status,omitempty"`
}

// ListRAGDataSourcesResponse ответ со списком источников
type ListRAGDataSourcesResponse struct {
	Sources []RAGDataSource `json:"sources"`
	Total   int             `json:"total"`
	Limit   int             `json:"limit"`
	Offset  int             `json:"offset"`
}

// TestConnectionRequest запрос на тестирование подключения
type TestConnectionRequest struct {
	SourceType  SourceType        `json:"source_type" binding:"required"`
	Config      SourceConfig      `json:"config" binding:"required"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// TestConnectionResponse результат тестирования подключения
type TestConnectionResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details,omitempty"`
	ResponseTime int                    `json:"response_time_ms"`
}

// SyncSourceRequest запрос на синхронизацию источника
type SyncSourceRequest struct {
	SourceID uuid.UUID `json:"source_id" binding:"required"`
	Force    bool      `json:"force,omitempty"` // force даже если недавно синхронизировали
}

// SyncSourceResponse результат запуска синхронизации
type SyncSourceResponse struct {
	JobID         int64  `json:"job_id"`
	Message       string `json:"message"`
	EstimatedTime string `json:"estimated_time,omitempty"`
}

