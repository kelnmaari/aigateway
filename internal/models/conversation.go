// Package models provides data models for Chat Conversations
package models

import (
	"time"
)

// Conversation представляет чат-диалог с моделью
type Conversation struct {
	// Основная информация
	ID    string `json:"id" db:"id"`       // Уникальный ID (UUID)
	Title string `json:"title" db:"title"` // Название диалога

	// Владение
	UserID   string  `json:"user_id" db:"user_id"`               // Владелец диалога
	TenantID *string `json:"tenant_id,omitempty" db:"tenant_id"` // Tenant (если в контексте организации)

	// Модель и настройки
	Model        string   `json:"model" db:"model"`                           // Используемая модель
	Temperature  *float64 `json:"temperature,omitempty" db:"temperature"`     // Temperature (0.0-2.0)
	SystemPrompt string   `json:"system_prompt,omitempty" db:"system_prompt"` // Системный промпт

	// Статус
	Status     ConversationStatus `json:"status" db:"status"`           // Статус диалога
	IsArchived bool               `json:"is_archived" db:"is_archived"` // Архивирован ли
	IsPinned   bool               `json:"is_pinned" db:"is_pinned"`     // Закреплен ли

	// Временные метки
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty" db:"last_message_at"`

	// Статистика
	MessageCount int   `json:"message_count" db:"message_count"` // Количество сообщений
	TotalTokens  int64 `json:"total_tokens" db:"total_tokens"`   // Общее количество токенов

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// ConversationStatus представляет статус диалога
type ConversationStatus string

const (
	ConversationStatusActive   ConversationStatus = "active"   // Активный
	ConversationStatusArchived ConversationStatus = "archived" // Архивирован
	ConversationStatusDeleted  ConversationStatus = "deleted"  // Удален (soft delete)
)

// Message представляет сообщение в диалоге
type Message struct {
	// Основная информация
	ID             string `json:"id" db:"id"`                           // Уникальный ID (UUID)
	ConversationID string `json:"conversation_id" db:"conversation_id"` // ID диалога

	// Роль и контент
	Role    MessageRole `json:"role" db:"role"`       // user, assistant, system, tool
	Content string      `json:"content" db:"content"` // Содержимое сообщения

	// Модель (для assistant messages)
	Model       string   `json:"model,omitempty" db:"model"`             // Модель, которая ответила
	Temperature *float64 `json:"temperature,omitempty" db:"temperature"` // Параметры генерации

	// Tool calls (для function calling)
	ToolCalls  []ToolCall `json:"tool_calls,omitempty" db:"tool_calls"`     // Вызовы инструментов (JSONB)
	ToolCallID string     `json:"tool_call_id,omitempty" db:"tool_call_id"` // ID tool call (для tool role)

	// Токены и стоимость
	PromptTokens     int `json:"prompt_tokens,omitempty" db:"prompt_tokens"`         // Токенов в промпте
	CompletionTokens int `json:"completion_tokens,omitempty" db:"completion_tokens"` // Токенов в ответе
	TotalTokens      int `json:"total_tokens,omitempty" db:"total_tokens"`           // Всего токенов

	// Временные метки
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// Attached files (FILE-STORAGE-01: Phase 4, v1.10.0+)
	FileIDs []string `json:"file_ids,omitempty" db:"-"` // Not stored in messages table, loaded from junction
}

// MessageRole представляет роль автора сообщения
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"      // Пользователь
	MessageRoleAssistant MessageRole = "assistant" // AI ассистент
	MessageRoleSystem    MessageRole = "system"    // Системное сообщение
	MessageRoleTool      MessageRole = "tool"      // Ответ от инструмента
)

// NOTE: ToolCall и FunctionCall определены в openai.go
// Используем те же типы для совместимости

// ConversationFilters представляет фильтры для списка диалогов
type ConversationFilters struct {
	UserID     string              `json:"user_id,omitempty"`
	TenantID   *string             `json:"tenant_id,omitempty"`
	Status     *ConversationStatus `json:"status,omitempty"`
	IsArchived *bool               `json:"is_archived,omitempty"`
	IsPinned   *bool               `json:"is_pinned,omitempty"`
	Model      string              `json:"model,omitempty"`
	Search     string              `json:"search,omitempty"` // Поиск по title

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`

	// Sorting
	OrderBy string `json:"order_by,omitempty"` // created_at, updated_at, last_message_at
	Order   string `json:"order,omitempty"`    // asc, desc
}

// ConversationWithMessages представляет диалог со всеми сообщениями
type ConversationWithMessages struct {
	*Conversation
	Messages []*Message `json:"messages"`
}

