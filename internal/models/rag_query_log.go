package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RAGQueryLog представляет лог RAG запроса для аналитики
type RAGQueryLog struct {
	ID                    int64     `json:"id" db:"id"`
	UserID                *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	ConversationID        *uuid.UUID `json:"conversation_id,omitempty" db:"conversation_id"`
	
	// Запрос
	QueryText             string    `json:"query_text" db:"query_text"`
	
	// Использованные источники
	SourceIDs             []uuid.UUID `json:"source_ids,omitempty" db:"source_ids"`
	
	// Результаты поиска
	ChunksRetrieved       int       `json:"chunks_retrieved,omitempty" db:"chunks_retrieved"`
	ChunksUsed            int       `json:"chunks_used,omitempty" db:"chunks_used"`
	
	// Метрики
	SearchTimeMs          int       `json:"search_time_ms,omitempty" db:"search_time_ms"`
	TotalTokensUsed       int       `json:"total_tokens_used,omitempty" db:"total_tokens_used"`
	
	// Результат
	ResponseQualityScore  *float64  `json:"response_quality_score,omitempty" db:"response_quality_score"`
	
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// SourceIDsArray для работы с PostgreSQL array
type SourceIDsArray []uuid.UUID

// Scan реализует sql.Scanner для SourceIDsArray
func (a *SourceIDsArray) Scan(value interface{}) error {
	if value == nil {
		*a = []uuid.UUID{}
		return nil
	}
	
	// Для SQLite (JSON array)
	if bytes, ok := value.([]byte); ok {
		var strArr []string
		if err := json.Unmarshal(bytes, &strArr); err != nil {
			return fmt.Errorf("failed to unmarshal SourceIDsArray: %w", err)
		}
		
		uuids := make([]uuid.UUID, len(strArr))
		for i, s := range strArr {
			u, err := uuid.Parse(s)
			if err != nil {
				return fmt.Errorf("failed to parse UUID in SourceIDsArray: %w", err)
			}
			uuids[i] = u
		}
		*a = uuids
		return nil
	}
	
	// Для PostgreSQL native array
	if str, ok := value.(string); ok {
		_ = str // TODO: implement proper PostgreSQL array parsing
		// PostgreSQL returns arrays as {uuid1,uuid2,...}
		// Simplified parsing - for production use pq.Array
		*a = []uuid.UUID{}
		return nil
	}
	
	return fmt.Errorf("cannot scan SourceIDsArray from %T", value)
}

// Value реализует driver.Valuer для SourceIDsArray
func (a SourceIDsArray) Value() (driver.Value, error) {
	if a == nil {
		return json.Marshal([]string{})
	}
	
	// Convert to string array for JSON
	strArr := make([]string, len(a))
	for i, u := range a {
		strArr[i] = u.String()
	}
	
	return json.Marshal(strArr)
}

// RAGQueryStats статистика RAG запросов
type RAGQueryStats struct {
	TotalQueries       int     `json:"total_queries"`
	AvgSearchTimeMs    float64 `json:"avg_search_time_ms"`
	AvgChunksRetrieved float64 `json:"avg_chunks_retrieved"`
	AvgChunksUsed      float64 `json:"avg_chunks_used"`
	TotalTokensUsed    int64   `json:"total_tokens_used"`
}


