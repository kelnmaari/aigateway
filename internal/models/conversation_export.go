// Package models provides data models for conversation export/import
// Version 1.12.3+: Conversation Export/Import (EXPORT-01)
package models

import "time"

// ExportFormat формат экспорта
type ExportFormat string

const (
	ExportFormatJSON     ExportFormat = "json"
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatText     ExportFormat = "text"
)

// ConversationExport структура для экспорта conversation
type ConversationExport struct {
	Metadata       ConversationMetadata `json:"metadata"`
	Conversation   Conversation         `json:"conversation"`
	Messages       []Message            `json:"messages"`
	ExportedAt     time.Time            `json:"exported_at"`
	ExportedByUser string               `json:"exported_by_user"`
	FormatVersion  string               `json:"format_version"` // "1.0"
}

// ConversationMetadata метаданные для экспорта
type ConversationMetadata struct {
	TotalMessages int       `json:"total_messages"`
	StartedAt     time.Time `json:"started_at"`
	LastMessageAt time.Time `json:"last_message_at"`
	ModelUsed     string    `json:"model_used,omitempty"`
	TokensUsed    int       `json:"tokens_used,omitempty"`
}

// BulkExportRequest запрос на bulk export
type BulkExportRequest struct {
	ConversationIDs []string     `json:"conversation_ids" binding:"required,min=1"`
	Format          ExportFormat `json:"format" binding:"required,oneof=json markdown text"`
	IncludeMetadata bool         `json:"include_metadata"`
}

// BulkExportResult результат bulk export
type BulkExportResult struct {
	TotalRequested int                     `json:"total_requested"`
	TotalExported  int                     `json:"total_exported"`
	Exports        []ConversationExport    `json:"exports,omitempty"`
	DownloadURL    string                  `json:"download_url,omitempty"` // For ZIP download
	Errors         []ConversationExportErr `json:"errors,omitempty"`
}

// ConversationExportErr ошибка экспорта conversation
type ConversationExportErr struct {
	ConversationID string `json:"conversation_id"`
	Error          string `json:"error"`
}

// ImportConversationRequest запрос на импорт conversation
type ImportConversationRequest struct {
	Format       ExportFormat         `json:"format" binding:"required,oneof=json markdown"`
	Data         string               `json:"data" binding:"required"` // JSON string or Markdown text
	Conversation *ConversationExport  `json:"conversation"`            // Parsed conversation for JSON format
	Options      ConversationImportOpt `json:"options"`
}

// ConversationImportOpt опции импорта
type ConversationImportOpt struct {
	MergeIntoExisting  bool   `json:"merge_into_existing"` // Merge messages в существующую conversation
	TargetConvID       string `json:"target_conversation_id,omitempty"`
	PreserveTimestamps bool   `json:"preserve_timestamps"` // Сохранить оригинальные timestamps
	PreserveIDs        bool   `json:"preserve_ids"`        // Сохранить оригинальные IDs
}

// ImportResult результат импорта
type ImportResult struct {
	Success          bool   `json:"success"`
	ConversationID   string `json:"conversation_id"`
	MessagesImported int    `json:"messages_imported"`
	Error            string `json:"error,omitempty"`
}
