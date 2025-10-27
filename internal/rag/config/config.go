// Package config provides RAG system configuration.
package config

import "time"

// RAGConfig содержит конфигурацию RAG системы
type RAGConfig struct {
	Enabled      bool              `mapstructure:"enabled"`
	VectorStore  VectorStoreConfig `mapstructure:"vector_store"`
	Embeddings   EmbeddingsConfig  `mapstructure:"embeddings"`
	Processing   ProcessingConfig  `mapstructure:"processing"`
	Retrieval    RetrievalConfig   `mapstructure:"retrieval"`
	Queue        QueueConfig       `mapstructure:"queue"`
	APISources   APISourcesConfig  `mapstructure:"api_sources"`
	DBSources    DBSourcesConfig   `mapstructure:"db_sources"`
	Security     SecurityConfig    `mapstructure:"security"`
}

// VectorStoreConfig конфигурация vector storage
type VectorStoreConfig struct {
	Backend      string            `mapstructure:"backend"` // "pgvector" или "qdrant"
	PGVector     PGVectorConfig    `mapstructure:"pgvector"`
	Qdrant       QdrantConfig      `mapstructure:"qdrant"`
}

// PGVectorConfig конфигурация pgvector
type PGVectorConfig struct {
	Dimensions        int    `mapstructure:"dimensions"`          // 768 для nomic-embed-text
	IndexType         string `mapstructure:"index_type"`          // "hnsw" или "ivfflat"
	DistanceMetric    string `mapstructure:"distance_metric"`     // "cosine", "l2", "inner_product"
	HNSWM             int    `mapstructure:"hnsw_m"`              // 16
	HNSWEFConstruction int   `mapstructure:"hnsw_ef_construction"` // 64
}

// QdrantConfig конфигурация Qdrant
type QdrantConfig struct {
	URL         string        `mapstructure:"url"`
	Collection  string        `mapstructure:"collection"`
	PreferGRPC  bool          `mapstructure:"prefer_grpc"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// EmbeddingsConfig конфигурация embeddings
type EmbeddingsConfig struct {
	Provider        string        `mapstructure:"provider"`         // "ollama"
	Model           string        `mapstructure:"model"`            // "nomic-embed-text" или "bge-m3"
	Dimensions      int           `mapstructure:"dimensions"`       // 768 или 1024
	BatchSize       int           `mapstructure:"batch_size"`       // 32
	Timeout         time.Duration `mapstructure:"timeout"`          // 60s
	ParallelWorkers int           `mapstructure:"parallel_workers"` // 2 для multi-GPU
}

// ProcessingConfig конфигурация document processing
type ProcessingConfig struct {
	ChunkSize        int             `mapstructure:"chunk_size"`         // 512 tokens
	ChunkOverlap     int             `mapstructure:"chunk_overlap"`      // 50 tokens (10%)
	ChunkingStrategy string          `mapstructure:"chunking_strategy"`  // "semantic", "fixed", "hierarchical"
	PDF              PDFConfig       `mapstructure:"pdf"`
	DOCX             DOCXConfig      `mapstructure:"docx"`
	CSV              CSVConfig       `mapstructure:"csv"`
	Images           ImagesConfig    `mapstructure:"images"`
}

// PDFConfig конфигурация PDF extraction
type PDFConfig struct {
	OCREnabled bool   `mapstructure:"ocr_enabled"`
	OCRLang    string `mapstructure:"ocr_lang"` // "rus+eng"
}

// DOCXConfig конфигурация DOCX extraction
type DOCXConfig struct {
	ExtractImages bool `mapstructure:"extract_images"`
	ExtractTables bool `mapstructure:"extract_tables"`
}

// CSVConfig конфигурация CSV parsing
type CSVConfig struct {
	MaxRows   int    `mapstructure:"max_rows"`
	Encoding  string `mapstructure:"encoding"`
	Delimiter string `mapstructure:"delimiter"`
}

// ImagesConfig конфигурация image processing
type ImagesConfig struct {
	VisionModel string `mapstructure:"vision_model"` // "llava" для OCR
	MaxSize     string `mapstructure:"max_size"`     // "10MB"
}

// RetrievalConfig конфигурация retrieval
type RetrievalConfig struct {
	TopK                int     `mapstructure:"top_k"`                 // 20
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`  // 0.7
	RerankEnabled       bool    `mapstructure:"rerank_enabled"`        // true
	RerankTopN          int     `mapstructure:"rerank_top_n"`          // 5
	RerankModel         string  `mapstructure:"rerank_model"`          // "llama3.1:8b"
	MaxContextTokens    int     `mapstructure:"max_context_tokens"`    // 65536
	ContextStrategy     string  `mapstructure:"context_strategy"`      // "dynamic" или "fixed"
}

// QueueConfig конфигурация job queue
type QueueConfig struct {
	Backend                 string        `mapstructure:"backend"` // "postgres" или "memory"
	Memory                  MemoryQueueConfig   `mapstructure:"memory"`
	Postgres                PostgresQueueConfig `mapstructure:"postgres"`
}

// MemoryQueueConfig конфигурация in-memory queue
type MemoryQueueConfig struct {
	BufferSize int `mapstructure:"buffer_size"` // 1000
	NumWorkers int `mapstructure:"num_workers"` // 4
}

// PostgresQueueConfig конфигурация PostgreSQL queue
type PostgresQueueConfig struct {
	PollInterval         time.Duration `mapstructure:"poll_interval"`           // 1s
	VisibilityTimeout    time.Duration `mapstructure:"visibility_timeout"`      // 5m
	MaxAttempts          int           `mapstructure:"max_attempts"`            // 3
	NumWorkers           int           `mapstructure:"num_workers"`             // 4
	CleanupCompletedAfter time.Duration `mapstructure:"cleanup_completed_after"` // 24h
}

// APISourcesConfig конфигурация API sources
type APISourcesConfig struct {
	DefaultTimeout      time.Duration `mapstructure:"default_timeout"`        // 30s
	MaxRetries          int           `mapstructure:"max_retries"`            // 3
	RateLimitPerSource  int           `mapstructure:"rate_limit_per_source"`  // 100 req/min
}

// DBSourcesConfig конфигурация database sources
type DBSourcesConfig struct {
	MaxConnectionsPerSource int           `mapstructure:"max_connections_per_source"` // 5
	QueryTimeout            time.Duration `mapstructure:"query_timeout"`              // 30s
	MaxRows                 int           `mapstructure:"max_rows"`                   // 10000
}

// SecurityConfig конфигурация security
type SecurityConfig struct {
	EncryptCredentials bool     `mapstructure:"encrypt_credentials"` // true
	EncryptionKey      string   `mapstructure:"encryption_key"`      // AES-256 key
	AllowedDBDrivers   []string `mapstructure:"allowed_db_drivers"`  // ["postgresql"]
	AllowedAPIDomains  []string `mapstructure:"allowed_api_domains"` // [] = все разрешены
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() RAGConfig {
	return RAGConfig{
		Enabled: true,
		VectorStore: VectorStoreConfig{
			Backend: "pgvector",
			PGVector: PGVectorConfig{
				Dimensions:         768,
				IndexType:          "hnsw",
				DistanceMetric:     "cosine",
				HNSWM:              16,
				HNSWEFConstruction: 64,
			},
		},
		Embeddings: EmbeddingsConfig{
			Provider:        "ollama",
			Model:           "nomic-embed-text",
			Dimensions:      768,
			BatchSize:       32,
			Timeout:         60 * time.Second,
			ParallelWorkers: 2,
		},
		Processing: ProcessingConfig{
			ChunkSize:        512,
			ChunkOverlap:     50,
			ChunkingStrategy: "semantic",
			PDF: PDFConfig{
				OCREnabled: true,
				OCRLang:    "rus+eng",
			},
			DOCX: DOCXConfig{
				ExtractImages: true,
				ExtractTables: true,
			},
			CSV: CSVConfig{
				MaxRows:   100000,
				Encoding:  "utf-8",
				Delimiter: ",",
			},
			Images: ImagesConfig{
				VisionModel: "llava",
				MaxSize:     "10MB",
			},
		},
		Retrieval: RetrievalConfig{
			TopK:                20,
			SimilarityThreshold: 0.7,
			RerankEnabled:       true,
			RerankTopN:          5,
			RerankModel:         "llama3.1:8b",
			MaxContextTokens:    65536,
			ContextStrategy:     "dynamic",
		},
		Queue: QueueConfig{
			Backend: "postgres",
			Memory: MemoryQueueConfig{
				BufferSize: 1000,
				NumWorkers: 4,
			},
			Postgres: PostgresQueueConfig{
				PollInterval:          1 * time.Second,
				VisibilityTimeout:     5 * time.Minute,
				MaxAttempts:           3,
				NumWorkers:            4,
				CleanupCompletedAfter: 24 * time.Hour,
			},
		},
		APISources: APISourcesConfig{
			DefaultTimeout:     30 * time.Second,
			MaxRetries:         3,
			RateLimitPerSource: 100,
		},
		DBSources: DBSourcesConfig{
			MaxConnectionsPerSource: 5,
			QueryTimeout:            30 * time.Second,
			MaxRows:                 10000,
		},
		Security: SecurityConfig{
			EncryptCredentials: true,
			AllowedDBDrivers:   []string{"postgresql"},
			AllowedAPIDomains:  []string{},
		},
	}
}


