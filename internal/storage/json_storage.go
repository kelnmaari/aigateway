// Package storage provides JSON file storage implementation for API keys
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// JSONStorage реализует APIKeyStorage используя JSON файлы
type JSONStorage struct {
	filePath string
	logger   *logrus.Logger
	mutex    sync.RWMutex

	// In-memory кеш для быстрого доступа
	cache     map[string]*models.APIKey
	hashIndex map[string]string // key_hash -> id mapping
	lastLoad  time.Time
}

// JSONStorageData структура данных для JSON файла
type JSONStorageData struct {
	Version   int                       `json:"version"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
	APIKeys   map[string]*models.APIKey `json:"api_keys"`
}

// NewJSONStorage создает новый JSON storage
func NewJSONStorage(filePath string, logger *logrus.Logger) *JSONStorage {
	return &JSONStorage{
		filePath:  filePath,
		logger:    logger,
		cache:     make(map[string]*models.APIKey),
		hashIndex: make(map[string]string),
	}
}

// Initialize инициализирует JSON storage
func (s *JSONStorage) Initialize(ctx context.Context) error {
	s.logger.WithField("file_path", s.filePath).Info("Initializing JSON storage for API keys")

	// Создаем директорию если не существует
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewStorageError(
			StorageErrorTypePermission,
			"initialize",
			"failed to create storage directory",
			"",
			err,
		)
	}

	// Загружаем существующие данные
	if err := s.loadFromFile(); err != nil {
		if os.IsNotExist(err) {
			// Файл не существует, создаем пустой storage
			s.logger.Info("Storage file does not exist, creating new empty storage")
			return s.saveToFile()
		}
		return fmt.Errorf("failed to load storage: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"total_keys": len(s.cache),
		"file_path":  s.filePath,
	}).Info("JSON storage initialized successfully")

	return nil
}

// CreateAPIKey сохраняет новый API ключ
func (s *JSONStorage) CreateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем что ключ не существует
	if _, exists := s.cache[apiKey.ID]; exists {
		return AlreadyExistsError("create", apiKey.ID)
	}

	// Проверяем уникальность хеша
	if existingID, exists := s.hashIndex[apiKey.KeyHash]; exists {
		return NewStorageError(
			StorageErrorTypeAlreadyExists,
			"create",
			fmt.Sprintf("API key hash already exists (existing ID: %s)", existingID),
			apiKey.ID,
			nil,
		)
	}

	// Добавляем в кеш
	s.cache[apiKey.ID] = apiKey
	s.hashIndex[apiKey.KeyHash] = apiKey.ID

	// Сохраняем в файл
	if err := s.saveToFile(); err != nil {
		// Rollback кеша
		delete(s.cache, apiKey.ID)
		delete(s.hashIndex, apiKey.KeyHash)
		return fmt.Errorf("failed to save API key: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"key_id": apiKey.ID,
		"name":   apiKey.Name,
	}).Info("API key created successfully")

	return nil
}

// GetAPIKey получает API ключ по ID
func (s *JSONStorage) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	apiKey, exists := s.cache[id]
	if !exists {
		return nil, NotFoundError("get", id)
	}

	// Возвращаем копию чтобы избежать модификации кеша
	keyCopy := *apiKey
	return &keyCopy, nil
}

// GetAPIKeyByHash получает API ключ по хешу
func (s *JSONStorage) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	id, exists := s.hashIndex[keyHash]
	if !exists {
		return nil, NewStorageError(
			StorageErrorTypeNotFound,
			"get_by_hash",
			"API key not found by hash",
			"",
			nil,
		)
	}

	apiKey := s.cache[id]
	keyCopy := *apiKey
	return &keyCopy, nil
}

// UpdateAPIKey обновляет существующий API ключ
func (s *JSONStorage) UpdateAPIKey(ctx context.Context, apiKey *models.APIKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем что ключ существует
	existingKey, exists := s.cache[apiKey.ID]
	if !exists {
		return NotFoundError("update", apiKey.ID)
	}

	// Если хеш изменился, обновляем индекс
	if existingKey.KeyHash != apiKey.KeyHash {
		delete(s.hashIndex, existingKey.KeyHash)
		s.hashIndex[apiKey.KeyHash] = apiKey.ID
	}

	// Обновляем время изменения
	apiKey.UpdatedAt = time.Now()

	// Обновляем кеш
	s.cache[apiKey.ID] = apiKey

	// Сохраняем в файл
	if err := s.saveToFile(); err != nil {
		// Rollback кеша
		s.cache[apiKey.ID] = existingKey
		if existingKey.KeyHash != apiKey.KeyHash {
			delete(s.hashIndex, apiKey.KeyHash)
			s.hashIndex[existingKey.KeyHash] = apiKey.ID
		}
		return fmt.Errorf("failed to update API key: %w", err)
	}

	s.logger.WithField("key_id", apiKey.ID).Info("API key updated successfully")
	return nil
}

// DeleteAPIKey удаляет API ключ
func (s *JSONStorage) DeleteAPIKey(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Проверяем что ключ существует
	apiKey, exists := s.cache[id]
	if !exists {
		return NotFoundError("delete", id)
	}

	// Удаляем из кеша
	delete(s.cache, id)
	delete(s.hashIndex, apiKey.KeyHash)

	// Сохраняем в файл
	if err := s.saveToFile(); err != nil {
		// Rollback кеша
		s.cache[id] = apiKey
		s.hashIndex[apiKey.KeyHash] = id
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	s.logger.WithField("key_id", id).Info("API key deleted successfully")
	return nil
}

// ListAPIKeys получает список API ключей с фильтрацией
func (s *JSONStorage) ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) ([]models.APIKey, int, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Собираем все ключи
	var allKeys []models.APIKey
	for _, apiKey := range s.cache {
		keyCopy := *apiKey
		allKeys = append(allKeys, keyCopy)
	}

	// Применяем фильтры
	filtered := s.applyFilters(allKeys, req)
	total := len(filtered)

	// Применяем сортировку
	s.applySorting(filtered, req.SortBy, req.SortOrder)

	// Применяем пагинацию
	start := req.Offset
	end := start + req.Limit

	if start >= len(filtered) {
		return []models.APIKey{}, total, nil
	}

	if end > len(filtered) {
		end = len(filtered)
	}

	if req.Limit > 0 {
		filtered = filtered[start:end]
	}

	return filtered, total, nil
}

// FindAPIKeysByStatus находит ключи по статусу
func (s *JSONStorage) FindAPIKeysByStatus(ctx context.Context, status models.APIKeyStatus) ([]models.APIKey, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []models.APIKey
	for _, apiKey := range s.cache {
		if apiKey.Status == status {
			keyCopy := *apiKey
			result = append(result, keyCopy)
		}
	}

	return result, nil
}

// FindAPIKeysByModel находит ключи с доступом к модели
func (s *JSONStorage) FindAPIKeysByModel(ctx context.Context, model string) ([]models.APIKey, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var result []models.APIKey
	for _, apiKey := range s.cache {
		if apiKey.HasModelAccess(model) {
			keyCopy := *apiKey
			result = append(result, keyCopy)
		}
	}

	return result, nil
}

// GetAPIKeyUsageStats получает статистику использования ключа
func (s *JSONStorage) GetAPIKeyUsageStats(ctx context.Context, id string) (*models.APIKeyUsage, error) {
	apiKey, err := s.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	usageCopy := apiKey.Usage
	return &usageCopy, nil
}

// UpdateAPIKeyUsage обновляет статистику использования
func (s *JSONStorage) UpdateAPIKeyUsage(ctx context.Context, id string, usage models.APIKeyUsage) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	apiKey, exists := s.cache[id]
	if !exists {
		return NotFoundError("update_usage", id)
	}

	apiKey.Usage = usage
	apiKey.UpdatedAt = time.Now()

	// Сохраняем в файл (можно оптимизировать batch updates)
	return s.saveToFile()
}

// RevokeAPIKey отзывает API ключ
func (s *JSONStorage) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	apiKey, exists := s.cache[id]
	if !exists {
		return NotFoundError("revoke", id)
	}

	now := time.Now()
	apiKey.Status = models.APIKeyStatusRevoked
	apiKey.RevokedAt = &now
	apiKey.RevokedReason = reason
	apiKey.UpdatedAt = now

	if err := s.saveToFile(); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"key_id": id,
		"reason": reason,
	}).Info("API key revoked")
	return nil
}

// EnableAPIKey активирует отозванный API ключ (AUTH-04)
func (s *JSONStorage) EnableAPIKey(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	apiKey, exists := s.cache[id]
	if !exists {
		return NotFoundError("enable", id)
	}

	// Активируем только если ключ был revoked
	if apiKey.Status != models.APIKeyStatusRevoked {
		return fmt.Errorf("cannot enable key with status: %s", apiKey.Status)
	}

	apiKey.Status = models.APIKeyStatusActive
	apiKey.RevokedAt = nil
	apiKey.RevokedReason = ""
	apiKey.UpdatedAt = time.Now()

	if err := s.saveToFile(); err != nil {
		return fmt.Errorf("failed to enable API key: %w", err)
	}

	s.logger.WithField("key_id", id).Info("API key enabled")
	return nil
}

// ExpireAPIKey помечает ключ как истекший
func (s *JSONStorage) ExpireAPIKey(ctx context.Context, id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	apiKey, exists := s.cache[id]
	if !exists {
		return NotFoundError("expire", id)
	}

	apiKey.Status = models.APIKeyStatusExpired
	now := time.Now()
	apiKey.ExpiresAt = &now
	apiKey.UpdatedAt = now

	if err := s.saveToFile(); err != nil {
		return fmt.Errorf("failed to expire API key: %w", err)
	}

	s.logger.WithField("key_id", id).Info("API key expired")
	return nil
}

// CleanupExpiredKeys очищает истекшие ключи
func (s *JSONStorage) CleanupExpiredKeys(ctx context.Context) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var expiredKeys []string
	now := time.Now()

	for id, apiKey := range s.cache {
		if apiKey.ExpiresAt != nil && now.After(*apiKey.ExpiresAt) {
			if apiKey.Status != models.APIKeyStatusExpired {
				apiKey.Status = models.APIKeyStatusExpired
				apiKey.UpdatedAt = now
			}
			expiredKeys = append(expiredKeys, id)
		}
	}

	if len(expiredKeys) > 0 {
		if err := s.saveToFile(); err != nil {
			return 0, fmt.Errorf("failed to save after cleanup: %w", err)
		}

		s.logger.WithFields(logrus.Fields{
			"expired_count": len(expiredKeys),
			"expired_keys":  expiredKeys,
		}).Info("Cleaned up expired API keys")
	}

	return len(expiredKeys), nil
}

// ExportAPIKeys экспортирует все API ключи
func (s *JSONStorage) ExportAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]models.APIKey, 0, len(s.cache))
	for _, apiKey := range s.cache {
		keyCopy := *apiKey
		result = append(result, keyCopy)
	}

	return result, nil
}

// ImportAPIKeys импортирует API ключи
func (s *JSONStorage) ImportAPIKeys(ctx context.Context, apiKeys []models.APIKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Backup текущего состояния
	backup := maps.Clone(s.cache)
	hashBackup := maps.Clone(s.hashIndex)

	// Добавляем новые ключи
	conflicts := 0
	for _, apiKey := range apiKeys {
		if _, exists := s.cache[apiKey.ID]; exists {
			conflicts++
			continue
		}

		keyCopy := apiKey
		s.cache[apiKey.ID] = &keyCopy
		s.hashIndex[apiKey.KeyHash] = apiKey.ID
	}

	// Сохраняем
	if err := s.saveToFile(); err != nil {
		// Rollback
		s.cache = backup
		s.hashIndex = hashBackup
		return fmt.Errorf("failed to import API keys: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"imported":  len(apiKeys) - conflicts,
		"conflicts": conflicts,
		"total":     len(apiKeys),
	}).Info("API keys imported")

	return nil
}

// HealthCheck проверяет здоровье storage
func (s *JSONStorage) HealthCheck(ctx context.Context) error {
	// Проверяем доступность файла
	if _, err := os.Stat(s.filePath); err != nil {
		if os.IsNotExist(err) {
			return NewStorageError(
				StorageErrorTypeNotFound,
				"health_check",
				"storage file does not exist",
				"",
				err,
			)
		}
		return NewStorageError(
			StorageErrorTypePermission,
			"health_check",
			"storage file is not accessible",
			"",
			err,
		)
	}

	// Проверяем что кеш загружен
	s.mutex.RLock()
	cacheSize := len(s.cache)
	lastLoad := s.lastLoad
	s.mutex.RUnlock()

	if cacheSize == 0 && !lastLoad.IsZero() {
		s.logger.Warn("Storage cache is empty but file was loaded")
	}

	return nil
}

// GetStorageStats возвращает статистику storage
func (s *JSONStorage) GetStorageStats(ctx context.Context) (map[string]any, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Подсчитываем статистику по статусам
	stats := map[string]any{
		"type":       "json",
		"file_path":  s.filePath,
		"total_keys": len(s.cache),
		"last_load":  s.lastLoad.Format(time.RFC3339),
	}

	statusCounts := make(map[models.APIKeyStatus]int)
	for _, apiKey := range s.cache {
		statusCounts[apiKey.Status]++
	}

	stats["active_keys"] = statusCounts[models.APIKeyStatusActive]
	stats["disabled_keys"] = statusCounts[models.APIKeyStatusDisabled]
	stats["expired_keys"] = statusCounts[models.APIKeyStatusExpired]
	stats["revoked_keys"] = statusCounts[models.APIKeyStatusRevoked]

	// Размер файла
	if fileInfo, err := os.Stat(s.filePath); err == nil {
		stats["file_size_bytes"] = fileInfo.Size()
		stats["file_modified"] = fileInfo.ModTime().Format(time.RFC3339)
	}

	return stats, nil
}

// Close закрывает storage
func (s *JSONStorage) Close() error {
	s.logger.Info("Closing JSON storage")
	return nil
}

// Private methods

// loadFromFile загружает данные из JSON файла
func (s *JSONStorage) loadFromFile() error {
	file, err := os.Open(s.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var data JSONStorageData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return NewStorageError(
			StorageErrorTypeCorrupted,
			"load",
			"failed to decode JSON storage file",
			"",
			err,
		)
	}

	// Очищаем кеш
	s.cache = make(map[string]*models.APIKey)
	s.hashIndex = make(map[string]string)

	// Загружаем данные в кеш
	for id, apiKey := range data.APIKeys {
		s.cache[id] = apiKey
		s.hashIndex[apiKey.KeyHash] = id
	}

	s.lastLoad = time.Now()

	s.logger.WithFields(logrus.Fields{
		"loaded_keys": len(s.cache),
		"version":     data.Version,
	}).Debug("Loaded API keys from JSON file")

	return nil
}

// saveToFile сохраняет данные в JSON файл
func (s *JSONStorage) saveToFile() error {
	data := JSONStorageData{
		Version:   1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		APIKeys:   s.cache,
	}

	// Создаем временный файл для atomic write
	tempFile := s.filePath + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return NewStorageError(
			StorageErrorTypePermission,
			"save",
			"failed to create temporary file",
			"",
			err,
		)
	}

	// Записываем JSON с красивым форматированием
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		file.Close()
		os.Remove(tempFile)
		return NewStorageError(
			StorageErrorTypeInternal,
			"save",
			"failed to encode JSON data",
			"",
			err,
		)
	}

	file.Close()

	// Atomic переименование
	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile)
		return NewStorageError(
			StorageErrorTypePermission,
			"save",
			"failed to rename temporary file",
			"",
			err,
		)
	}

	return nil
}

// applyFilters применяет фильтры к списку ключей
func (s *JSONStorage) applyFilters(keys []models.APIKey, req models.ListAPIKeysRequest) []models.APIKey {
	var filtered []models.APIKey

	for _, key := range keys {
		// Фильтр по статусу
		if req.Status != nil && key.Status != *req.Status {
			continue
		}

		// Фильтр по модели
		if req.Model != nil && !key.HasModelAccess(*req.Model) {
			continue
		}

		filtered = append(filtered, key)
	}

	return filtered
}

// applySorting применяет сортировку к списку ключей
func (s *JSONStorage) applySorting(keys []models.APIKey, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}

	sort.Slice(keys, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "name":
			less = strings.ToLower(keys[i].Name) < strings.ToLower(keys[j].Name)
		case "created_at":
			less = keys[i].CreatedAt.Before(keys[j].CreatedAt)
		case "last_used":
			// Обработка nil значений
			if keys[i].LastUsedAt == nil && keys[j].LastUsedAt == nil {
				less = false
			} else if keys[i].LastUsedAt == nil {
				less = true
			} else if keys[j].LastUsedAt == nil {
				less = false
			} else {
				less = keys[i].LastUsedAt.Before(*keys[j].LastUsedAt)
			}
		default:
			less = keys[i].CreatedAt.Before(keys[j].CreatedAt)
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}
