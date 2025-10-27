package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RAGChunk представляет chunk текста в RAG системе
type RAGChunk struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	DocumentID  uuid.UUID     `json:"document_id" db:"document_id"`
	SourceID    uuid.UUID     `json:"source_id" db:"source_id"`
	
	// Chunk контент
	ChunkText   string        `json:"chunk_text" db:"chunk_text"`
	ChunkIndex  int           `json:"chunk_index" db:"chunk_index"`  // позиция в документе
	ChunkTokens int           `json:"chunk_tokens" db:"chunk_tokens"`
	
	// Embedding (будет добавлено в v1.13.3)
	// Embedding   []float32     `json:"embedding,omitempty" db:"embedding"`
	
	// Метаданные чанка
	Metadata    ChunkMetadata `json:"metadata,omitempty" db:"metadata"`
	
	// Для overlap detection
	StartOffset *int          `json:"start_offset,omitempty" db:"start_offset"`
	EndOffset   *int          `json:"end_offset,omitempty" db:"end_offset"`
	
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// ChunkMetadata метаданные chunk (page_number, headers, context, etc.)
type ChunkMetadata map[string]interface{}

// Scan реализует sql.Scanner для ChunkMetadata
func (m *ChunkMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(ChunkMetadata)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ChunkMetadata: expected []byte, got %T", value)
	}
	
	return json.Unmarshal(bytes, m)
}

// Value реализует driver.Valuer для ChunkMetadata
func (m ChunkMetadata) Value() (driver.Value, error) {
	if m == nil {
		return json.Marshal(make(ChunkMetadata))
	}
	return json.Marshal(m)
}

// CreateChunksRequest запрос на создание chunks из документа
type CreateChunksRequest struct {
	DocumentID uuid.UUID `json:"document_id" binding:"required"`
	Chunks     []struct {
		ChunkText   string                 `json:"chunk_text" binding:"required"`
		ChunkIndex  int                    `json:"chunk_index" binding:"required"`
		ChunkTokens int                    `json:"chunk_tokens" binding:"required"`
		Metadata    map[string]interface{} `json:"metadata,omitempty"`
		StartOffset *int                   `json:"start_offset,omitempty"`
		EndOffset   *int                   `json:"end_offset,omitempty"`
	} `json:"chunks" binding:"required"`
}

