// Package export provides conversation export/import services
// Version 1.12.3+: Conversation Export/Import (EXPORT-01)
package export

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"

	"github.com/sirupsen/logrus"
)

// ConversationExporter экспортирует conversations в различные форматы
type ConversationExporter struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewConversationExporter создает новый exporter
func NewConversationExporter(db storage.Database, logger *logrus.Logger) *ConversationExporter {
	return &ConversationExporter{
		db:     db,
		logger: logger,
	}
}

// ExportToJSON экспортирует conversation в JSON
func (e *ConversationExporter) ExportToJSON(
	ctx context.Context,
	conversationID string,
	userID string,
) (string, error) {
	export, err := e.buildExport(ctx, conversationID, userID)
	if err != nil {
		return "", err
	}

	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonData), nil
}

// ExportToMarkdown экспортирует conversation в Markdown
func (e *ConversationExporter) ExportToMarkdown(
	ctx context.Context,
	conversationID string,
	userID string,
) (string, error) {
	export, err := e.buildExport(ctx, conversationID, userID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Conversation: %s\n\n", export.Conversation.Title))
	sb.WriteString(fmt.Sprintf("**Created:** %s\n\n", export.Conversation.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("**Model:** %s\n\n", export.Metadata.ModelUsed))
	sb.WriteString(fmt.Sprintf("**Total Messages:** %d\n\n", export.Metadata.TotalMessages))
	sb.WriteString("---\n\n")

	// Messages
	for i, msg := range export.Messages {
		// Role header
		roleEmoji := "🤖"
		if msg.Role == "user" {
			roleEmoji = "👤"
		} else if msg.Role == "system" {
			roleEmoji = "⚙️"
		}

		sb.WriteString(fmt.Sprintf("## %s Message %d (%s)\n\n", roleEmoji, i+1, msg.Role))
		sb.WriteString(fmt.Sprintf("**Time:** %s\n\n", msg.CreatedAt.Format(time.RFC3339)))

		// Content
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")

		// Metadata (optional) - будет расширено в будущих версиях

		sb.WriteString("---\n\n")
	}

	// Footer
	sb.WriteString(fmt.Sprintf("**Exported:** %s\n", export.ExportedAt.Format(time.RFC3339)))

	return sb.String(), nil
}

// ExportToText экспортирует conversation в простой текст
func (e *ConversationExporter) ExportToText(
	ctx context.Context,
	conversationID string,
	userID string,
) (string, error) {
	export, err := e.buildExport(ctx, conversationID, userID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Conversation: %s\n", export.Conversation.Title))
	sb.WriteString(fmt.Sprintf("Created: %s\n", export.Conversation.CreatedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Messages: %d\n", export.Metadata.TotalMessages))
	sb.WriteString(strings.Repeat("=", 80))
	sb.WriteString("\n\n")

	// Messages
	for _, msg := range export.Messages {
		sb.WriteString(fmt.Sprintf("[%s] %s:\n", msg.CreatedAt.Format("15:04:05"), msg.Role))
		sb.WriteString(msg.Content)
		sb.WriteString("\n\n")
		sb.WriteString(strings.Repeat("-", 80))
		sb.WriteString("\n\n")
	}

	return sb.String(), nil
}

// buildExport строит ConversationExport структуру
func (e *ConversationExporter) buildExport(
	ctx context.Context,
	conversationID string,
	userID string,
) (*models.ConversationExport, error) {
	// Get conversation
	conv, err := e.db.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Verify ownership
	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied: conversation belongs to different user")
	}

	// Get messages
	messagesPtr, err := e.db.ListConversationMessages(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// Convert []*Message to []Message
	messages := make([]models.Message, 0, len(messagesPtr))
	for _, msgPtr := range messagesPtr {
		if msgPtr != nil {
			messages = append(messages, *msgPtr)
		}
	}

	// Build metadata
	metadata := models.ConversationMetadata{
		TotalMessages: len(messages),
		StartedAt:     conv.CreatedAt,
		ModelUsed:     conv.Model,
	}

	if len(messages) > 0 {
		metadata.LastMessageAt = messages[len(messages)-1].CreatedAt
		// Note: TokensUsed tracking будет добавлено в будущих версиях
		metadata.TokensUsed = 0
	}

	export := &models.ConversationExport{
		Metadata:       metadata,
		Conversation:   *conv,
		Messages:       messages,
		ExportedAt:     time.Now(),
		ExportedByUser: userID,
		FormatVersion:  "1.0",
	}

	return export, nil
}

// BulkExport экспортирует множество conversations
func (e *ConversationExporter) BulkExport(
	ctx context.Context,
	conversationIDs []string,
	userID string,
	format models.ExportFormat,
) (*models.BulkExportResult, error) {
	result := &models.BulkExportResult{
		TotalRequested: len(conversationIDs),
		Exports:        make([]models.ConversationExport, 0, len(conversationIDs)),
		Errors:         make([]models.ConversationExportErr, 0),
	}

	for _, convID := range conversationIDs {
		export, err := e.buildExport(ctx, convID, userID)
		if err != nil {
			result.Errors = append(result.Errors, models.ConversationExportErr{
				ConversationID: convID,
				Error:          err.Error(),
			})
			continue
		}

		result.Exports = append(result.Exports, *export)
		result.TotalExported++
	}

	return result, nil
}


