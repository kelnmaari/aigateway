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

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
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
func (s *DataSourceService) CreateDataSource(ctx context.Context, req *models.CreateRAGDataSourceRequest, userID uuid.UUID, tenantID *uuid.UUID) (*models.RAGDataSource, error) {
	// Encrypt credentials if provided
	var credentialsEncrypted string
	if len(req.Credentials) > 0 {
		encrypted, err := s.encryptCredentials(req.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		credentialsEncrypted = encrypted
	}
	
	// Create data source
	source := &models.RAGDataSource{
		ID:                   uuid.New(),
		UserID:               userID,
		TenantID:             tenantID,
		Name:                 req.Name,
		Description:          req.Description,
		SourceType:           req.SourceType,
		Config:               req.Config,
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
func (s *DataSourceService) GetDataSource(ctx context.Context, id uuid.UUID) (*models.RAGDataSource, error) {
	return s.db.GetRAGDataSource(ctx, id.String())
}

// ListDataSources возвращает список источников с фильтрацией
func (s *DataSourceService) ListDataSources(ctx context.Context, filter storage.DataSourceFilter) ([]models.RAGDataSource, int, error) {
	// Преобразуем DataSourceFilter в RAGDataSourceFilter
	ragFilter := &storage.RAGDataSourceFilter{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	
	if filter.UserID != nil {
		userIDStr := filter.UserID.String()
		ragFilter.UserID = &userIDStr
	}
	if filter.TenantID != nil {
		tenantIDStr := filter.TenantID.String()
		ragFilter.TenantID = &tenantIDStr
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
func (s *DataSourceService) UpdateDataSource(ctx context.Context, id uuid.UUID, req *models.UpdateRAGDataSourceRequest) (*models.RAGDataSource, error) {
	// Get existing source
	source, err := s.db.GetRAGDataSource(ctx, id.String())
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
func (s *DataSourceService) DeleteDataSource(ctx context.Context, id uuid.UUID) error {
	if err := s.db.DeleteRAGDataSource(ctx, id.String()); err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"source_id": id,
	}).Info("Data source deleted")
	
	return nil
}

// UpdateSourceStatus обновляет статус источника
func (s *DataSourceService) UpdateSourceStatus(ctx context.Context, id uuid.UUID, status models.SourceStatus, errorMsg string) error {
	// Получаем источник и обновляем его статус
	source, err := s.db.GetRAGDataSource(ctx, id.String())
	if err != nil {
		return err
	}
	
	source.Status = status
	source.LastError = errorMsg
	source.UpdatedAt = time.Now()
	
	return s.db.UpdateRAGDataSource(ctx, source)
}

// UpdateSyncInfo обновляет информацию о синхронизации
func (s *DataSourceService) UpdateSyncInfo(ctx context.Context, id uuid.UUID, status models.SyncStatus, chunkCount int) error {
	// Получаем источник и обновляем информацию о синхронизации
	source, err := s.db.GetRAGDataSource(ctx, id.String())
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

