package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// RAGQueryLog представляет лог RAG запроса для аналитики
type RAGQueryLog struct {
	ID             int64   `json:"id" db:"id"`
	UserID         *string `json:"user_id,omitempty" db:"user_id"`
	ConversationID *string `json:"conversation_id,omitempty" db:"conversation_id"`

	// Запрос
	QueryText string `json:"query_text" db:"query_text"`

	// Использованные источники
	SourceIDs []string `json:"source_ids,omitempty" db:"source_ids"`

	// Результаты поиска
	ChunksRetrieved int `json:"chunks_retrieved,omitempty" db:"chunks_retrieved"`
	ChunksUsed      int `json:"chunks_used,omitempty" db:"chunks_used"`

	// Метрики
	SearchTimeMs    int `json:"search_time_ms,omitempty" db:"search_time_ms"`
	TotalTokensUsed int `json:"total_tokens_used,omitempty" db:"total_tokens_used"`

	// Результат
	ResponseQualityScore *float64 `json:"response_quality_score,omitempty" db:"response_quality_score"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// SourceIDsArray для работы с PostgreSQL array
type SourceIDsArray []string

// Scan реализует sql.Scanner для SourceIDsArray
func (a *SourceIDsArray) Scan(value any) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	// Для SQLite/PostgreSQL JSONB (JSON array)
	if bytes, ok := value.([]byte); ok {
		var strArr []string
		if err := json.Unmarshal(bytes, &strArr); err != nil {
			return fmt.Errorf("failed to unmarshal SourceIDsArray: %w", err)
		}
		*a = strArr
		return nil
	}

	// Для PostgreSQL TEXT[] array (используется через pq.Array в rag.go)
	if str, ok := value.(string); ok {
		_ = str // PostgreSQL pq.Array handles this automatically
		*a = []string{}
		return nil
	}

	return fmt.Errorf("cannot scan SourceIDsArray from %T", value)
}

// Value реализует driver.Valuer для SourceIDsArray
func (a SourceIDsArray) Value() (driver.Value, error) {
	if a == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(a)
}

// RAGQueryStats статистика RAG запросов
type RAGQueryStats struct {
	TotalQueries       int     `json:"total_queries"`
	AvgSearchTimeMs    float64 `json:"avg_search_time_ms"`
	AvgChunksRetrieved float64 `json:"avg_chunks_retrieved"`
	AvgChunksUsed      float64 `json:"avg_chunks_used"`
	TotalTokensUsed    int64   `json:"total_tokens_used"`
}
