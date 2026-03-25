// Package vector implements pgvector storage.
package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
)

// PgVectorStore implements VectorStore using PostgreSQL pgvector extension
type PgVectorStore struct {
	db     *sql.DB
	config PostgreSQLConfig
	logger *logrus.Logger
}

// NewPgVectorStore creates new pgvector store
func NewPgVectorStore(config PostgreSQLConfig, logger *logrus.Logger) (*PgVectorStore, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Build connection string (приоритет ConnectionString)
	var connStr string
	if config.ConnectionString != "" {
		connStr = config.ConnectionString
	} else {
		connStr = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
		)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &PgVectorStore{
		db:     db,
		config: config,
		logger: logger,
	}

	// Initialize table и extension
	if err := store.initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}

	logger.Info("PgVector store initialized successfully")

	return store, nil
}

// Name возвращает имя provider
func (s *PgVectorStore) Name() string {
	return "pgvector"
}

// initialize создает extension и таблицу если их нет
func (s *PgVectorStore) initialize() error {
	// Включаем pgvector extension
	_, err := s.db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Создаем таблицу если не существует
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			chunk_id TEXT NOT NULL UNIQUE,
			embedding vector(%d) NOT NULL,
			text TEXT NOT NULL,
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		)
	`, s.config.TableName, s.config.Dimensions)

	_, err = s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"table":      s.config.TableName,
		"dimensions": s.config.Dimensions,
	}).Info("Vector table initialized")

	// Создаем index если требуется (Version 1.14.0+)
	if s.config.CreateIndex {
		indexName := fmt.Sprintf("%s_embedding_idx", s.config.TableName)
		indexType := s.config.IndexType
		if indexType == "" {
			indexType = "hnsw" // Default to HNSW
		}

		// pgvector операторные классы для разных метрик
		opsClass := "vector_cosine_ops" // default: cosine distance
		if s.config.DistanceMetric == "l2" {
			opsClass = "vector_l2_ops" // L2/Euclidean distance
		} else if s.config.DistanceMetric == "inner_product" || s.config.DistanceMetric == "dot_product" {
			opsClass = "vector_ip_ops" // Inner product
		}

		// HNSW parameters
		m := 16              // default
		efConstruction := 64 // default
		if s.config.HNSWParams != nil {
			if mVal, ok := s.config.HNSWParams["m"].(int); ok {
				m = mVal
			}
			if efVal, ok := s.config.HNSWParams["ef_construction"].(int); ok {
				efConstruction = efVal
			}
		}

		createIndexSQL := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING %s (embedding %s) 
			WITH (m = %d, ef_construction = %d)
		`, indexName, s.config.TableName, indexType, opsClass, m, efConstruction)

		_, err = s.db.Exec(createIndexSQL)
		if err != nil {
			return fmt.Errorf("failed to create vector index: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"index": indexName,
			"type":  indexType,
			"m":     m,
			"ef":    efConstruction,
		}).Info("Vector index created")
	}

	return nil
}

// Insert добавляет один vector
func (s *PgVectorStore) Insert(ctx context.Context, doc VectorDocument) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, chunk_id, embedding, text, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chunk_id) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			text = EXCLUDED.text,
			metadata = EXCLUDED.metadata
	`, s.config.TableName)

	// Convert []float64 to PostgreSQL array format
	vectorStr := vectorToString(doc.Vector)

	_, err = s.db.ExecContext(ctx, query,
		doc.ID,
		doc.ID, // chunk_id = id
		vectorStr,
		doc.Text,
		string(metadataJSON),
		doc.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert vector: %w", err)
	}

	return nil
}

// InsertBatch добавляет batch vectors
func (s *PgVectorStore) InsertBatch(ctx context.Context, docs []VectorDocument) error {
	if len(docs) == 0 {
		return nil
	}

	// Используем batch insert
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO %s (id, chunk_id, embedding, text, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (chunk_id) DO UPDATE SET
			embedding = EXCLUDED.embedding,
			text = EXCLUDED.text,
			metadata = EXCLUDED.metadata
	`, s.config.TableName)

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, doc := range docs {
		metadataJSON, _ := json.Marshal(doc.Metadata)
		vectorStr := vectorToString(doc.Vector)

		_, err := stmt.ExecContext(ctx,
			doc.ID,
			doc.ID,
			vectorStr,
			doc.Text,
			string(metadataJSON),
			doc.CreatedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to insert vector %s: %w", doc.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithField("count", len(docs)).Info("Batch vectors inserted")

	return nil
}

// Search выполняет vector similarity search
func (s *PgVectorStore) Search(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	if len(req.Query) == 0 {
		return nil, fmt.Errorf("query vector is empty")
	}

	if req.TopK <= 0 {
		req.TopK = 10
	}

	// Build query с distance calculation
	var distanceOp string
	switch s.config.DistanceMetric {
	case "cosine":
		distanceOp = "<=>"
	case "l2":
		distanceOp = "<->"
	case "dot_product":
		distanceOp = "<#>"
	default:
		distanceOp = "<=>" // Default cosine
	}

	var querySQL strings.Builder
	querySQL.WriteString(fmt.Sprintf(`
		SELECT 
			id, chunk_id, text, metadata, created_at,
			1 - (embedding %s $1::vector) AS score
		FROM %s
		WHERE 1=1
	`, distanceOp, s.config.TableName))

	args := []any{vectorToString(req.Query)}
	argIndex := 2

	// Add filters
	if len(req.Filters) > 0 {
		for key, value := range req.Filters {
			querySQL.WriteString(fmt.Sprintf(" AND metadata->>'%s' = $%d", key, argIndex))
			args = append(args, fmt.Sprintf("%v", value))
			argIndex++
		}
	}

	// Add min score filter
	if req.MinScore > 0 {
		querySQL.WriteString(fmt.Sprintf(" AND 1 - (embedding %s $1::vector) >= $%d", distanceOp, argIndex))
		args = append(args, req.MinScore)
		argIndex++
	}

	// Order by similarity и limit
	querySQL.WriteString(fmt.Sprintf(" ORDER BY embedding %s $1::vector LIMIT $%d", distanceOp, argIndex))
	args = append(args, req.TopK)

	startTime := time.Now()

	rows, err := s.db.QueryContext(ctx, querySQL.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer rows.Close()

	var documents []VectorDocument

	for rows.Next() {
		var doc VectorDocument
		var chunkID string
		var metadataJSON string

		err := rows.Scan(
			&doc.ID,
			&chunkID,
			&doc.Text,
			&metadataJSON,
			&doc.CreatedAt,
			&doc.Score,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Parse metadata
		if err := json.Unmarshal([]byte(metadataJSON), &doc.Metadata); err != nil {
			doc.Metadata = make(map[string]any)
		}

		documents = append(documents, doc)
	}

	searchTime := time.Since(startTime).Milliseconds()

	s.logger.WithFields(logrus.Fields{
		"results":     len(documents),
		"search_time": searchTime,
		"top_k":       req.TopK,
	}).Debug("Vector search completed")

	return &SearchResponse{
		Documents:  documents,
		TotalFound: len(documents),
		SearchTime: int(searchTime),
	}, nil
}

// Delete удаляет vector по ID
func (s *PgVectorStore) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.config.TableName)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteByMetadata удаляет vectors по метаданным
func (s *PgVectorStore) DeleteByMetadata(ctx context.Context, filters map[string]any) (int, error) {
	if len(filters) == 0 {
		return 0, fmt.Errorf("filters are required")
	}

	var query strings.Builder
	query.WriteString(fmt.Sprintf("DELETE FROM %s WHERE 1=1", s.config.TableName))
	var args []any
	argIndex := 1

	for key, value := range filters {
		query.WriteString(fmt.Sprintf(" AND metadata->>'%s' = $%d", key, argIndex))
		args = append(args, fmt.Sprintf("%v", value))
		argIndex++
	}

	result, err := s.db.ExecContext(ctx, query.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete by metadata: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

// Update обновляет vector
func (s *PgVectorStore) Update(ctx context.Context, doc VectorDocument) error {
	metadataJSON, _ := json.Marshal(doc.Metadata)

	query := fmt.Sprintf(`
		UPDATE %s SET
			embedding = $1,
			text = $2,
			metadata = $3
		WHERE id = $4
	`, s.config.TableName)

	vectorStr := vectorToString(doc.Vector)

	_, err := s.db.ExecContext(ctx, query, vectorStr, doc.Text, string(metadataJSON), doc.ID)
	return err
}

// GetByID получает vector по ID
func (s *PgVectorStore) GetByID(ctx context.Context, id string) (*VectorDocument, error) {
	query := fmt.Sprintf(`
		SELECT id, chunk_id, text, metadata, created_at
		FROM %s
		WHERE id = $1
	`, s.config.TableName)

	var doc VectorDocument
	var chunkID string
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&chunkID,
		&doc.Text,
		&metadataJSON,
		&doc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vector not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get vector: %w", err)
	}

	json.Unmarshal([]byte(metadataJSON), &doc.Metadata)

	return &doc, nil
}

// CreateIndex создает HNSW индекс для быстрого поиска
func (s *PgVectorStore) CreateIndex(ctx context.Context, indexType string, params map[string]any) error {
	indexName := fmt.Sprintf("%s_embedding_idx", s.config.TableName)

	if indexType == "hnsw" {
		m := 16
		efConstruction := 64

		if val, ok := params["m"].(int); ok {
			m = val
		}
		if val, ok := params["ef_construction"].(int); ok {
			efConstruction = val
		}

		query := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING hnsw (embedding vector_cosine_ops)
			WITH (m = %d, ef_construction = %d)
		`, indexName, s.config.TableName, m, efConstruction)

		_, err := s.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create HNSW index: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"index":           indexName,
			"m":               m,
			"ef_construction": efConstruction,
		}).Info("HNSW index created")

	} else {
		// Default IVF или flat index
		query := fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s 
			USING ivfflat (embedding vector_cosine_ops)
		`, indexName, s.config.TableName)

		_, err := s.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetIndexStats возвращает статистику индекса
func (s *PgVectorStore) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", s.config.TableName)

	var totalVectors int64
	err := s.db.QueryRowContext(ctx, query).Scan(&totalVectors)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &IndexStats{
		TotalVectors:  totalVectors,
		Dimensions:    s.config.Dimensions,
		IndexType:     "hnsw",
		AvgSearchTime: 0.0,
	}, nil
}

// HealthCheck проверяет доступность
func (s *PgVectorStore) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close закрывает соединение
func (s *PgVectorStore) Close() error {
	return s.db.Close()
}

// vectorToString converts []float64 to PostgreSQL array format
func vectorToString(vec []float64) string {
	if len(vec) == 0 {
		return "[]"
	}

	strs := make([]string, len(vec))
	for i, v := range vec {
		strs[i] = fmt.Sprintf("%f", v)
	}

	return "[" + strings.Join(strs, ",") + "]"
}

// stringToVector converts PostgreSQL array string to []float64
func stringToVector(s string) ([]float64, error) {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	if s == "" {
		return []float64{}, nil
	}

	parts := strings.Split(s, ",")
	vec := make([]float64, len(parts))

	for i, part := range parts {
		var val float64
		_, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &val)
		if err != nil {
			return nil, fmt.Errorf("failed to parse vector element: %w", err)
		}
		vec[i] = val
	}

	return vec, nil
}

// Ensure PgVectorStore implements VectorStore interface
var _ VectorStore = (*PgVectorStore)(nil)
