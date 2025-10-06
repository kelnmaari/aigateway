// Package request provides request tracking and monitoring functionality
package request

// Status определяет статус запроса
type Status string

const (
	StatusPending  Status = "pending"
	StatusSuccess  Status = "success"
	StatusError    Status = "error"
	StatusCanceled Status = "canceled"
)

// RequestInfo содержит информацию о запросе для мониторинга
type RequestInfo struct {
	// Базовая информация
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"` // Unix timestamp (seconds)
	Method    string `json:"method"`
	Endpoint  string `json:"endpoint"`
	Status    Status `json:"status"`
	Duration  int64  `json:"duration_ms"` // в миллисекундах

	// Request данные
	Model      string `json:"model,omitempty"`
	APIKeyID   string `json:"api_key_id,omitempty"`   // ID ключа (не сам ключ!)
	APIKeyName string `json:"api_key_name,omitempty"` // Имя ключа для удобства
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`

	// Request body info (для chat/completions)
	Messages    int      `json:"messages,omitempty"`    // количество сообщений
	Tools       int      `json:"tools,omitempty"`       // количество tools
	Stream      bool     `json:"stream,omitempty"`      // streaming enabled
	Temperature *float64 `json:"temperature,omitempty"` // temperature parameter

	// Response данные
	StatusCode   int    `json:"status_code,omitempty"`
	ResponseSize int64  `json:"response_size,omitempty"` // bytes
	ErrorMessage string `json:"error_message,omitempty"`

	// Token usage (для chat/completions)
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
	TotalTokens      int    `json:"total_tokens,omitempty"`
	FinishReason     string `json:"finish_reason,omitempty"`
}

// IsComplete возвращает true, если запрос завершен (не pending)
func (r *RequestInfo) IsComplete() bool {
	return r.Status != StatusPending
}

// IsSuccess возвращает true, если запрос успешен
func (r *RequestInfo) IsSuccess() bool {
	return r.Status == StatusSuccess
}

// IsError возвращает true, если запрос завершился с ошибкой
func (r *RequestInfo) IsError() bool {
	return r.Status == StatusError
}

// GetMaskedKey возвращает замаскированную версию API ключа для отображения
func (r *RequestInfo) GetMaskedKey() string {
	if r.APIKeyID == "" {
		return "N/A"
	}
	if len(r.APIKeyID) > 8 {
		return r.APIKeyID[:4] + "..." + r.APIKeyID[len(r.APIKeyID)-4:]
	}
	return r.APIKeyID
}

// FilterFunc определяет функцию фильтрации запросов
type FilterFunc func(*RequestInfo) bool

// FilterByStatus создает фильтр по статусу
func FilterByStatus(status Status) FilterFunc {
	return func(r *RequestInfo) bool {
		return r.Status == status
	}
}

// FilterByEndpoint создает фильтр по endpoint
func FilterByEndpoint(endpoint string) FilterFunc {
	return func(r *RequestInfo) bool {
		return r.Endpoint == endpoint
	}
}

// FilterByModel создает фильтр по модели
func FilterByModel(model string) FilterFunc {
	return func(r *RequestInfo) bool {
		return r.Model == model
	}
}

// SortField определяет поле для сортировки
type SortField string

const (
	SortByTime     SortField = "time"
	SortByDuration SortField = "duration"
	SortByStatus   SortField = "status"
	SortByEndpoint SortField = "endpoint"
)
