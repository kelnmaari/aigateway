// Package models provides data models for API Key Management
package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// APIKey представляет API ключ с метаданными и правами доступа
type APIKey struct {
	// Основная информация
	ID          string `json:"id"`                    // Уникальный ID ключа
	Name        string `json:"name"`                  // Человекочитаемое имя
	Description string `json:"description,omitempty"` // Описание назначения
	KeyHash     string `json:"key_hash"`              // Хеш ключа (bcrypt)

	// Владение (Version 1.3.0+: Multi-Tenancy support)
	UserID   *string     `json:"user_id,omitempty" db:"user_id"`     // ID владельца (User.ID) - для personal keys
	TenantID *string     `json:"tenant_id,omitempty" db:"tenant_id"` // ID tenant - для organization keys
	Scope    APIKeyScope `json:"scope" db:"scope"`                   // personal или tenant

	// Права доступа
	Models      []string `json:"models"`      // Разрешенные модели (* для всех)
	Permissions []string `json:"permissions"` // Разрешения (chat, models, admin)

	// Rate limiting
	RateLimits RateLimits `json:"rate_limits"`

	// Временные метки
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"` // nil = не истекает
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`

	// Статус и метаданные
	Status        APIKeyStatus           `json:"status"`
	RevokedAt     *time.Time             `json:"revoked_at,omitempty"`     // Когда был отозван
	RevokedReason string                 `json:"revoked_reason,omitempty"` // Причина отзыва
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Статистика использования
	Usage APIKeyUsage `json:"usage"`
}

// APIKeyStatus представляет статус API ключа
type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusDisabled APIKeyStatus = "disabled"
	APIKeyStatusExpired  APIKeyStatus = "expired"
	APIKeyStatusRevoked  APIKeyStatus = "revoked"
)

// APIKeyScope представляет область видимости API ключа (Version 1.3.0+)
type APIKeyScope string

const (
	APIKeyScopeGlobal   APIKeyScope = "global"   // Глобальный ключ (backward compatibility, no user/tenant)
	APIKeyScopePersonal APIKeyScope = "personal" // Личный ключ пользователя (UserID set)
	APIKeyScopeTenant   APIKeyScope = "tenant"   // Ключ организации (TenantID set)
)

// RateLimits определяет лимиты запросов для API ключа
type RateLimits struct {
	RequestsPerMinute int `json:"requests_per_minute"` // Запросов в минуту
	RequestsPerHour   int `json:"requests_per_hour"`   // Запросов в час
	RequestsPerDay    int `json:"requests_per_day"`    // Запросов в день
	TokensPerMinute   int `json:"tokens_per_minute"`   // Токенов в минуту
	TokensPerDay      int `json:"tokens_per_day"`      // Токенов в день
}

// APIKeyUsage содержит статистику использования ключа
type APIKeyUsage struct {
	TotalRequests      int64               `json:"total_requests"`
	SuccessfulRequests int64               `json:"successful_requests"`
	FailedRequests     int64               `json:"failed_requests"`
	TotalTokens        int64               `json:"total_tokens"`
	LastRequestAt      *time.Time          `json:"last_request_at,omitempty"`
	ModelUsage         map[string]int64    `json:"model_usage,omitempty"`    // Использование по моделям
	EndpointUsage      map[string]int64    `json:"endpoint_usage,omitempty"` // Использование по эндпоинтам
	DailyUsage         map[string]DayUsage `json:"daily_usage,omitempty"`    // Статистика по дням
}

// DayUsage содержит статистику за день
type DayUsage struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// CreateAPIKeyRequest представляет запрос на создание API ключа
type CreateAPIKeyRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description,omitempty"`
	Models      []string               `json:"models,omitempty"`      // ["*"] для всех моделей
	Permissions []string               `json:"permissions,omitempty"` // ["chat", "models"]
	RateLimits  *RateLimits            `json:"rate_limits,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateAPIKeyResponse представляет ответ на создание API ключа
type CreateAPIKeyResponse struct {
	APIKey   APIKeyPublic `json:"api_key"`
	PlainKey string       `json:"plain_key"` // Отдается только один раз!
	Warning  string       `json:"warning,omitempty"`
}

// APIKeyPublic представляет публичную информацию об API ключе (без хеша)
type APIKeyPublic struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Models        []string               `json:"models"`
	Permissions   []string               `json:"permissions"`
	RateLimits    RateLimits             `json:"rate_limits"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ExpiresAt     *time.Time             `json:"expires_at,omitempty"`
	LastUsedAt    *time.Time             `json:"last_used_at,omitempty"`
	Status        APIKeyStatus           `json:"status"`
	RevokedAt     *time.Time             `json:"revoked_at,omitempty"`
	RevokedReason string                 `json:"revoked_reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Usage         APIKeyUsage            `json:"usage"`
}

// UpdateAPIKeyRequest представляет запрос на обновление API ключа
type UpdateAPIKeyRequest struct {
	Name        *string                 `json:"name,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Models      *[]string               `json:"models,omitempty"`
	Permissions *[]string               `json:"permissions,omitempty"`
	RateLimits  *RateLimits             `json:"rate_limits,omitempty"`
	Status      *APIKeyStatus           `json:"status,omitempty"`
	ExpiresAt   *time.Time              `json:"expires_at,omitempty"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

// ListAPIKeysRequest представляет запрос на получение списка ключей
type ListAPIKeysRequest struct {
	Status    *APIKeyStatus `form:"status"`     // Фильтр по статусу
	Model     *string       `form:"model"`      // Фильтр по модели
	Limit     int           `form:"limit"`      // Лимит результатов
	Offset    int           `form:"offset"`     // Смещение для пагинации
	SortBy    string        `form:"sort_by"`    // Сортировка (created_at, name, last_used)
	SortOrder string        `form:"sort_order"` // asc/desc
}

// ListAPIKeysResponse представляет ответ со списком ключей
type ListAPIKeysResponse struct {
	APIKeys []APIKeyPublic `json:"api_keys"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
}

// APIKeyValidationResult результат валидации API ключа
type APIKeyValidationResult struct {
	Valid       bool          `json:"valid"`
	APIKey      *APIKeyPublic `json:"api_key,omitempty"`
	Error       string        `json:"error,omitempty"`
	ModelAccess bool          `json:"model_access"`
	RateLimited bool          `json:"rate_limited"`
}

// Методы для APIKey

// ToPublic конвертирует APIKey в публичную версию (без хеша)
func (k *APIKey) ToPublic() APIKeyPublic {
	return APIKeyPublic{
		ID:            k.ID,
		Name:          k.Name,
		Description:   k.Description,
		Models:        k.Models,
		Permissions:   k.Permissions,
		RateLimits:    k.RateLimits,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
		ExpiresAt:     k.ExpiresAt,
		LastUsedAt:    k.LastUsedAt,
		Status:        k.Status,
		RevokedAt:     k.RevokedAt,
		RevokedReason: k.RevokedReason,
		Metadata:      k.Metadata,
		Usage:         k.Usage,
	}
}

// IsExpired проверяет истек ли ключ
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive проверяет активен ли ключ
func (k *APIKey) IsActive() bool {
	return k.Status == APIKeyStatusActive && !k.IsExpired()
}

// HasModelAccess проверяет доступ к модели
func (k *APIKey) HasModelAccess(model string) bool {
	// Если доступ ко всем моделям
	for _, allowedModel := range k.Models {
		if allowedModel == "*" {
			return true
		}
		if allowedModel == model {
			return true
		}
	}
	return false
}

// HasPermission проверяет наличие разрешения
func (k *APIKey) HasPermission(permission string) bool {
	for _, perm := range k.Permissions {
		if perm == "*" || perm == permission {
			return true
		}
	}
	return false
}

// VerifyKey проверяет plaintext ключ против хеша
func (k *APIKey) VerifyKey(plainKey string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(plainKey))
	return err == nil
}

// UpdateLastUsed обновляет время последнего использования
func (k *APIKey) UpdateLastUsed() {
	now := time.Now()
	k.LastUsedAt = &now
	k.UpdatedAt = now
}

// IncrementUsage увеличивает счетчики использования
func (k *APIKey) IncrementUsage(model, endpoint string, tokens int64, success bool) {
	if success {
		k.Usage.SuccessfulRequests++
	} else {
		k.Usage.FailedRequests++
	}

	k.Usage.TotalRequests++
	k.Usage.TotalTokens += tokens

	now := time.Now()
	k.Usage.LastRequestAt = &now

	// Статистика по моделям
	if k.Usage.ModelUsage == nil {
		k.Usage.ModelUsage = make(map[string]int64)
	}
	k.Usage.ModelUsage[model]++

	// Статистика по эндпоинтам
	if k.Usage.EndpointUsage == nil {
		k.Usage.EndpointUsage = make(map[string]int64)
	}
	k.Usage.EndpointUsage[endpoint]++

	// Статистика по дням
	if k.Usage.DailyUsage == nil {
		k.Usage.DailyUsage = make(map[string]DayUsage)
	}

	dateKey := now.Format("2006-01-02")
	dayUsage := k.Usage.DailyUsage[dateKey]
	dayUsage.Date = dateKey
	dayUsage.Requests++
	dayUsage.Tokens += tokens
	k.Usage.DailyUsage[dateKey] = dayUsage

	k.UpdatedAt = now
}

// Вспомогательные функции

// GenerateAPIKey генерирует новый безопасный API ключ
func GenerateAPIKey() (string, error) {
	// Генерируем 32 байта случайных данных
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Формируем ключ в стиле OpenAI: sk-proj-<hex>
	key := "sk-proj-" + hex.EncodeToString(bytes)
	return key, nil
}

// GenerateAPIKeyWithID генерирует API ключ с встроенным key_id
// Формат: sk-<key_id>-<random_suffix>
func GenerateAPIKeyWithID(keyID string) (string, error) {
	// Генерируем 16 байт случайных данных для суффикса
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Формируем ключ: sk-<key_id>-<hex_suffix>
	// Пример: sk-ak_1728000000_1a2b3c4d-f8e7d6c5b4a39281
	key := fmt.Sprintf("sk-%s-%s", keyID, hex.EncodeToString(bytes))
	return key, nil
}

// HashAPIKey хеширует API ключ для безопасного хранения
func HashAPIKey(plainKey string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash API key: %w", err)
	}
	return string(hash), nil
}

// GenerateAPIKeyID генерирует уникальный ID для API ключа
func GenerateAPIKeyID() string {
	return fmt.Sprintf("ak_%d_%s", time.Now().Unix(), generateShortID())
}

// generateShortID генерирует короткий ID
func generateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// NewAPIKey создает новый API ключ
func NewAPIKey(req CreateAPIKeyRequest) (*APIKey, string, error) {
	// Генерируем key_id сначала
	keyID := GenerateAPIKeyID()

	// Генерируем ключ с key_id
	plainKey, err := GenerateAPIKeyWithID(keyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Хешируем ключ
	keyHash, err := HashAPIKey(plainKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	// Устанавливаем defaults
	models := req.Models
	if len(models) == 0 {
		models = []string{"*"} // Доступ ко всем моделям по умолчанию
	}

	permissions := req.Permissions
	if len(permissions) == 0 {
		permissions = []string{"chat", "models"} // Базовые разрешения
	}

	var rateLimits RateLimits
	if req.RateLimits != nil {
		rateLimits = *req.RateLimits
	} else {
		// Default rate limits
		rateLimits = RateLimits{
			RequestsPerMinute: 30,
			RequestsPerHour:   500,
			RequestsPerDay:    5000,
			TokensPerMinute:   10000,
			TokensPerDay:      100000,
		}
	}

	now := time.Now()

	apiKey := &APIKey{
		ID:          keyID, // Используем тот же ID что встроен в ключ
		Name:        req.Name,
		Description: req.Description,
		KeyHash:     keyHash,
		Models:      models,
		Permissions: permissions,
		RateLimits:  rateLimits,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   req.ExpiresAt,
		Status:      APIKeyStatusActive,
		Metadata:    req.Metadata,
		Usage: APIKeyUsage{
			ModelUsage:    make(map[string]int64),
			EndpointUsage: make(map[string]int64),
			DailyUsage:    make(map[string]DayUsage),
		},
	}

	return apiKey, plainKey, nil
}

// Validate валидирует CreateAPIKeyRequest
func (r *CreateAPIKeyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	if len(r.Name) > 100 {
		return fmt.Errorf("name too long (max 100 characters)")
	}

	if len(r.Description) > 500 {
		return fmt.Errorf("description too long (max 500 characters)")
	}

	// Валидация permissions
	validPermissions := map[string]bool{
		"chat":       true,
		"models":     true,
		"admin":      true,
		"embeddings": true,
		"*":          true,
	}

	for _, perm := range r.Permissions {
		if !validPermissions[perm] {
			return fmt.Errorf("invalid permission: %s", perm)
		}
	}

	// Валидация rate limits
	if r.RateLimits != nil {
		if r.RateLimits.RequestsPerMinute < 0 || r.RateLimits.RequestsPerMinute > 1000 {
			return fmt.Errorf("requests_per_minute must be between 0 and 1000")
		}
		if r.RateLimits.RequestsPerHour < 0 || r.RateLimits.RequestsPerHour > 10000 {
			return fmt.Errorf("requests_per_hour must be between 0 and 10000")
		}
	}

	// Валидация expiration
	if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expiration date cannot be in the past")
	}

	return nil
}

// GetDefaultRateLimits возвращает rate limits по умолчанию
func GetDefaultRateLimits() RateLimits {
	return RateLimits{
		RequestsPerMinute: 30,
		RequestsPerHour:   500,
		RequestsPerDay:    5000,
		TokensPerMinute:   10000,
		TokensPerDay:      100000,
	}
}

// GetDefaultPermissions возвращает разрешения по умолчанию
func GetDefaultPermissions() []string {
	return []string{"chat", "models"}
}

// ParseAPIKeyFromHeader извлекает API ключ из Authorization заголовка
func ParseAPIKeyFromHeader(authHeader string) string {
	// Bearer sk-proj-...
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}

// IsValidAPIKeyFormat проверяет формат API ключа
func IsValidAPIKeyFormat(key string) bool {
	// Должен начинаться с sk-proj- и иметь достаточную длину
	return len(key) > 15 && (key[:8] == "sk-proj-" || key[:3] == "sk-")
}

// Методы для APIKeyPublic

// HasPermission проверяет наличие разрешения у публичного ключа
func (k *APIKeyPublic) HasPermission(permission string) bool {
	for _, perm := range k.Permissions {
		if perm == "*" || perm == permission {
			return true
		}
	}
	return false
}

// HasModelAccess проверяет доступ к модели у публичного ключа
func (k *APIKeyPublic) HasModelAccess(model string) bool {
	// Если доступ ко всем моделям
	for _, allowedModel := range k.Models {
		if allowedModel == "*" {
			return true
		}
		if allowedModel == model {
			return true
		}
	}
	return false
}

// IsExpired проверяет истек ли публичный ключ
func (k *APIKeyPublic) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsActive проверяет активен ли публичный ключ
func (k *APIKeyPublic) IsActive() bool {
	return k.Status == APIKeyStatusActive && !k.IsExpired()
}
