// Package models provides quota data models
// Version: 1.11.7+ (Enterprise Suite - Usage Quotas System)
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// QuotaScope определяет scope квоты
type QuotaScope string

const (
	QuotaScopeUser   QuotaScope = "user"   // Per-user quota
	QuotaScopeTenant QuotaScope = "tenant" // Per-tenant quota
)

// Quota представляет лимиты использования для user или tenant
type Quota struct {
	ID       string     `json:"id" db:"id"`
	Name     string     `json:"name" db:"name"`
	Scope    QuotaScope `json:"scope" db:"scope"`
	TargetID string     `json:"target_id" db:"target_id"` // user_id or tenant_id

	// Token limits
	TokensPerDay   *int64 `json:"tokens_per_day,omitempty" db:"tokens_per_day"`
	TokensPerMonth *int64 `json:"tokens_per_month,omitempty" db:"tokens_per_month"`

	// Request limits
	RequestsPerDay   *int64 `json:"requests_per_day,omitempty" db:"requests_per_day"`
	RequestsPerMonth *int64 `json:"requests_per_month,omitempty" db:"requests_per_month"`
	MaxConcurrent    *int   `json:"max_concurrent,omitempty" db:"max_concurrent"`

	// Storage limits
	MaxStorageBytes  *int64 `json:"max_storage_bytes,omitempty" db:"max_storage_bytes"`
	MaxConversations *int   `json:"max_conversations,omitempty" db:"max_conversations"`
	MaxFileSize      *int64 `json:"max_file_size,omitempty" db:"max_file_size"`

	// Model restrictions
	AllowedModels StringSlice `json:"allowed_models,omitempty" db:"allowed_models"` // JSON array

	// Metadata
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// QuotaUsage представляет текущее использование квоты
type QuotaUsage struct {
	ID       string `json:"id" db:"id"`
	QuotaID  string `json:"quota_id" db:"quota_id"`
	TargetID string `json:"target_id" db:"target_id"`

	// Current usage counters
	TokensUsedToday   int64 `json:"tokens_used_today" db:"tokens_used_today"`
	TokensUsedMonth   int64 `json:"tokens_used_month" db:"tokens_used_month"`
	RequestsToday     int64 `json:"requests_today" db:"requests_today"`
	RequestsMonth     int64 `json:"requests_month" db:"requests_month"`
	CurrentConcurrent int   `json:"current_concurrent" db:"current_concurrent"`
	StorageUsedBytes  int64 `json:"storage_used_bytes" db:"storage_used_bytes"`
	ConversationsCount int  `json:"conversations_count" db:"conversations_count"`

	// Reset timestamps
	LastDailyReset   time.Time `json:"last_daily_reset" db:"last_daily_reset"`
	LastMonthlyReset time.Time `json:"last_monthly_reset" db:"last_monthly_reset"`

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// QuotaCheck представляет результат проверки квоты
type QuotaCheck struct {
	Allowed     bool   `json:"allowed"`
	QuotaExists bool   `json:"quota_exists"`
	ErrorType   string `json:"error_type,omitempty"`   // "daily_tokens", "monthly_tokens", etc.
	Limit       int64  `json:"limit,omitempty"`        // The quota limit
	Used        int64  `json:"used,omitempty"`         // Current usage
	Remaining   int64  `json:"remaining,omitempty"`    // Remaining quota
	Message     string `json:"message,omitempty"`      // Human-readable message
}

// QuotaStats представляет статистику использования квоты (для UI)
type QuotaStats struct {
	QuotaID   string     `json:"quota_id"`
	Scope     QuotaScope `json:"scope"`
	TargetID  string     `json:"target_id"`
	TargetName string    `json:"target_name,omitempty"` // User/Tenant name for display

	// Token stats
	TokensPerDay      *int64  `json:"tokens_per_day,omitempty"`
	TokensUsedToday   int64   `json:"tokens_used_today"`
	TokensPercentDay  float64 `json:"tokens_percent_day,omitempty"`

	TokensPerMonth      *int64  `json:"tokens_per_month,omitempty"`
	TokensUsedMonth     int64   `json:"tokens_used_month"`
	TokensPercentMonth  float64 `json:"tokens_percent_month,omitempty"`

	// Request stats
	RequestsPerDay      *int64  `json:"requests_per_day,omitempty"`
	RequestsToday       int64   `json:"requests_today"`
	RequestsPercentDay  float64 `json:"requests_percent_day,omitempty"`

	RequestsPerMonth      *int64  `json:"requests_per_month,omitempty"`
	RequestsMonth         int64   `json:"requests_month"`
	RequestsPercentMonth  float64 `json:"requests_percent_month,omitempty"`

	// Concurrent stats
	MaxConcurrent     *int    `json:"max_concurrent,omitempty"`
	CurrentConcurrent int     `json:"current_concurrent"`
	ConcurrentPercent float64 `json:"concurrent_percent,omitempty"`

	// Storage stats
	MaxStorageBytes     *int64  `json:"max_storage_bytes,omitempty"`
	StorageUsedBytes    int64   `json:"storage_used_bytes"`
	StoragePercentUsed  float64 `json:"storage_percent_used,omitempty"`

	// Last reset times
	LastDailyReset   time.Time `json:"last_daily_reset"`
	LastMonthlyReset time.Time `json:"last_monthly_reset"`
}

// StringSlice is a []string that can be stored in database as JSON
type StringSlice []string

// Value implements driver.Valuer
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		// Try string
		str, ok := value.(string)
		if !ok {
			return nil
		}
		bytes = []byte(str)
	}

	return json.Unmarshal(bytes, s)
}

// CreateQuotaRequest представляет запрос на создание квоты
type CreateQuotaRequest struct {
	Name     string     `json:"name" binding:"required"`
	Scope    QuotaScope `json:"scope" binding:"required,oneof=user tenant"`
	TargetID string     `json:"target_id" binding:"required"`

	TokensPerDay      *int64      `json:"tokens_per_day,omitempty"`
	TokensPerMonth    *int64      `json:"tokens_per_month,omitempty"`
	RequestsPerDay    *int64      `json:"requests_per_day,omitempty"`
	RequestsPerMonth  *int64      `json:"requests_per_month,omitempty"`
	MaxConcurrent     *int        `json:"max_concurrent,omitempty"`
	MaxStorageBytes   *int64      `json:"max_storage_bytes,omitempty"`
	MaxConversations  *int        `json:"max_conversations,omitempty"`
	MaxFileSize       *int64      `json:"max_file_size,omitempty"`
	AllowedModels     StringSlice `json:"allowed_models,omitempty"`
}

// UpdateQuotaRequest представляет запрос на обновление квоты
type UpdateQuotaRequest struct {
	Name             *string     `json:"name,omitempty"`
	TokensPerDay     *int64      `json:"tokens_per_day,omitempty"`
	TokensPerMonth   *int64      `json:"tokens_per_month,omitempty"`
	RequestsPerDay   *int64      `json:"requests_per_day,omitempty"`
	RequestsPerMonth *int64      `json:"requests_per_month,omitempty"`
	MaxConcurrent    *int        `json:"max_concurrent,omitempty"`
	MaxStorageBytes  *int64      `json:"max_storage_bytes,omitempty"`
	MaxConversations *int        `json:"max_conversations,omitempty"`
	MaxFileSize      *int64      `json:"max_file_size,omitempty"`
	AllowedModels    *StringSlice `json:"allowed_models,omitempty"`
	Enabled          *bool       `json:"enabled,omitempty"`
}


