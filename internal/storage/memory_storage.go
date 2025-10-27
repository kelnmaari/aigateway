// Package storage provides storage implementations for API keys
package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/models"
)

// MemoryStorage реализует in-memory хранилище для тестов
type MemoryStorage struct {
	mu   sync.RWMutex
	keys map[string]*models.APIKey
}

// NewMemoryStorage создает новое in-memory хранилище
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		keys: make(map[string]*models.APIKey),
	}
}

// Initialize инициализирует хранилище
func (s *MemoryStorage) Initialize(ctx context.Context) error {
	return nil
}

// Close закрывает хранилище
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = make(map[string]*models.APIKey)
	return nil
}

// CreateAPIKey создает новый API ключ
func (s *MemoryStorage) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[key.ID]; exists {
		return fmt.Errorf("API key with ID %s already exists", key.ID)
	}

	s.keys[key.ID] = key
	return nil
}

// GetAPIKey получает API ключ по ID
func (s *MemoryStorage) GetAPIKey(ctx context.Context, keyID string) (*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("API key not found: %s", keyID)
	}

	return key, nil
}

// GetAPIKeyByHash получает API ключ по хешу
func (s *MemoryStorage) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, key := range s.keys {
		if key.KeyHash == keyHash {
			return key, nil
		}
	}

	return nil, fmt.Errorf("API key not found by hash")
}

// ListAPIKeys возвращает список API ключей с фильтрацией и пагинацией
func (s *MemoryStorage) ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) ([]models.APIKey, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var allKeys []models.APIKey
	for _, key := range s.keys {
		// Фильтрация по статусу
		if req.Status != nil && key.Status != *req.Status {
			continue
		}

		// Фильтрация по модели
		if req.Model != nil && *req.Model != "" {
			hasModel := false
			for _, m := range key.Models {
				if m == *req.Model || m == "*" {
					hasModel = true
					break
				}
			}
			if !hasModel {
				continue
			}
		}

		allKeys = append(allKeys, *key)
	}

	total := len(allKeys)

	// Простая пагинация
	start := req.Offset
	if start > len(allKeys) {
		return []models.APIKey{}, total, nil
	}

	end := start + req.Limit
	// Если limit = 0, возвращаем все ключи от start
	if req.Limit == 0 {
		end = len(allKeys)
	}
	if end > len(allKeys) {
		end = len(allKeys)
	}

	return allKeys[start:end], total, nil
}

// FindAPIKeysByStatus находит ключи по статусу
func (s *MemoryStorage) FindAPIKeysByStatus(ctx context.Context, status models.APIKeyStatus) ([]models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.APIKey
	for _, key := range s.keys {
		if key.Status == status {
			result = append(result, *key)
		}
	}

	return result, nil
}

// FindAPIKeysByModel находит ключи с доступом к модели
func (s *MemoryStorage) FindAPIKeysByModel(ctx context.Context, model string) ([]models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.APIKey
	for _, key := range s.keys {
		for _, m := range key.Models {
			if m == model || m == "*" {
				result = append(result, *key)
				break
			}
		}
	}

	return result, nil
}

// UpdateAPIKey обновляет API ключ
func (s *MemoryStorage) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[key.ID]; !exists {
		return fmt.Errorf("API key not found: %s", key.ID)
	}

	key.UpdatedAt = time.Now()
	s.keys[key.ID] = key
	return nil
}

// DeleteAPIKey удаляет API ключ
func (s *MemoryStorage) DeleteAPIKey(ctx context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[keyID]; !exists {
		return fmt.Errorf("API key not found: %s", keyID)
	}

	delete(s.keys, keyID)
	return nil
}

// GetAPIKeyUsageStats получает статистику использования ключа
func (s *MemoryStorage) GetAPIKeyUsageStats(ctx context.Context, id string) (*models.APIKeyUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[id]
	if !exists {
		return nil, fmt.Errorf("API key not found: %s", id)
	}

	return &key.Usage, nil
}

// UpdateAPIKeyUsage обновляет статистику использования
func (s *MemoryStorage) UpdateAPIKeyUsage(ctx context.Context, id string, usage models.APIKeyUsage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[id]
	if !exists {
		return fmt.Errorf("API key not found: %s", id)
	}

	key.Usage = usage
	key.UpdatedAt = time.Now()
	return nil
}

// RevokeAPIKey отзывает API ключ
func (s *MemoryStorage) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[id]
	if !exists {
		return fmt.Errorf("API key not found: %s", id)
	}

	key.Status = models.APIKeyStatusRevoked
	key.UpdatedAt = time.Now()
	return nil
}

// EnableAPIKey включает ранее отозванный или отключенный ключ (AUTH-04)
func (s *MemoryStorage) EnableAPIKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[id]
	if !exists {
		return fmt.Errorf("API key not found: %s", id)
	}

	key.Status = models.APIKeyStatusActive
	key.UpdatedAt = time.Now()
	return nil
}

// ExpireAPIKey помечает ключ как истекший
func (s *MemoryStorage) ExpireAPIKey(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[id]
	if !exists {
		return fmt.Errorf("API key not found: %s", id)
	}

	key.Status = models.APIKeyStatusExpired
	key.UpdatedAt = time.Now()
	return nil
}

// CleanupExpiredKeys удаляет истекшие ключи
func (s *MemoryStorage) CleanupExpiredKeys(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleaned := 0
	now := time.Now()

	for id, key := range s.keys {
		if key.ExpiresAt != nil && key.ExpiresAt.Before(now) {
			delete(s.keys, id)
			cleaned++
		}
	}

	return cleaned, nil
}

// ExportAPIKeys экспортирует все ключи
func (s *MemoryStorage) ExportAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.APIKey
	for _, key := range s.keys {
		result = append(result, *key)
	}

	return result, nil
}

// ImportAPIKeys импортирует ключи
func (s *MemoryStorage) ImportAPIKeys(ctx context.Context, apiKeys []models.APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range apiKeys {
		s.keys[apiKeys[i].ID] = &apiKeys[i]
	}

	return nil
}

// HealthCheck проверяет здоровье storage
func (s *MemoryStorage) HealthCheck(ctx context.Context) error {
	return nil
}

// GetStorageStats получает статистику storage
func (s *MemoryStorage) GetStorageStats(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeCount := 0
	expiredCount := 0
	revokedCount := 0

	for _, key := range s.keys {
		switch key.Status {
		case models.APIKeyStatusActive:
			activeCount++
		case models.APIKeyStatusExpired:
			expiredCount++
		case models.APIKeyStatusRevoked:
			revokedCount++
		}
	}

	return map[string]interface{}{
		"type":         "memory",
		"total_keys":   len(s.keys),
		"active_keys":  activeCount,
		"expired_keys": expiredCount,
		"revoked_keys": revokedCount,
	}, nil
}

// ErrorStorage реализует storage который всегда возвращает ошибку (для тестов)
type ErrorStorage struct {
	Err error
}

// Initialize возвращает ошибку
func (s *ErrorStorage) Initialize(ctx context.Context) error {
	return s.Err
}

// Close возвращает ошибку
func (s *ErrorStorage) Close() error {
	return s.Err
}

// CreateAPIKey возвращает ошибку
func (s *ErrorStorage) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	return s.Err
}

// GetAPIKey возвращает ошибку
func (s *ErrorStorage) GetAPIKey(ctx context.Context, keyID string) (*models.APIKey, error) {
	return nil, s.Err
}

// GetAPIKeyByHash возвращает ошибку
func (s *ErrorStorage) GetAPIKeyByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	return nil, s.Err
}

// ListAPIKeys возвращает ошибку
func (s *ErrorStorage) ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) ([]models.APIKey, int, error) {
	return nil, 0, s.Err
}

// FindAPIKeysByStatus возвращает ошибку
func (s *ErrorStorage) FindAPIKeysByStatus(ctx context.Context, status models.APIKeyStatus) ([]models.APIKey, error) {
	return nil, s.Err
}

// FindAPIKeysByModel возвращает ошибку
func (s *ErrorStorage) FindAPIKeysByModel(ctx context.Context, model string) ([]models.APIKey, error) {
	return nil, s.Err
}

// UpdateAPIKey возвращает ошибку
func (s *ErrorStorage) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	return s.Err
}

// DeleteAPIKey возвращает ошибку
func (s *ErrorStorage) DeleteAPIKey(ctx context.Context, keyID string) error {
	return s.Err
}

// GetAPIKeyUsageStats возвращает ошибку
func (s *ErrorStorage) GetAPIKeyUsageStats(ctx context.Context, id string) (*models.APIKeyUsage, error) {
	return nil, s.Err
}

// UpdateAPIKeyUsage возвращает ошибку
func (s *ErrorStorage) UpdateAPIKeyUsage(ctx context.Context, id string, usage models.APIKeyUsage) error {
	return s.Err
}

// RevokeAPIKey возвращает ошибку
func (s *ErrorStorage) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	return s.Err
}

// EnableAPIKey возвращает ошибку (AUTH-04)
func (s *ErrorStorage) EnableAPIKey(ctx context.Context, id string) error {
	return s.Err
}

// ExpireAPIKey возвращает ошибку
func (s *ErrorStorage) ExpireAPIKey(ctx context.Context, id string) error {
	return s.Err
}

// CleanupExpiredKeys возвращает ошибку
func (s *ErrorStorage) CleanupExpiredKeys(ctx context.Context) (int, error) {
	return 0, s.Err
}

// ExportAPIKeys возвращает ошибку
func (s *ErrorStorage) ExportAPIKeys(ctx context.Context) ([]models.APIKey, error) {
	return nil, s.Err
}

// ImportAPIKeys возвращает ошибку
func (s *ErrorStorage) ImportAPIKeys(ctx context.Context, apiKeys []models.APIKey) error {
	return s.Err
}

// HealthCheck возвращает ошибку
func (s *ErrorStorage) HealthCheck(ctx context.Context) error {
	return s.Err
}

// GetStorageStats возвращает ошибку
func (s *ErrorStorage) GetStorageStats(ctx context.Context) (map[string]interface{}, error) {
	return nil, s.Err
}

