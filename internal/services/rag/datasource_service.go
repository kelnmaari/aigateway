// Package rag provides RAG system services.
package rag

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DataSourceService сервис для управления data sources
type DataSourceService struct {
	db            storage.Database
	encryptionKey []byte
	logger        *logrus.Logger
}

// NewDataSourceService создает новый DataSourceService
func NewDataSourceService(db storage.Database, encryptionKey string, logger *logrus.Logger) (*DataSourceService, error) {
	// Validate encryption key
	if len(encryptionKey) != 32 {
		return nil, errors.New("encryption key must be 32 bytes for AES-256")
	}

	return &DataSourceService{
		db:            db,
		encryptionKey: []byte(encryptionKey),
		logger:        logger,
	}, nil
}

// CreateDataSource создает новый источник данных
func (s *DataSourceService) CreateDataSource(ctx context.Context, req *models.CreateRAGDataSourceRequest, userID string, tenantID *string) (*models.RAGDataSource, error) {
	// Encrypt credentials if provided
	var credentialsEncrypted string
	if len(req.Credentials) > 0 {
		encrypted, err := s.encryptCredentials(req.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		credentialsEncrypted = encrypted
	}

	// For database sources, build connection_string from config + credentials
	config := req.Config
	if req.SourceType == models.SourceTypeDatabase {
		if err := s.buildDatabaseConnectionString(&config, req.Credentials); err != nil {
			return nil, fmt.Errorf("failed to build database connection string: %w", err)
		}
	}

	// Create data source
	source := &models.RAGDataSource{
		ID:                   uuid.New().String(),
		UserID:               userID,
		TenantID:             tenantID,
		Name:                 req.Name,
		Description:          req.Description,
		SourceType:           req.SourceType,
		Config:               config,
		CredentialsEncrypted: credentialsEncrypted,
		Status:               models.SourceStatusActive,
		IndexingConfig:       req.IndexingConfig,
		Tags:                 req.Tags,
		IsShared:             req.IsShared,
		TotalChunks:          0,
		TotalTokens:          0,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.db.CreateRAGDataSource(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"source_type": source.SourceType,
		"user_id":     userID,
	}).Info("Data source created")

	return source, nil
}

// GetDataSource получает источник по ID
func (s *DataSourceService) GetDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	return s.db.GetRAGDataSource(ctx, id)
}

// ListDataSources возвращает список источников с фильтрацией
func (s *DataSourceService) ListDataSources(ctx context.Context, filter storage.DataSourceFilter) ([]models.RAGDataSource, int, error) {
	// Преобразуем DataSourceFilter в RAGDataSourceFilter
	ragFilter := &storage.RAGDataSourceFilter{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	if filter.UserID != nil {
		ragFilter.UserID = filter.UserID
	}
	if filter.TenantID != nil {
		ragFilter.TenantID = filter.TenantID
	}
	if filter.SourceType != nil {
		sourceTypeStr := string(*filter.SourceType)
		ragFilter.SourceType = &sourceTypeStr
	}
	if filter.Status != nil {
		statusStr := string(*filter.Status)
		ragFilter.Status = &statusStr
	}
	ragFilter.Tags = filter.Tags

	sources, total, err := s.db.ListRAGDataSources(ctx, ragFilter)
	if err != nil {
		return nil, 0, err
	}

	// Преобразуем []*models.RAGDataSource в []models.RAGDataSource
	result := make([]models.RAGDataSource, len(sources))
	for i, src := range sources {
		result[i] = *src
	}

	return result, total, nil
}

// UpdateDataSource обновляет источник
func (s *DataSourceService) UpdateDataSource(ctx context.Context, id string, req *models.UpdateRAGDataSourceRequest) (*models.RAGDataSource, error) {
	// Get existing source
	source, err := s.db.GetRAGDataSource(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	// Update fields
	if req.Name != nil {
		source.Name = *req.Name
	}
	if req.Description != nil {
		source.Description = *req.Description
	}
	if req.Config != nil {
		source.Config = *req.Config
	}
	if len(req.Credentials) > 0 {
		encrypted, err := s.encryptCredentials(req.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		source.CredentialsEncrypted = encrypted
	}
	if req.SyncFrequency != nil {
		source.SyncFrequency = req.SyncFrequency
	}
	if req.IndexingConfig != nil {
		source.IndexingConfig = *req.IndexingConfig
	}
	if req.Tags != nil {
		source.Tags = req.Tags
	}
	if req.IsShared != nil {
		source.IsShared = *req.IsShared
	}
	if req.Status != nil {
		source.Status = *req.Status
	}

	source.UpdatedAt = time.Now()

	if err := s.db.UpdateRAGDataSource(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to update data source: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"source_id": source.ID,
	}).Info("Data source updated")

	return source, nil
}

// DeleteDataSource удаляет источник
func (s *DataSourceService) DeleteDataSource(ctx context.Context, id string) error {
	if err := s.db.DeleteRAGDataSource(ctx, id); err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"source_id": id,
	}).Info("Data source deleted")

	return nil
}

// StartSync создает RAG job для синхронизации источника
func (s *DataSourceService) StartSync(ctx context.Context, sourceID string) (int64, error) {
	// Get source to determine job type
	source, err := s.db.GetRAGDataSource(ctx, sourceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get data source: %w", err)
	}

	// Determine job type based on source type
	var jobType models.JobType
	switch source.SourceType {
	case models.SourceTypeAPI:
		jobType = models.JobTypeAPISync
	case models.SourceTypeDatabase:
		jobType = models.JobTypeDBQuery
	case models.SourceTypeWeb:
		jobType = models.JobTypeWebScrape
	case models.SourceTypeFile:
		jobType = models.JobTypeAPISync // File sources use generic sync
	default:
		jobType = models.JobTypeAPISync // fallback
	}

	// Create job
	job := &models.RAGJob{
		JobType: jobType,
		Status:  models.JobStatusPending,
		Payload: models.JobPayload{
			"source_id":   sourceID,
			"source_type": string(source.SourceType),
			"source_name": source.Name,
		},
		Priority:    5, // normal priority
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	if err := s.db.CreateRAGJob(ctx, job); err != nil {
		return 0, fmt.Errorf("failed to create sync job: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"job_id":    job.ID,
		"source_id": sourceID,
		"job_type":  jobType,
	}).Info("Sync job created successfully")

	return job.ID, nil
}

// UpdateSourceStatus обновляет статус источника
func (s *DataSourceService) UpdateSourceStatus(ctx context.Context, id string, status models.SourceStatus, errorMsg string) error {
	// Получаем источник и обновляем его статус
	source, err := s.db.GetRAGDataSource(ctx, id)
	if err != nil {
		return err
	}

	source.Status = status
	source.LastError = errorMsg
	source.UpdatedAt = time.Now()

	return s.db.UpdateRAGDataSource(ctx, source)
}

// UpdateSyncInfo обновляет информацию о синхронизации
func (s *DataSourceService) UpdateSyncInfo(ctx context.Context, id string, status models.SyncStatus, chunkCount int) error {
	// Получаем источник и обновляем информацию о синхронизации
	source, err := s.db.GetRAGDataSource(ctx, id)
	if err != nil {
		return err
	}

	source.LastSyncStatus = &status
	lastSyncAt := time.Now()
	source.LastSyncAt = &lastSyncAt
	source.LastChunkCount = chunkCount
	source.UpdatedAt = time.Now()

	return s.db.UpdateRAGDataSource(ctx, source)
}

// encryptCredentials шифрует credentials с использованием AES-256
func (s *DataSourceService) encryptCredentials(credentials map[string]string) (string, error) {
	// Convert map to JSON-like string
	data := fmt.Sprintf("%v", credentials)

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	// Create a new GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Return base64 encoded string
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptCredentials расшифровывает credentials (для будущего использования)
func (s *DataSourceService) decryptCredentials(encryptedData string) (map[string]string, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	_ = plaintext // Use plaintext

	// TODO: proper JSON unmarshaling
	// For now, this is a placeholder
	return make(map[string]string), nil
}

// buildDatabaseConnectionString constructs connection_string from config fields + password
func (s *DataSourceService) buildDatabaseConnectionString(config *models.SourceConfig, credentials map[string]string) error {
	cfg := *config

	// Extract fields
	host, _ := cfg["host"].(string)
	port, _ := cfg["port"].(float64) // JSON unmarshal converts numbers to float64
	if port == 0 {
		port = 5432 // default PostgreSQL port
	}
	database, _ := cfg["database"].(string)
	username, _ := cfg["username"].(string)
	password, _ := credentials["password"]
	dbType, _ := cfg["database_type"].(string)
	if dbType == "" {
		dbType = "postgres" // default
	}

	// Validate required fields
	if host == "" || database == "" || username == "" {
		return fmt.Errorf("missing required database connection fields (host, database, username)")
	}

	// Build connection string based on database type
	var connectionString string
	switch dbType {
	case "postgres", "postgresql":
		connectionString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			username, password, host, int(port), database)
	case "mysql":
		connectionString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			username, password, host, int(port), database)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}

	// Add connection_string to config
	cfg["connection_string"] = connectionString
	*config = cfg

	return nil
}
