// Package models provides data models for rate limiting
// Version 1.12.2+: Advanced Rate Limiting (RATE-02)
package models

import "time"

// RateLimitScope определяет область применения rate limit
type RateLimitScope string

const (
	RateLimitScopeGlobal  RateLimitScope = "global"
	RateLimitScopeTenant  RateLimitScope = "tenant"
	RateLimitScopeUser    RateLimitScope = "user"
	RateLimitScopeAPIKey  RateLimitScope = "api_key"
	RateLimitScopeModel   RateLimitScope = "model"
)

// RateLimitConfig конфигурация rate limit
type RateLimitConfig struct {
	ID       string         `json:"id" db:"id"`
	Name     string         `json:"name" db:"name"`
	Scope    RateLimitScope `json:"scope" db:"scope"`
	TargetID *string        `json:"target_id,omitempty" db:"target_id"`

	// Model-specific (optional)
	ModelName *string `json:"model_name,omitempty" db:"model_name"`

	// Rate limits (NULL = no limit)
	RequestsPerSecond *int `json:"requests_per_second,omitempty" db:"requests_per_second"`
	RequestsPerMinute *int `json:"requests_per_minute,omitempty" db:"requests_per_minute"`
	RequestsPerHour   *int `json:"requests_per_hour,omitempty" db:"requests_per_hour"`
	RequestsPerDay    *int `json:"requests_per_day,omitempty" db:"requests_per_day"`

	// Burst allowance
	BurstSize int `json:"burst_size" db:"burst_size"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RateLimitUsage tracking использования rate limit (для sliding window)
type RateLimitUsage struct {
	ID            string    `json:"id" db:"id"`
	RateLimitID   string    `json:"rate_limit_id" db:"rate_limit_id"`
	WindowType    string    `json:"window_type" db:"window_type"` // "second", "minute", "hour", "day"
	WindowStart   time.Time `json:"window_start" db:"window_start"`
	RequestCount  int       `json:"request_count" db:"request_count"`
	LastRequestAt time.Time `json:"last_request_at" db:"last_request_at"`
}

// RateLimitResult результат проверки rate limit
type RateLimitResult struct {
	Allowed   bool      `json:"allowed"`
	Remaining int       `json:"remaining"` // сколько запросов осталось
	ResetAt   time.Time `json:"reset_at"`  // когда сбросится limit
	Limit     int       `json:"limit"`     // общий лимит
	Window    string    `json:"window"`    // "second", "minute", "hour", "day"
	Scope     string    `json:"scope"`     // какой scope сработал
}

// CreateRateLimitRequest запрос на создание rate limit
type CreateRateLimitRequest struct {
	Name              string         `json:"name" binding:"required,min=3,max=255"`
	Scope             RateLimitScope `json:"scope" binding:"required,oneof=global tenant user api_key model"`
	TargetID          *string        `json:"target_id"`
	ModelName         *string        `json:"model_name"`
	RequestsPerSecond *int           `json:"requests_per_second"`
	RequestsPerMinute *int           `json:"requests_per_minute"`
	RequestsPerHour   *int           `json:"requests_per_hour"`
	RequestsPerDay    *int           `json:"requests_per_day"`
	BurstSize         int            `json:"burst_size"`
}

// UpdateRateLimitRequest запрос на обновление rate limit
type UpdateRateLimitRequest struct {
	Name              *string `json:"name"`
	RequestsPerSecond *int    `json:"requests_per_second"`
	RequestsPerMinute *int    `json:"requests_per_minute"`
	RequestsPerHour   *int    `json:"requests_per_hour"`
	RequestsPerDay    *int    `json:"requests_per_day"`
	BurstSize         *int    `json:"burst_size"`
}

