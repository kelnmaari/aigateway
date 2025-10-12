// Package models provides data models for API Usage Statistics
package models

import (
	"time"
)

// APIUsage представляет запись об использовании API
type APIUsage struct {
	// Основная информация
	ID string `json:"id" db:"id"` // Уникальный ID (UUID)

	// Владение
	UserID   string  `json:"user_id" db:"user_id"`                 // ID пользователя
	TenantID *string `json:"tenant_id,omitempty" db:"tenant_id"`   // ID tenant (если в контексте организации)
	APIKeyID *string `json:"api_key_id,omitempty" db:"api_key_id"` // ID использованного API ключа (nullable для JWT auth, v1.5.12)

	// Запрос
	Endpoint string `json:"endpoint" db:"endpoint"` // Эндпоинт (/v1/chat/completions, etc.)
	Method   string `json:"method" db:"method"`     // HTTP метод
	Model    string `json:"model" db:"model"`       // Используемая модель

	// Результат
	StatusCode   int    `json:"status_code" db:"status_code"`               // HTTP status code
	Success      bool   `json:"success" db:"success"`                       // Успешен ли запрос
	ErrorMessage string `json:"error_message,omitempty" db:"error_message"` // Сообщение об ошибке

	// Токены и стоимость
	PromptTokens     int `json:"prompt_tokens" db:"prompt_tokens"`         // Токенов в промпте
	CompletionTokens int `json:"completion_tokens" db:"completion_tokens"` // Токенов в ответе
	TotalTokens      int `json:"total_tokens" db:"total_tokens"`           // Всего токенов

	// Производительность
	DurationMS int64 `json:"duration_ms" db:"duration_ms"` // Длительность в миллисекундах

	// Временная метка
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Дополнительная информация
	UserAgent      string  `json:"user_agent,omitempty" db:"user_agent"`           // User-Agent
	IPAddress      string  `json:"ip_address,omitempty" db:"ip_address"`           // IP адрес
	ConversationID *string `json:"conversation_id,omitempty" db:"conversation_id"` // ID диалога (если через Chat UI)

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// UsageStats представляет агрегированную статистику использования
type UsageStats struct {
	// Период
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`

	// Общие показатели
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	SuccessRate        float64 `json:"success_rate"` // Процент успешных запросов
	ErrorRate          float64 `json:"error_rate"`   // Процент ошибок (для фронтенда)

	// Токены
	TotalTokens      int64 `json:"total_tokens"`
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`

	// Производительность
	AvgDuration   int64   `json:"avg_duration"` // Средняя длительность в ms (для фронтенда)
	AvgDurationMS float64 `json:"avg_duration_ms"`
	MinDurationMS int64   `json:"min_duration_ms"`
	MaxDurationMS int64   `json:"max_duration_ms"`

	// По моделям (для фронтенда - массив)
	Models     []*ModelUsageStats          `json:"models"`
	ModelUsage map[string]*ModelUsageStats `json:"model_usage"`

	// По эндпоинтам
	EndpointUsage map[string]*EndpointUsageStats `json:"endpoint_usage"`

	// По API ключам (для фронтенда)
	APIKeys []*APIKeyUsageStats `json:"api_keys"`

	// Последние запросы (для фронтенда)
	RecentRequests []*RecentRequest `json:"recent_requests"`

	// По дням (для графиков)
	DailyUsage []*DailyUsageStats `json:"daily_usage"`
}

// ModelUsageStats представляет статистику по модели
type ModelUsageStats struct {
	Model         string  `json:"model"`
	Requests      int64   `json:"requests"` // Для фронтенда
	RequestCount  int64   `json:"request_count"`
	Tokens        int64   `json:"tokens"` // Для фронтенда
	TotalTokens   int64   `json:"total_tokens"`
	AvgDuration   int64   `json:"avg_duration"` // Для фронтенда (ms)
	AvgDurationMS float64 `json:"avg_duration_ms"`
	SuccessRate   float64 `json:"success_rate"`
}

// EndpointUsageStats представляет статистику по эндпоинту
type EndpointUsageStats struct {
	Endpoint      string  `json:"endpoint"`
	RequestCount  int64   `json:"request_count"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	SuccessRate   float64 `json:"success_rate"`
}

// DailyUsageStats представляет статистику за день
type DailyUsageStats struct {
	Date          string  `json:"date"` // YYYY-MM-DD
	RequestCount  int64   `json:"request_count"`
	TotalTokens   int64   `json:"total_tokens"`
	AvgDurationMS float64 `json:"avg_duration_ms"`
	SuccessRate   float64 `json:"success_rate"`
}

// APIKeyUsageStats представляет статистику по API ключу
type APIKeyUsageStats struct {
	KeyID    string    `json:"key_id"`
	Requests int64     `json:"requests"`
	Tokens   int64     `json:"tokens"`
	LastUsed time.Time `json:"last_used"`
}

// RecentRequest представляет недавний запрос
type RecentRequest struct {
	Timestamp time.Time `json:"timestamp"`
	Model     string    `json:"model"`
	KeyID     string    `json:"key_id"`
	Tokens    int       `json:"tokens"`
	Duration  int64     `json:"duration"` // ms
	Success   bool      `json:"success"`
}

// UsageFilters представляет фильтры для запросов статистики
type UsageFilters struct {
	UserID   string  `json:"user_id,omitempty"`
	TenantID *string `json:"tenant_id,omitempty"`
	APIKeyID string  `json:"api_key_id,omitempty"`
	Model    string  `json:"model,omitempty"`
	Endpoint string  `json:"endpoint,omitempty"`
	Success  *bool   `json:"success,omitempty"`

	// Временной период
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// ========================================
// Reports Statistics Models (v1.6.3+)
// ========================================

// UsageReportStats представляет статистику для usage отчета
type UsageReportStats struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	TotalTokens        int64
	UniqueUsers        int
	UniqueModels       int
	TopModels          []struct {
		Model    string
		Requests int64
		Tokens   int64
	}
	TopUsers []struct {
		Username string
		Requests int64
		Tokens   int64
	}
}

// PerformanceReportStats представляет статистику для performance отчета
type PerformanceReportStats struct {
	TotalRequests      int64
	AvgLatencyMS       float64
	P50LatencyMS       float64
	P95LatencyMS       float64
	P99LatencyMS       float64
	SlowRequestsCount  int
	ErrorCount         int
	RequestsPerSecond  float64
	FastestRequest     float64
	SlowestRequest     float64
}