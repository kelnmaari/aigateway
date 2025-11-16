// Package apikey provides API Key management for Ollama-OpenAI Proxy
package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// APIKeyInvalidator interface for cache invalidation (v3.0.6+)
type APIKeyInvalidator interface {
	InvalidateAPIKey(ctx context.Context, keyID string) error
}

// Manager управляет API ключами
type Manager struct {
	config      *config.Config
	logger      *logrus.Logger
	storage     storage.APIKeyStorage
	invalidator APIKeyInvalidator  // v3.0.6+: Redis cache invalidation

	// Статистика
	stats ManagerStats
}

// SetCacheInvalidator sets the cache invalidator for Redis (v3.0.6+)
func (m *Manager) SetCacheInvalidator(invalidator APIKeyInvalidator) {
	m.invalidator = invalidator
	m.logger.Info("✅ API Key cache invalidation enabled")
}

// ManagerStats статистика API Key Manager
type ManagerStats struct {
	TotalKeys          int64     `json:"total_keys"`
	ActiveKeys         int64     `json:"active_keys"`
	KeysCreatedToday   int64     `json:"keys_created_today"`
	KeysUsedToday      int64     `json:"keys_used_today"`
	TotalRequestsToday int64     `json:"total_requests_today"`
	LastCleanup        time.Time `json:"last_cleanup"`
	LastValidation     time.Time `json:"last_validation"`
}

// NewManager создает новый API Key Manager
func NewManager(cfg *config.Config, logger *logrus.Logger, stor storage.APIKeyStorage) *Manager {
	return &Manager{
		config:  cfg,
		logger:  logger,
		storage: stor,
		stats:   ManagerStats{},
	}
}

// Initialize инициализирует API Key Manager
func (m *Manager) Initialize(ctx context.Context) error {
	m.logger.Info("Initializing API Key Manager")

	// Инициализируем storage
	if err := m.storage.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Создаем bootstrap admin ключ если его нет
	if err := m.createBootstrapAdminKey(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to create bootstrap admin key")
	}

	// Выполняем начальную очистку истекших ключей
	if cleaned, err := m.storage.CleanupExpiredKeys(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to cleanup expired keys during initialization")
	} else if cleaned > 0 {
		m.logger.WithField("cleaned", cleaned).Info("Cleaned up expired keys during initialization")
	}

	// Обновляем статистику
	if err := m.updateStats(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to update stats during initialization")
	}

	m.logger.Info("API Key Manager initialized successfully")
	return nil
}

// CreateAPIKey создает новый API ключ
func (m *Manager) CreateAPIKey(ctx context.Context, req models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	m.logger.WithFields(logrus.Fields{
		"name":        req.Name,
		"description": req.Description,
		"models":      req.Models,
		"permissions": req.Permissions,
	}).Info("Creating new API key")

	// Валидация запроса
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("request validation failed: %w", err)
	}

	// Создаем новый API ключ
	apiKey, plainKey, err := models.NewAPIKey(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Сохраняем в storage
	if err := m.storage.CreateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	// Обновляем статистику
	m.stats.TotalKeys++
	m.stats.ActiveKeys++
	m.stats.KeysCreatedToday++

	m.logger.WithFields(logrus.Fields{
		"key_id": apiKey.ID,
		"name":   apiKey.Name,
	}).Info("API key created successfully")

	publicKey := apiKey.ToPublic()
	response := &models.CreateAPIKeyResponse{
		APIKey:   publicKey,
		PlainKey: plainKey,
		Warning:  "Save this key securely! It will not be shown again.",
	}

	return response, nil
}

// GetAPIKey получает API ключ по ID
func (m *Manager) GetAPIKey(ctx context.Context, id string) (*models.APIKeyPublic, error) {
	apiKey, err := m.storage.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	publicKey := apiKey.ToPublic()
	return &publicKey, nil
}

// ValidateAPIKey валидирует API ключ
func (m *Manager) ValidateAPIKey(ctx context.Context, plainKey string) (*models.APIKeyValidationResult, error) {
	// Проверка формата ключа
	if !models.IsValidAPIKeyFormat(plainKey) {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "invalid API key format",
		}, nil
	}

	// Получаем все ключи и проверяем каждый с помощью bcrypt
	// (мы не можем искать по хешу, т.к. bcrypt генерирует разные хеши для одного значения)
	allKeys, err := m.storage.ExportAPIKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	// Ищем ключ, проверяя plain key против каждого хеша
	var matchedKey *models.APIKey
	for i := range allKeys {
		if allKeys[i].VerifyKey(plainKey) {
			matchedKey = &allKeys[i]
			break
		}
	}

	// Если ключ не найден
	if matchedKey == nil {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "API key not found",
		}, nil
	}

	// Проверяем статус и срок действия
	if !matchedKey.IsActive() {
		errorMsg := fmt.Sprintf("API key is %s", matchedKey.Status)
		if matchedKey.IsExpired() {
			errorMsg = "API key has expired"
		}

		publicKey := matchedKey.ToPublic()
		return &models.APIKeyValidationResult{
			Valid:  false,
			APIKey: &publicKey,
			Error:  errorMsg,
		}, nil
	}

	// Обновляем время последнего использования
	matchedKey.UpdateLastUsed()
	if err := m.storage.UpdateAPIKey(ctx, matchedKey); err != nil {
		m.logger.WithError(err).Warn("Failed to update last used time")
	}

	m.stats.LastValidation = time.Now()

	publicKey := matchedKey.ToPublic()
	return &models.APIKeyValidationResult{
		Valid:       true,
		APIKey:      &publicKey,
		ModelAccess: true, // Будет проверяться отдельно
	}, nil
}

// ListAPIKeys получает список API ключей
func (m *Manager) ListAPIKeys(ctx context.Context, req models.ListAPIKeysRequest) (*models.ListAPIKeysResponse, error) {
	// Устанавливаем defaults для пагинации
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	apiKeys, total, err := m.storage.ListAPIKeys(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	// Конвертируем в публичный формат
	publicKeys := make([]models.APIKeyPublic, len(apiKeys))
	for i, key := range apiKeys {
		publicKeys[i] = key.ToPublic()
	}

	return &models.ListAPIKeysResponse{
		APIKeys: publicKeys,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// UpdateAPIKey обновляет существующий API ключ
func (m *Manager) UpdateAPIKey(ctx context.Context, id string, req models.UpdateAPIKeyRequest) (*models.APIKeyPublic, error) {
	m.logger.WithField("key_id", id).Info("Updating API key")

	// Получаем существующий ключ
	apiKey, err := m.storage.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	// Создаем копию для безопасного concurrent access
	keyCopy := *apiKey
	apiKey = &keyCopy

	// Применяем обновления
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.Description != nil {
		apiKey.Description = *req.Description
	}
	if req.Models != nil {
		apiKey.Models = *req.Models
	}
	if req.Permissions != nil {
		apiKey.Permissions = *req.Permissions
	}
	if req.RateLimits != nil {
		apiKey.RateLimits = *req.RateLimits
	}
	if req.Status != nil {
		apiKey.Status = *req.Status
	}
	if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
	}
	if req.Metadata != nil {
		apiKey.Metadata = *req.Metadata
	}

	// Сохраняем обновления
	if err := m.storage.UpdateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	// Invalidate cache (v3.0.6+)
	if m.invalidator != nil {
		if err := m.invalidator.InvalidateAPIKey(ctx, id); err != nil {
			m.logger.WithError(err).Warn("Failed to invalidate API key cache")
		}
	}

	m.logger.WithField("key_id", id).Info("API key updated successfully")

	publicKey := apiKey.ToPublic()
	return &publicKey, nil
}

// DeleteAPIKey удаляет API ключ
func (m *Manager) DeleteAPIKey(ctx context.Context, id string) error {
	m.logger.WithField("key_id", id).Info("Deleting API key")

	if err := m.storage.DeleteAPIKey(ctx, id); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	// Invalidate cache (v3.0.6+)
	if m.invalidator != nil {
		if err := m.invalidator.InvalidateAPIKey(ctx, id); err != nil {
			m.logger.WithError(err).Warn("Failed to invalidate API key cache")
		}
	}

	m.stats.TotalKeys--
	m.stats.ActiveKeys--

	m.logger.WithField("key_id", id).Info("API key deleted successfully")
	return nil
}

// RevokeAPIKey отзывает API ключ
func (m *Manager) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	m.logger.WithFields(logrus.Fields{
		"key_id": id,
		"reason": reason,
	}).Info("Revoking API key")

	if err := m.storage.RevokeAPIKey(ctx, id, reason); err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	// Invalidate cache (v3.0.6+)
	if m.invalidator != nil {
		if err := m.invalidator.InvalidateAPIKey(ctx, id); err != nil {
			m.logger.WithError(err).Warn("Failed to invalidate API key cache")
		}
	}

	m.stats.ActiveKeys--

	return nil
}

// EnableAPIKey активирует отозванный API ключ (AUTH-04)
func (m *Manager) EnableAPIKey(ctx context.Context, id string) error {
	m.logger.WithField("key_id", id).Info("Enabling API key")

	if err := m.storage.EnableAPIKey(ctx, id); err != nil {
		return fmt.Errorf("failed to enable API key: %w", err)
	}

	// Invalidate cache (v3.0.6+)
	if m.invalidator != nil {
		if err := m.invalidator.InvalidateAPIKey(ctx, id); err != nil {
			m.logger.WithError(err).Warn("Failed to invalidate API key cache")
		}
	}

	m.stats.ActiveKeys++

	m.logger.WithField("key_id", id).Info("API key enabled successfully")
	return nil
}

// ExtendExpiration продлевает срок действия API ключа (AUTH-04)
func (m *Manager) ExtendExpiration(ctx context.Context, id string, days int) (*models.APIKeyPublic, error) {
	m.logger.WithFields(logrus.Fields{
		"key_id": id,
		"days":   days,
	}).Info("Extending API key expiration")

	// Получаем существующий ключ
	apiKey, err := m.storage.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	// Вычисляем новую дату истечения
	newExpiration := time.Now().AddDate(0, 0, days)
	apiKey.ExpiresAt = &newExpiration
	apiKey.UpdatedAt = time.Now()

	// Если ключ был expired, делаем его active
	if apiKey.Status == models.APIKeyStatusExpired {
		apiKey.Status = models.APIKeyStatusActive
	}

	// Сохраняем обновления
	if err := m.storage.UpdateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"key_id":     id,
		"expires_at": newExpiration.Format(time.RFC3339),
	}).Info("API key expiration extended successfully")

	publicKey := apiKey.ToPublic()
	return &publicKey, nil
}

// UpdateAPIKeyPermissions обновляет permissions и models для ключа (AUTH-04)
func (m *Manager) UpdateAPIKeyPermissions(ctx context.Context, id string, models []string, permissions []string) (*models.APIKeyPublic, error) {
	m.logger.WithFields(logrus.Fields{
		"key_id":      id,
		"models":      models,
		"permissions": permissions,
	}).Info("Updating API key permissions")

	// Получаем существующий ключ
	apiKey, err := m.storage.GetAPIKey(ctx, id)
	if err != nil {
		return nil, err
	}

	// Обновляем permissions и models
	if models != nil {
		apiKey.Models = models
	}
	if permissions != nil {
		apiKey.Permissions = permissions
	}
	apiKey.UpdatedAt = time.Now()

	// Сохраняем обновления
	if err := m.storage.UpdateAPIKey(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to update API key permissions: %w", err)
	}

	m.logger.WithField("key_id", id).Info("API key permissions updated successfully")

	publicKey := apiKey.ToPublic()
	return &publicKey, nil
}

// CheckModelAccess проверяет доступ API ключа к модели
func (m *Manager) CheckModelAccess(ctx context.Context, keyID, model string) (bool, error) {
	apiKey, err := m.storage.GetAPIKey(ctx, keyID)
	if err != nil {
		return false, err
	}

	if !apiKey.IsActive() {
		return false, fmt.Errorf("API key is not active")
	}

	return apiKey.HasModelAccess(model), nil
}

// RecordUsage записывает использование API ключа
func (m *Manager) RecordUsage(ctx context.Context, keyID, model, endpoint string, tokens int64, success bool) error {
	apiKey, err := m.storage.GetAPIKey(ctx, keyID)
	if err != nil {
		return err
	}

	// Обновляем статистику использования
	apiKey.IncrementUsage(model, endpoint, tokens, success)

	// Сохраняем обновления
	if err := m.storage.UpdateAPIKey(ctx, apiKey); err != nil {
		m.logger.WithError(err).Warn("Failed to update API key usage")
		return err
	}

	// Обновляем общую статистику
	m.stats.TotalRequestsToday++
	if success {
		m.stats.KeysUsedToday++
	}

	return nil
}

// GetStats возвращает статистику Manager'а
func (m *Manager) GetStats(ctx context.Context) (ManagerStats, error) {
	if err := m.updateStats(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to update stats")
	}

	return m.stats, nil
}

// PerformMaintenance выполняет регулярное обслуживание
func (m *Manager) PerformMaintenance(ctx context.Context) error {
	m.logger.Info("Performing API Key Manager maintenance")

	// Очистка истекших ключей
	cleaned, err := m.storage.CleanupExpiredKeys(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Failed to cleanup expired keys")
	} else if cleaned > 0 {
		m.logger.WithField("cleaned", cleaned).Info("Cleaned up expired keys")
	}

	// Обновление статистики
	if err := m.updateStats(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to update stats during maintenance")
	}

	m.stats.LastCleanup = time.Now()

	return nil
}

// HealthCheck проверяет здоровье API Key Manager
func (m *Manager) HealthCheck(ctx context.Context) error {
	// Проверяем storage
	if err := m.storage.HealthCheck(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	// Проверяем что можем прочитать ключи
	_, _, err := m.storage.ListAPIKeys(ctx, models.ListAPIKeysRequest{Limit: 1})
	if err != nil {
		return fmt.Errorf("failed to test storage access: %w", err)
	}

	return nil
}

// Close закрывает Manager и освобождает ресурсы
func (m *Manager) Close() error {
	m.logger.Info("Closing API Key Manager")
	return m.storage.Close()
}

// Private methods

// updateStats обновляет статистику manager'а
func (m *Manager) updateStats(ctx context.Context) error {
	// Получаем статистику storage
	storageStats, err := m.storage.GetStorageStats(ctx)
	if err != nil {
		return err
	}

	// Обновляем статистику
	if totalKeys, ok := storageStats["total_keys"].(int); ok {
		m.stats.TotalKeys = int64(totalKeys)
	}
	if activeKeys, ok := storageStats["active_keys"].(int); ok {
		m.stats.ActiveKeys = int64(activeKeys)
	}

	return nil
}

// isNotFoundError проверяет является ли ошибка "не найдено"
func isNotFoundError(err error) bool {
	if storageErr, ok := err.(*storage.StorageError); ok {
		return storageErr.Type == storage.StorageErrorTypeNotFound
	}
	return false
}

// ValidateKeyAccess валидирует доступ ключа к операции
func (m *Manager) ValidateKeyAccess(ctx context.Context, keyID, operation, model string) error {
	apiKey, err := m.storage.GetAPIKey(ctx, keyID)
	if err != nil {
		return err
	}

	// Проверяем статус ключа
	if !apiKey.IsActive() {
		return fmt.Errorf("API key is not active (status: %s)", apiKey.Status)
	}

	// Проверяем разрешения
	switch operation {
	case "chat":
		if !apiKey.HasPermission("chat") && !apiKey.HasPermission("*") {
			return fmt.Errorf("API key does not have chat permission")
		}
	case "models":
		if !apiKey.HasPermission("models") && !apiKey.HasPermission("*") {
			return fmt.Errorf("API key does not have models permission")
		}
	case "admin":
		if !apiKey.HasPermission("admin") && !apiKey.HasPermission("*") {
			return fmt.Errorf("API key does not have admin permission")
		}
	}

	// Проверяем доступ к модели
	if model != "" && !apiKey.HasModelAccess(model) {
		return fmt.Errorf("API key does not have access to model: %s", model)
	}

	return nil
}

// GetAPIKeysByModel получает ключи с доступом к модели
func (m *Manager) GetAPIKeysByModel(ctx context.Context, model string) ([]models.APIKeyPublic, error) {
	apiKeys, err := m.storage.FindAPIKeysByModel(ctx, model)
	if err != nil {
		return nil, fmt.Errorf("failed to find API keys by model: %w", err)
	}

	result := make([]models.APIKeyPublic, len(apiKeys))
	for i, key := range apiKeys {
		result[i] = key.ToPublic()
	}

	return result, nil
}

// GetUsageReport получает отчет об использовании ключа
func (m *Manager) GetUsageReport(ctx context.Context, keyID string, days int) (*UsageReport, error) {
	apiKey, err := m.storage.GetAPIKey(ctx, keyID)
	if err != nil {
		return nil, err
	}

	report := &UsageReport{
		KeyID:          keyID,
		KeyName:        apiKey.Name,
		TotalRequests:  apiKey.Usage.TotalRequests,
		TotalTokens:    apiKey.Usage.TotalTokens,
		SuccessRate:    calculateSuccessRate(apiKey.Usage),
		ModelUsage:     apiKey.Usage.ModelUsage,
		EndpointUsage:  apiKey.Usage.EndpointUsage,
		RecentActivity: getRecentActivity(apiKey.Usage.DailyUsage, days),
	}

	return report, nil
}

// UsageReport представляет отчет об использовании API ключа
type UsageReport struct {
	KeyID          string            `json:"key_id"`
	KeyName        string            `json:"key_name"`
	TotalRequests  int64             `json:"total_requests"`
	TotalTokens    int64             `json:"total_tokens"`
	SuccessRate    float64           `json:"success_rate"`
	ModelUsage     map[string]int64  `json:"model_usage"`
	EndpointUsage  map[string]int64  `json:"endpoint_usage"`
	RecentActivity []models.DayUsage `json:"recent_activity"`
}

// calculateSuccessRate подсчитывает процент успешных запросов
func calculateSuccessRate(usage models.APIKeyUsage) float64 {
	if usage.TotalRequests == 0 {
		return 0
	}
	return float64(usage.SuccessfulRequests) / float64(usage.TotalRequests) * 100
}

// getRecentActivity получает активность за последние N дней
func getRecentActivity(dailyUsage map[string]models.DayUsage, days int) []models.DayUsage {
	if days <= 0 {
		days = 7
	}

	var activity []models.DayUsage
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i).Format("2006-01-02")
		if usage, exists := dailyUsage[date]; exists {
			activity = append(activity, usage)
		} else {
			activity = append(activity, models.DayUsage{
				Date:     date,
				Requests: 0,
				Tokens:   0,
			})
		}
	}

	// Сортируем по дате (новые сначала)
	return activity
}

// createBootstrapAdminKey создает первый admin ключ если его нет
func (m *Manager) createBootstrapAdminKey(ctx context.Context) error {
	// Проверяем есть ли admin ключи
	adminKeys, err := m.storage.FindAPIKeysByStatus(ctx, models.APIKeyStatusActive)
	if err != nil {
		return fmt.Errorf("failed to check existing admin keys: %w", err)
	}

	// Если есть активные ключи с admin правами, ничего не делаем
	for _, key := range adminKeys {
		if key.HasPermission("admin") {
			m.logger.Debug("Admin key already exists, skipping bootstrap")
			return nil
		}
	}

	// Создаем bootstrap admin ключ
	adminKeyReq := models.CreateAPIKeyRequest{
		Name:        "Bootstrap Admin Key",
		Description: "Initial admin key created during system initialization",
		Models:      []string{"*"},
		Permissions: []string{"*"}, // Все разрешения
		RateLimits: &models.RateLimits{
			RequestsPerMinute: 1000, // Высокие лимиты для admin
			RequestsPerHour:   10000,
			RequestsPerDay:    100000,
			TokensPerMinute:   100000,
			TokensPerDay:      1000000,
		},
	}

	adminKey, plainKey, err := models.NewAPIKey(adminKeyReq)
	if err != nil {
		return fmt.Errorf("failed to create bootstrap admin key: %w", err)
	}

	// Устанавливаем предсказуемый ключ для конфигурации
	if m.config.Auth.AdminKey != "" {
		// Хешируем admin ключ из конфигурации
		hash, err := models.HashAPIKey(m.config.Auth.AdminKey)
		if err != nil {
			return fmt.Errorf("failed to hash admin key from config: %w", err)
		}
		adminKey.KeyHash = hash
		plainKey = m.config.Auth.AdminKey
	}

	// Сохраняем bootstrap ключ
	if err := m.storage.CreateAPIKey(ctx, adminKey); err != nil {
		return fmt.Errorf("failed to save bootstrap admin key: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"key_id":    adminKey.ID,
		"key_name":  adminKey.Name,
		"plain_key": plainKey,
	}).Info("Bootstrap admin key created - SAVE THIS KEY!")

	fmt.Printf("🔑 BOOTSTRAP ADMIN KEY CREATED:\n")
	fmt.Printf("   Key ID: %s\n", adminKey.ID)
	fmt.Printf("   Plain Key: %s\n", plainKey)
	fmt.Printf("   ⚠️  SAVE THIS KEY - IT WILL NOT BE SHOWN AGAIN!\n\n")

	return nil
}

