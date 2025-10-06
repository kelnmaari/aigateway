// Package storage provides storage interfaces and implementations for API keys
package storage

import (
	"context"
	"fmt"

	"ollama-openai-proxy/internal/models"
)

// APIKeyStorage определяет интерфейс для хранения API ключей
type APIKeyStorage interface {
	// CRUD операции
	CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error
	GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error
	DeleteAPIKey(ctx context.Context, id string) error
	
	// Поиск и фильтрация
	ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) ([]models.APIKey, int, error)
	FindAPIKeysByStatus(ctx context.Context, status models.APIKeyStatus) ([]models.APIKey, error)
	FindAPIKeysByModel(ctx context.Context, model string) ([]models.APIKey, error)
	
	// Статистика и мониторинг
	GetAPIKeyUsageStats(ctx context.Context, id string) (*models.APIKeyUsage, error)
	UpdateAPIKeyUsage(ctx context.Context, id string, usage models.APIKeyUsage) error
	
	// Административные операции
	RevokeAPIKey(ctx context.Context, id string, reason string) error
	EnableAPIKey(ctx context.Context, id string) error
	ExpireAPIKey(ctx context.Context, id string) error
	CleanupExpiredKeys(ctx context.Context) (int, error)
	
	// Backup и restore
	ExportAPIKeys(ctx context.Context) ([]models.APIKey, error)
	ImportAPIKeys(ctx context.Context, apiKeys []models.APIKey) error
	
	// Здоровье storage
	HealthCheck(ctx context.Context) error
	GetStorageStats(ctx context.Context) (map[string]interface{}, error)
	
	// Lifecycle
	Initialize(ctx context.Context) error
	Close() error
}

// StorageConfig конфигурация для storage provider
type StorageConfig struct {
	Type string `json:"type"` // "json", "sqlite", "postgres"
	Path string `json:"path"` // Путь к файлу или строка подключения
	
	// Опции для различных типов storage
	Options map[string]interface{} `json:"options,omitempty"`
}

// StorageStats статистика storage
type StorageStats struct {
	Type               string `json:"type"`
	TotalKeys          int    `json:"total_keys"`
	ActiveKeys         int    `json:"active_keys"`
	ExpiredKeys        int    `json:"expired_keys"`
	RevokedKeys        int    `json:"revoked_keys"`
	LastBackupTime     *int64 `json:"last_backup_time,omitempty"`
	StorageSize        int64  `json:"storage_size_bytes"`
	LastMaintenanceTime *int64 `json:"last_maintenance_time,omitempty"`
}

// StorageError представляет ошибку storage layer
type StorageError struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Operation string `json:"operation"`
	KeyID     string `json:"key_id,omitempty"`
	Cause     error  `json:"-"`
}

func (e *StorageError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("storage %s error in %s: %s (cause: %v)", e.Type, e.Operation, e.Message, e.Cause)
	}
	return fmt.Sprintf("storage %s error in %s: %s", e.Type, e.Operation, e.Message)
}

func (e *StorageError) Unwrap() error {
	return e.Cause
}

// Storage error types
const (
	StorageErrorTypeNotFound     = "not_found"
	StorageErrorTypeAlreadyExists = "already_exists"
	StorageErrorTypeInvalidData  = "invalid_data"
	StorageErrorTypePermission   = "permission"
	StorageErrorTypeConnection   = "connection"
	StorageErrorTypeCorrupted    = "corrupted"
	StorageErrorTypeInternal     = "internal"
)

// NewStorageError создает новую ошибку storage
func NewStorageError(errorType, operation, message string, keyID string, cause error) *StorageError {
	return &StorageError{
		Type:      errorType,
		Message:   message,
		Operation: operation,
		KeyID:     keyID,
		Cause:     cause,
	}
}

// NotFoundError создает ошибку "не найдено"
func NotFoundError(operation, keyID string) *StorageError {
	return NewStorageError(
		StorageErrorTypeNotFound,
		operation,
		fmt.Sprintf("API key with ID '%s' not found", keyID),
		keyID,
		nil,
	)
}

// AlreadyExistsError создает ошибку "уже существует"
func AlreadyExistsError(operation, keyID string) *StorageError {
	return NewStorageError(
		StorageErrorTypeAlreadyExists,
		operation,
		fmt.Sprintf("API key with ID '%s' already exists", keyID),
		keyID,
		nil,
	)
}

// InvalidDataError создает ошибку "неверные данные"
func InvalidDataError(operation, message string, cause error) *StorageError {
	return NewStorageError(
		StorageErrorTypeInvalidData,
		operation,
		message,
		"",
		cause,
	)
}

// ConnectionError создает ошибку подключения
func ConnectionError(operation, message string, cause error) *StorageError {
	return NewStorageError(
		StorageErrorTypeConnection,
		operation,
		message,
		"",
		cause,
	)
}
