// Package sources implements PostgreSQL data source.
package sources

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
)

// DatabaseDataSource implements DataSource for PostgreSQL
type DatabaseDataSource struct {
	name   string
	db     *sql.DB
	config DatabaseConfig
	logger *logrus.Logger
}

// DatabaseConfig конфигурация для database source
type DatabaseConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
	
	// Query settings
	Query        string // SQL query для загрузки данных
	IDColumn     string // Колонка с ID
	TitleColumn  string // Колонка с title (optional)
	ContentColumn string // Колонка с content
	
	// Incremental sync
	UpdatedAtColumn string // Колонка для incremental sync (optional)
}

// NewDatabaseDataSource creates new database data source
func NewDatabaseDataSource(name string, config DatabaseConfig, logger *logrus.Logger) (*DatabaseDataSource, error) {
	if logger == nil {
		logger = logrus.New()
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.WithField("database", config.Database).Info("Database source connected")

	return &DatabaseDataSource{
		name:   name,
		db:     db,
		config: config,
		logger: logger,
	}, nil
}

// Type возвращает тип источника
func (s *DatabaseDataSource) Type() DataSourceType {
	return SourceTypeDatabase
}

// Name возвращает имя источника
func (s *DatabaseDataSource) Name() string {
	return s.name
}

// GetMetadata возвращает метаданные
func (s *DatabaseDataSource) GetMetadata() map[string]interface{} {
	return map[string]interface{}{
		"type":     string(s.Type()),
		"name":     s.name,
		"database": s.config.Database,
		"host":     s.config.Host,
	}
}

// TestConnection проверяет доступность базы данных
func (s *DatabaseDataSource) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	startTime := time.Now()

	err := s.db.PingContext(ctx)
	latency := time.Since(startTime)

	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("Database connection failed: %v", err),
			Latency: latency,
		}, nil
	}

	// Try to execute the query
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("%s LIMIT 1", s.config.Query))
	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("Query test failed: %v", err),
			Latency: latency,
		}, nil
	}
	defer rows.Close()

	return &ConnectionTestResult{
		Success: true,
		Message: "Database connection and query successful",
		Latency: latency,
		Details: map[string]interface{}{
			"database": s.config.Database,
		},
	}, nil
}

// Fetch загружает документы из базы данных
func (s *DatabaseDataSource) Fetch(ctx context.Context) ([]Document, error) {
	s.logger.WithField("query", s.config.Query).Info("Fetching documents from database")

	startTime := time.Now()

	rows, err := s.db.QueryContext(ctx, s.config.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	documents := []Document{}

	for rows.Next() {
		// Create a slice of interface{} to hold each column value
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			s.logger.WithError(err).Warn("Failed to scan row, skipping")
			continue
		}

		// Build document from row
		doc, err := s.buildDocument(columns, values)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to build document, skipping")
			continue
		}

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	duration := time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"count":       len(documents),
		"duration_ms": duration.Milliseconds(),
	}).Info("Documents fetched from database")

	return documents, nil
}

// Sync синхронизирует данные
func (s *DatabaseDataSource) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	documents, err := s.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch documents: %w", err)
	}

	// В production здесь нужно реализовать incremental sync
	// используя UpdatedAtColumn

	result := &SyncResult{
		TotalDocuments:   len(documents),
		NewDocuments:     len(documents),
		UpdatedDocuments: 0,
		Errors:           []string{},
		SyncDuration:     time.Since(startTime),
		LastSyncAt:       time.Now(),
	}

	return result, nil
}

// buildDocument создает Document из row
func (s *DatabaseDataSource) buildDocument(columns []string, values []interface{}) (Document, error) {
	// Build metadata map
	metadata := make(map[string]interface{})
	for i, col := range columns {
		metadata[col] = values[i]
	}

	// Extract ID
	id := s.extractString(metadata, s.config.IDColumn)
	if id == "" {
		return Document{}, fmt.Errorf("ID column '%s' not found or empty", s.config.IDColumn)
	}

	// Extract content
	content := s.extractString(metadata, s.config.ContentColumn)
	if content == "" {
		return Document{}, fmt.Errorf("content column '%s' not found or empty", s.config.ContentColumn)
	}

	// Extract title (optional)
	title := ""
	if s.config.TitleColumn != "" {
		title = s.extractString(metadata, s.config.TitleColumn)
	}

	doc := Document{
		ID:          id,
		Title:       title,
		Content:     content,
		ContentType: "text/plain",
		Metadata:    metadata,
		FetchedAt:   time.Now(),
	}

	return doc, nil
}

// extractString извлекает строку из metadata
func (s *DatabaseDataSource) extractString(metadata map[string]interface{}, key string) string {
	if val, ok := metadata[key]; ok {
		switch v := val.(type) {
		case string:
			return v
		case []byte:
			return string(v)
		default:
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// Close закрывает соединение с БД
func (s *DatabaseDataSource) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Ensure DatabaseDataSource implements DataSource interface
var _ DataSource = (*DatabaseDataSource)(nil)

