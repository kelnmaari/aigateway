// Package export provides conversation export/import services
// Version 1.12.3+: Conversation Export/Import (EXPORT-01)
package export

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ConversationImporter импортирует conversations из различных форматов
type ConversationImporter struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewConversationImporter создает новый importer
func NewConversationImporter(db storage.Database, logger *logrus.Logger) *ConversationImporter {
	return &ConversationImporter{
		db:     db,
		logger: logger,
	}
}

// ImportFromJSON импортирует conversation из JSON
func (i *ConversationImporter) ImportFromJSON(
	ctx context.Context,
	jsonData string,
	userID string,
	tenantID string,
	options models.ConversationImportOpt,
) (*models.ImportResult, error) {
	// Parse JSON
	var export models.ConversationExport
	if err := json.Unmarshal([]byte(jsonData), &export); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	return i.importConversation(ctx, &export, userID, tenantID, options)
}

// importConversation выполняет импорт conversation
func (i *ConversationImporter) importConversation(
	ctx context.Context,
	export *models.ConversationExport,
	userID string,
	tenantID string,
	options models.ConversationImportOpt,
) (*models.ImportResult, error) {
	// Merge into existing conversation
	if options.MergeIntoExisting && options.TargetConvID != "" {
		return i.mergeIntoExisting(ctx, export, options.TargetConvID, userID, options)
	}

	// Create new conversation
	return i.createNew(ctx, export, userID, tenantID, options)
}

// createNew создает новую conversation при импорте
func (i *ConversationImporter) createNew(
	ctx context.Context,
	export *models.ConversationExport,
	userID string,
	tenantID string,
	options models.ConversationImportOpt,
) (*models.ImportResult, error) {
	// Generate new IDs or preserve original
	convID := uuid.New().String()
	if options.PreserveIDs && export.Conversation.ID != "" {
		convID = export.Conversation.ID
	}

	// Create conversation
	var tenantIDPtr *string
	if tenantID != "" {
		tenantIDPtr = &tenantID
	}
	
	conv := &models.Conversation{
		ID:       convID,
		UserID:   userID,
		TenantID: tenantIDPtr,
		Title:    export.Conversation.Title,
		Model:    export.Conversation.Model,
		SystemPrompt: export.Conversation.SystemPrompt,
	}

	if options.PreserveTimestamps {
		conv.CreatedAt = export.Conversation.CreatedAt
		conv.UpdatedAt = export.Conversation.UpdatedAt
	} else {
		now := time.Now()
		conv.CreatedAt = now
		conv.UpdatedAt = now
	}

	// Save conversation
	if err := i.db.CreateConversation(ctx, conv); err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Import messages
	messagesImported := 0
	for _, msg := range export.Messages {
		messageID := uuid.New().String()
		if options.PreserveIDs && msg.ID != "" {
			messageID = msg.ID
		}

		newMsg := &models.Message{
			ID:             messageID,
			ConversationID: convID,
			Role:           msg.Role,
			Content:        msg.Content,
		}

		if options.PreserveTimestamps {
			newMsg.CreatedAt = msg.CreatedAt
		} else {
			newMsg.CreatedAt = time.Now()
		}

		if err := i.db.CreateMessage(ctx, newMsg); err != nil {
			i.logger.WithError(err).WithField("message_id", messageID).Warn("Failed to import message")
			continue
		}

		messagesImported++
	}

	return &models.ImportResult{
		Success:          true,
		ConversationID:   convID,
		MessagesImported: messagesImported,
	}, nil
}

// mergeIntoExisting добавляет сообщения в существующую conversation
func (i *ConversationImporter) mergeIntoExisting(
	ctx context.Context,
	export *models.ConversationExport,
	targetConvID string,
	userID string,
	options models.ConversationImportOpt,
) (*models.ImportResult, error) {
	// Verify conversation exists and belongs to user
	conv, err := i.db.GetConversation(ctx, targetConvID)
	if err != nil {
		return nil, fmt.Errorf("target conversation not found: %w", err)
	}

	if conv.UserID != userID {
		return nil, fmt.Errorf("access denied: conversation belongs to different user")
	}

	// Import messages
	messagesImported := 0
	for _, msg := range export.Messages {
		messageID := uuid.New().String()
		if options.PreserveIDs && msg.ID != "" {
			messageID = msg.ID
		}

		newMsg := &models.Message{
			ID:             messageID,
			ConversationID: targetConvID,
			Role:           msg.Role,
			Content:        msg.Content,
		}

		if options.PreserveTimestamps {
			newMsg.CreatedAt = msg.CreatedAt
		} else {
			newMsg.CreatedAt = time.Now()
		}

		if err := i.db.CreateMessage(ctx, newMsg); err != nil {
			i.logger.WithError(err).WithField("message_id", messageID).Warn("Failed to import message")
			continue
		}

		messagesImported++
	}

	// Update conversation timestamp
	conv.UpdatedAt = time.Now()
	if err := i.db.UpdateConversation(ctx, conv); err != nil {
		i.logger.WithError(err).Warn("Failed to update conversation timestamp")
	}

	return &models.ImportResult{
		Success:          true,
		ConversationID:   targetConvID,
		MessagesImported: messagesImported,
	}, nil
}

