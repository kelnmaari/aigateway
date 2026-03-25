package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DocumentStatus статус обработки документа
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusCompleted  DocumentStatus = "completed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// RAGDocument представляет документ в RAG системе
type RAGDocument struct {
	ID       string `json:"id" db:"id"`
	SourceID string `json:"source_id" db:"source_id"`

	// Файл информация
	Filename  string `json:"filename,omitempty" db:"filename"`
	MimeType  string `json:"mime_type,omitempty" db:"mime_type"`
	SizeBytes int64  `json:"size_bytes,omitempty" db:"size_bytes"`

	// Хранилище
	StorageBackend string `json:"storage_backend" db:"storage_backend"` // 'local', 's3'
	StoragePath    string `json:"storage_path" db:"storage_path"`
	StorageBucket  string `json:"storage_bucket,omitempty" db:"storage_bucket"`

	// Обработка
	Status                DocumentStatus `json:"status" db:"status"`
	ProcessingStartedAt   *time.Time     `json:"processing_started_at,omitempty" db:"processing_started_at"`
	ProcessingCompletedAt *time.Time     `json:"processing_completed_at,omitempty" db:"processing_completed_at"`
	ProcessingError       string         `json:"processing_error,omitempty" db:"processing_error"`

	// Метаданные документа
	Metadata DocumentMetadata `json:"metadata,omitempty" db:"metadata"`

	// Статистика
	TotalChunks int   `json:"total_chunks" db:"total_chunks"`
	TotalTokens int64 `json:"total_tokens" db:"total_tokens"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DocumentMetadata метаданные документа
type DocumentMetadata map[string]any

// Scan реализует sql.Scanner для DocumentMetadata
func (m *DocumentMetadata) Scan(value any) error {
	if value == nil {
		*m = make(DocumentMetadata)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan DocumentMetadata: expected []byte or string, got %T", value)
	}

	return json.Unmarshal(bytes, m)
}

// Value реализует driver.Valuer для DocumentMetadata
func (m DocumentMetadata) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(make(DocumentMetadata))
	}
	return json.Marshal(m)
}
