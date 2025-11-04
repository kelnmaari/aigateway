// Package models provides data models for API Key Management
package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"sync/atomic"
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

	// Device metadata (Version 2.4.0+: Desktop Client Support)
	DeviceName        *string    `json:"device_name,omitempty" db:"device_name"`               // User-friendly device name
	DeviceOS          *string    `json:"device_os,omitempty" db:"device_os"`                   // OS: windows, darwin, linux
	DeviceHostname    *string    `json:"device_hostname,omitempty" db:"device_hostname"`       // System hostname
	DeviceVersion     *string    `json:"device_version,omitempty" db:"device_version"`         // Desktop app version
	DeviceFingerprint *string    `json:"device_fingerprint,omitempty" db:"device_fingerprint"` // SHA256 hash for unique ID
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`             // Last API request timestamp
	AutoExpireAt      *time.Time `json:"auto_expire_at,omitempty" db:"auto_expire_at"`         // Auto-expiry for device keys

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
	UserID        *string                `json:"user_id,omitempty"`   // Owner user ID (Version 1.3.0+)
	TenantID      *string                `json:"tenant_id,omitempty"` // Tenant ID (Version 1.3.0+)
	Scope         APIKeyScope            `json:"scope,omitempty"`     // personal or tenant (Version 1.3.0+)
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

	// Device metadata (Version 2.4.0+: Desktop Client Support)
	DeviceName        *string    `json:"device_name,omitempty"`
	DeviceOS          *string    `json:"device_os,omitempty"`
	DeviceHostname    *string    `json:"device_hostname,omitempty"`
	DeviceVersion     *string    `json:"device_version,omitempty"`
	DeviceFingerprint *string    `json:"device_fingerprint,omitempty"`
	LastSeenAt        *time.Time `json:"last_seen_at,omitempty"`
	AutoExpireAt      *time.Time `json:"auto_expire_at,omitempty"`

	// Enriched fields for admin display (not stored in DB)
	OwnerUsername string `json:"owner_username,omitempty"` // Username of owner (Version 1.3.0+)
	TenantName    string `json:"tenant_name,omitempty"`    // Name of tenant/organization (Version 1.3.0+)
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
	return slices.Contains(k.Models, "*") || slices.Contains(k.Models, model)
}

// HasPermission проверяет наличие разрешения
func (k *APIKey) HasPermission(permission string) bool {
	return slices.Contains(k.Permissions, "*") || slices.Contains(k.Permissions, permission)
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
// Version 1.7.0: Fixed race condition - использует atomic operations для счетчиков
//
// ВАЖНО: Map operations (ModelUsage, EndpointUsage, DailyUsage) НЕ thread-safe!
// Для production с высокой конкурентностью используйте APIKeyUsageHot из apikey_optimized.go
func (k *APIKey) IncrementUsage(model, endpoint string, tokens int64, success bool) {
	// Atomic operations для основных счетчиков (CRITICAL FIX)
	atomic.AddInt64(&k.Usage.TotalRequests, 1)
	atomic.AddInt64(&k.Usage.TotalTokens, tokens)

	if success {
		atomic.AddInt64(&k.Usage.SuccessfulRequests, 1)
	} else {
		atomic.AddInt64(&k.Usage.FailedRequests, 1)
	}

	now := time.Now()
	k.Usage.LastRequestAt = &now

	// NOTE: Map operations ниже НЕ thread-safe и могут вызвать race condition
	// при высокой конкурентности. Для production используйте sync.Mutex или
	// мигрируйте на APIKeyUsageHot с отдельным холодным хранилищем для maps.

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
	return slices.Contains(k.Permissions, "*") || slices.Contains(k.Permissions, permission)
}

// HasModelAccess проверяет доступ к модели у публичного ключа
func (k *APIKeyPublic) HasModelAccess(model string) bool {
	// Если доступ ко всем моделям
	return slices.Contains(k.Models, "*") || slices.Contains(k.Models, model)
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

// ========================================
// Device Registration (Version 2.4.0+)
// ========================================

// DeviceRegistrationRequest represents device registration request from desktop client
type DeviceRegistrationRequest struct {
	DeviceName        string `json:"device_name"`                                             // User-friendly name (optional, auto-generated if empty)
	DeviceOS          string `json:"device_os" binding:"required,oneof=windows darwin linux"` // Operating system
	DeviceHostname    string `json:"device_hostname"`                                         // System hostname
	DeviceVersion     string `json:"device_version"`                                          // Desktop app version
	DeviceFingerprint string `json:"device_fingerprint" binding:"required,min=32"`            // SHA256 hash for unique identification
	AutoExpireDays    int    `json:"auto_expire_days,omitempty"`                              // Auto-expiry in days (default: 90)
}

// DeviceRegistrationResponse represents device registration response
type DeviceRegistrationResponse struct {
	APIKey      string     `json:"api_key"`                // Plain API key (returned only once!)
	KeyID       string     `json:"key_id"`                 // Key ID
	DeviceName  string     `json:"device_name"`            // Registered device name
	DeviceOS    string     `json:"device_os"`              // Operating system
	CreatedAt   time.Time  `json:"created_at"`             // Creation timestamp
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`   // Expiration date (if auto-expire enabled)
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"` // Last seen timestamp (for existing devices)
	IsNewDevice bool       `json:"is_new_device"`          // True if new device, false if existing
	Message     string     `json:"message,omitempty"`      // Optional message
}

// Validate validates DeviceRegistrationRequest
func (r *DeviceRegistrationRequest) Validate() error {
	// DeviceOS validation
	validOS := map[string]bool{
		"windows": true,
		"darwin":  true,
		"linux":   true,
	}
	if !validOS[r.DeviceOS] {
		return fmt.Errorf("invalid device_os: %s (must be windows, darwin, or linux)", r.DeviceOS)
	}

	// DeviceFingerprint validation
	if len(r.DeviceFingerprint) < 32 {
		return fmt.Errorf("device_fingerprint too short (minimum 32 characters)")
	}
	if len(r.DeviceFingerprint) > 128 {
		return fmt.Errorf("device_fingerprint too long (maximum 128 characters)")
	}

	// DeviceName validation (optional)
	if len(r.DeviceName) > 100 {
		return fmt.Errorf("device_name too long (maximum 100 characters)")
	}

	// AutoExpireDays validation
	if r.AutoExpireDays < 0 {
		return fmt.Errorf("auto_expire_days cannot be negative")
	}
	if r.AutoExpireDays > 365 {
		return fmt.Errorf("auto_expire_days cannot exceed 365 days")
	}

	return nil
}

// GenerateDeviceName generates auto device name if not provided
func GenerateDeviceName(os, hostname string) string {
	osName := map[string]string{
		"windows": "Windows",
		"darwin":  "macOS",
		"linux":   "Linux",
	}[os]

	if osName == "" {
		osName = "Unknown"
	}

	if hostname != "" {
		return fmt.Sprintf("Desktop (%s) - %s", osName, hostname)
	}

	return fmt.Sprintf("Desktop %s - %s", osName, time.Now().Format("Jan 02"))
}

// IsDeviceKey checks if API key is a device key
func (k *APIKey) IsDeviceKey() bool {
	return k.DeviceFingerprint != nil && *k.DeviceFingerprint != ""
}

// IsDeviceKey checks if public API key is a device key
func (k *APIKeyPublic) IsDeviceKey() bool {
	return k.DeviceFingerprint != nil && *k.DeviceFingerprint != ""
}

// ========================================
// Device Management (Version 2.4.2+)
// ========================================

// DeviceInfo represents device information for management API
type DeviceInfo struct {
	ID                string       `json:"id"`                 // API Key ID
	DeviceName        string       `json:"device_name"`        // User-friendly name
	DeviceOS          string       `json:"device_os"`          // windows, darwin, linux
	DeviceHostname    string       `json:"device_hostname"`    // System hostname
	DeviceVersion     string       `json:"device_version"`     // App version
	DeviceFingerprint string       `json:"device_fingerprint"` // SHA256 hash (first 16 chars for display)
	CreatedAt         time.Time    `json:"created_at"`         // Registration date
	LastSeenAt        *time.Time   `json:"last_seen_at"`       // Last API request
	ExpiresAt         *time.Time   `json:"expires_at"`         // Auto-expiry date
	IsCurrentDevice   bool         `json:"is_current_device"`  // Is this the device making request?
	Status            string       `json:"status"`             // active, expired, revoked
	Usage             *APIKeyUsage `json:"usage,omitempty"`    // Usage statistics (optional, for detailed view)
}

// ListDevicesResponse represents response for listing devices
type ListDevicesResponse struct {
	Devices      []DeviceInfo `json:"devices"`
	Total        int          `json:"total"`
	CurrentCount int          `json:"current_count"` // Active devices
}

// UpdateDeviceNameRequest represents request for updating device name
type UpdateDeviceNameRequest struct {
	DeviceName string `json:"device_name" binding:"required,min=1,max=100"`
}

// DeviceFilters represents filters for listing devices
type DeviceFilters struct {
	Status string // active, expired, all
	Sort   string // last_seen, created_at, name
	Order  string // asc, desc
}

// ToDeviceInfo converts APIKey to DeviceInfo
func (k *APIKey) ToDeviceInfo(currentKeyID string) DeviceInfo {
	info := DeviceInfo{
		ID:              k.ID,
		DeviceName:      safeStringDeref(k.DeviceName),
		DeviceOS:        safeStringDeref(k.DeviceOS),
		DeviceHostname:  safeStringDeref(k.DeviceHostname),
		DeviceVersion:   safeStringDeref(k.DeviceVersion),
		CreatedAt:       k.CreatedAt,
		LastSeenAt:      k.LastSeenAt,
		ExpiresAt:       k.AutoExpireAt,
		IsCurrentDevice: k.ID == currentKeyID,
		Status:          string(k.Status),
	}

	// Show only first 16 chars of fingerprint for display
	if k.DeviceFingerprint != nil && len(*k.DeviceFingerprint) > 16 {
		info.DeviceFingerprint = (*k.DeviceFingerprint)[:16]
	} else if k.DeviceFingerprint != nil {
		info.DeviceFingerprint = *k.DeviceFingerprint
	}

	// Determine status
	if k.IsExpired() {
		info.Status = "expired"
	} else if k.Status == APIKeyStatusRevoked {
		info.Status = "revoked"
	} else if k.Status == APIKeyStatusActive {
		info.Status = "active"
	}

	return info
}

// ToDeviceInfoWithUsage converts APIKey to DeviceInfo with usage statistics
func (k *APIKey) ToDeviceInfoWithUsage(currentKeyID string) DeviceInfo {
	info := k.ToDeviceInfo(currentKeyID)
	usage := k.Usage
	info.Usage = &usage
	return info
}

// safeStringDeref safely dereferences string pointer
func safeStringDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
