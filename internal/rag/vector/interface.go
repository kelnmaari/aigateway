// Package vector provides vector storage interfaces for RAG system.
package vector

import (
	"context"
	"time"
)

// VectorDocument представляет документ в vector store
type VectorDocument struct {
	ID          string                 // Уникальный ID chunk
	Vector      []float64              // Embedding vector
	Text        string                 // Текст chunk
	Metadata    map[string]interface{} // Метаданные (source_id, document_id, etc.)
	Score       float64                // Similarity score (для результатов поиска)
	CreatedAt   time.Time              // Время создания
}

// SearchRequest запрос для vector search
type SearchRequest struct {
	Query          []float64              // Query vector
	TopK           int                    // Количество результатов
	MinScore       float64                // Минимальный similarity score
	Filters        map[string]interface{} // Фильтры по метаданным
	IncludeVectors bool                   // Включать векторы в результат
}

// SearchResponse результат поиска
type SearchResponse struct {
	Documents  []VectorDocument // Найденные документы
	TotalFound int              // Общее количество найденных
	SearchTime int              // Время поиска в мс
}

// IndexStats статистика индекса
type IndexStats struct {
	TotalVectors   int64   // Общее количество векторов
	Dimensions     int     // Размерность векторов
	IndexType      string  // Тип индекса (HNSW, IVF, etc.)
	IndexSize      int64   // Размер индекса в байтах
	AvgSearchTime  float64 // Среднее время поиска в мс
}

// VectorStore interface для работы с vector storage
type VectorStore interface {
	// Insert добавляет один vector в хранилище
	Insert(ctx context.Context, doc VectorDocument) error
	
	// InsertBatch добавляет batch vectors
	InsertBatch(ctx context.Context, docs []VectorDocument) error
	
	// Search выполняет vector similarity search
	Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
	
	// Delete удаляет vector по ID
	Delete(ctx context.Context, id string) error
	
	// DeleteByMetadata удаляет vectors по фильтрам метаданных
	DeleteByMetadata(ctx context.Context, filters map[string]interface{}) (int, error)
	
	// Update обновляет vector
	Update(ctx context.Context, doc VectorDocument) error
	
	// GetByID получает vector по ID
	GetByID(ctx context.Context, id string) (*VectorDocument, error)
	
	// CreateIndex создает индекс для быстрого поиска
	CreateIndex(ctx context.Context, indexType string, params map[string]interface{}) error
	
	// GetIndexStats возвращает статистику индекса
	GetIndexStats(ctx context.Context) (*IndexStats, error)
	
	// HealthCheck проверяет доступность vector store
	HealthCheck(ctx context.Context) error
	
	// Name возвращает имя vector store provider
	Name() string
}

// VectorStoreConfig конфигурация vector store
type VectorStoreConfig struct {
	Provider    string // "pgvector", "qdrant", "milvus", "weaviate"
	
	// PostgreSQL pgvector
	PostgreSQL *PostgreSQLConfig
	
	// Qdrant
	Qdrant *QdrantConfig
	
	// Common settings
	Dimensions    int     // Размерность векторов
	IndexType     string  // "hnsw", "ivf", "flat"
	DistanceMetric string // "cosine", "l2", "dot_product"
}

// PostgreSQLConfig конфигурация для pgvector
type PostgreSQLConfig struct {
	// Connection (либо ConnectionString, либо Host+Port+etc)
	ConnectionString string // PostgreSQL connection string (приоритет)
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SSLMode          string
	
	// Table settings
	TableName      string // Название таблицы для vectors
	ChunkIDColumn  string // Колонка для chunk_id
	VectorColumn   string // Колонка для vector
	MetadataColumn string // Колонка для metadata (JSONB)
	
	// Vector settings
	Dimensions     int    // Размерность векторов (768, 1024, etc.)
	DistanceMetric string // Metric для similarity search (cosine, l2, inner_product)
	IndexType      string // hnsw или ivfflat
	CreateIndex    bool   // Создать index при инициализации
	
	// Index settings
	HNSWParams map[string]interface{} // m, ef_construction, ef_search
}

// QdrantConfig конфигурация для Qdrant
type QdrantConfig struct {
	URL        string
	APIKey     string
	Collection string
	Timeout    int
}

// DefaultVectorStoreConfig возвращает конфигурацию по умолчанию
func DefaultVectorStoreConfig() VectorStoreConfig {
	return VectorStoreConfig{
		Provider:       "pgvector",
		Dimensions:     1024,
		IndexType:      "hnsw",
		DistanceMetric: "cosine",
		PostgreSQL: &PostgreSQLConfig{
			Host:           "localhost",
			Port:           5432,
			Database:       "aigateway",
			User:           "postgres",
			Password:       "postgres",
			SSLMode:        "disable",
			TableName:      "rag_chunk_embeddings",
			ChunkIDColumn:  "chunk_id",
			VectorColumn:   "embedding",
			MetadataColumn: "metadata",
			HNSWParams: map[string]interface{}{
				"m":               16,
				"ef_construction": 64,
				"ef_search":       40,
			},
		},
	}
}


