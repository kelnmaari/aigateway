// Package sqlite provides SQLite implementation of conversation-related database operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"aigateway/internal/models"
)

// ========================================
// Conversations CRUD Operations
// ========================================

// CreateConversation создает новую беседу
func (s *SQLiteDB) CreateConversation(ctx context.Context, conv *models.Conversation) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("title", conv.Title).Debug("Creating conversation")

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	var err error
	if conv.Metadata != nil {
		metadataJSON, err = json.Marshal(conv.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO conversations (
			id, title, user_id, tenant_id,
			model, temperature, system_prompt,
			status, is_archived, is_pinned,
			created_at, updated_at, last_message_at,
			message_count, total_tokens,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		conv.ID,
		conv.Title,
		conv.UserID,
		conv.TenantID,
		conv.Model,
		conv.Temperature,
		conv.SystemPrompt,
		conv.Status,
		conv.IsArchived,
		conv.IsPinned,
		conv.CreatedAt,
		conv.UpdatedAt,
		conv.LastMessageAt,
		conv.MessageCount,
		conv.TotalTokens,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert conversation: %w", err)
	}

	s.logger.WithField("conversation_id", conv.ID).Info("Conversation created successfully")
	return nil
}

// GetConversation получает беседу по ID
func (s *SQLiteDB) GetConversation(ctx context.Context, id string) (*models.Conversation, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, title, user_id, tenant_id,
			model, temperature, system_prompt,
			status, is_archived, is_pinned,
			created_at, updated_at, last_message_at,
			message_count, total_tokens,
			metadata
		FROM conversations
		WHERE id = ?
	`

	conv, err := s.scanConversation(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("conversation not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return conv, nil
}

// UpdateConversation обновляет данные беседы
func (s *SQLiteDB) UpdateConversation(ctx context.Context, conv *models.Conversation) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("conversation_id", conv.ID).Debug("Updating conversation")

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	var err error
	if conv.Metadata != nil {
		metadataJSON, err = json.Marshal(conv.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE conversations SET
			title = ?,
			model = ?,
			temperature = ?,
			system_prompt = ?,
			status = ?,
			is_archived = ?,
			is_pinned = ?,
			updated_at = ?,
			last_message_at = ?,
			message_count = ?,
			total_tokens = ?,
			metadata = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		conv.Title,
		conv.Model,
		conv.Temperature,
		conv.SystemPrompt,
		conv.Status,
		conv.IsArchived,
		conv.IsPinned,
		conv.UpdatedAt,
		conv.LastMessageAt,
		conv.MessageCount,
		conv.TotalTokens,
		metadataJSON,
		conv.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found: %s", conv.ID)
	}

	s.logger.WithField("conversation_id", conv.ID).Info("Conversation updated successfully")
	return nil
}

// DeleteConversation удаляет беседу
func (s *SQLiteDB) DeleteConversation(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("conversation_id", id).Debug("Deleting conversation")

	// This will cascade delete all messages due to FK constraint
	query := `DELETE FROM conversations WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("conversation not found: %s", id)
	}

	s.logger.WithField("conversation_id", id).Info("Conversation deleted successfully")
	return nil
}

// ListUserConversations возвращает список бесед пользователя с фильтрацией
func (s *SQLiteDB) ListUserConversations(ctx context.Context, userID string, filters models.ConversationFilters) ([]*models.Conversation, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, title, user_id, tenant_id,
			model, temperature, system_prompt,
			status, is_archived, is_pinned,
			created_at, updated_at, last_message_at,
			message_count, total_tokens,
			metadata
		FROM conversations
		WHERE user_id = ?
	`

	var args []any
	args = append(args, userID)

	// Apply filters
	if filters.TenantID != nil {
		query += " AND tenant_id = ?"
		args = append(args, *filters.TenantID)
	}

	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, *filters.Status)
	}

	if filters.IsArchived != nil {
		query += " AND is_archived = ?"
		args = append(args, *filters.IsArchived)
	}

	if filters.IsPinned != nil {
		query += " AND is_pinned = ?"
		args = append(args, *filters.IsPinned)
	}

	if filters.Search != "" {
		query += " AND title LIKE ?"
		args = append(args, "%"+filters.Search+"%")
	}

	// Add ordering
	query += " ORDER BY "
	if filters.IsPinned != nil && *filters.IsPinned {
		query += "is_pinned DESC, "
	}
	query += "updated_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*models.Conversation
	for rows.Next() {
		conv, err := s.scanConversation(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	s.logger.WithField("count", len(conversations)).Debug("Listed user conversations")
	return conversations, nil
}

// ========================================
// Messages CRUD Operations
// ========================================

// CreateMessage создает новое сообщение
func (s *SQLiteDB) CreateMessage(ctx context.Context, msg *models.Message) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("role", msg.Role).Debug("Creating message")

	// Serialize tool_calls to JSON if present
	var toolCallsJSON []byte
	var err error
	if msg.ToolCalls != nil {
		toolCallsJSON, err = json.Marshal(msg.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to marshal tool_calls: %w", err)
		}
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if msg.Metadata != nil {
		metadataJSON, err = json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO messages (
			id, conversation_id, role, content,
			model, temperature,
			tool_calls, tool_call_id,
			prompt_tokens, completion_tokens, total_tokens,
			created_at,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		msg.ID,
		msg.ConversationID,
		msg.Role,
		msg.Content,
		msg.Model,
		msg.Temperature,
		toolCallsJSON,
		msg.ToolCallID,
		msg.PromptTokens,
		msg.CompletionTokens,
		msg.TotalTokens,
		msg.CreatedAt,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Save file attachments if any (FILE-STORAGE-01: Phase 4)
	if len(msg.FileIDs) > 0 {
		if err := s.saveMessageFiles(ctx, msg.ID, msg.FileIDs); err != nil {
			return fmt.Errorf("failed to save message files: %w", err)
		}
	}

	s.logger.WithField("message_id", msg.ID).Info("Message created successfully")
	return nil
}

// saveMessageFiles сохраняет связи message-files в junction table
func (s *SQLiteDB) saveMessageFiles(ctx context.Context, messageID string, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	query := `INSERT INTO message_files (message_id, file_id) VALUES (?, ?)`

	for _, fileID := range fileIDs {
		_, err := s.db.ExecContext(ctx, query, messageID, fileID)
		if err != nil {
			return fmt.Errorf("failed to link file %s to message: %w", fileID, err)
		}
	}

	return nil
}

// GetMessage получает сообщение по ID
func (s *SQLiteDB) GetMessage(ctx context.Context, id string) (*models.Message, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, conversation_id, role, content,
			model, temperature,
			tool_calls, tool_call_id,
			prompt_tokens, completion_tokens, total_tokens,
			created_at,
			metadata
		FROM messages
		WHERE id = ?
	`

	msg, err := s.scanMessage(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Load attached files (FILE-STORAGE-01: Phase 4)
	fileIDs, err := s.loadMessageFiles(ctx, msg.ID)
	if err != nil {
		// Log error but don't fail the entire request
		s.logger.WithError(err).Warn("Failed to load message files")
	} else {
		msg.FileIDs = fileIDs
	}

	return msg, nil
}

// loadMessageFiles загружает file_ids для сообщения из junction table
func (s *SQLiteDB) loadMessageFiles(ctx context.Context, messageID string) ([]string, error) {
	query := `SELECT file_id FROM message_files WHERE message_id = ? ORDER BY created_at`

	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query message files: %w", err)
	}
	defer rows.Close()

	var fileIDs []string
	for rows.Next() {
		var fileID string
		if err := rows.Scan(&fileID); err != nil {
			return nil, fmt.Errorf("failed to scan file_id: %w", err)
		}
		fileIDs = append(fileIDs, fileID)
	}

	return fileIDs, rows.Err()
}

// ListConversationMessages возвращает список сообщений беседы
func (s *SQLiteDB) ListConversationMessages(ctx context.Context, convID string) ([]*models.Message, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, conversation_id, role, content,
			model, temperature,
			tool_calls, tool_call_id,
			prompt_tokens, completion_tokens, total_tokens,
			created_at,
			metadata
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		msg, err := s.scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Load attached files for each message (FILE-STORAGE-01: Phase 4)
	for _, msg := range messages {
		fileIDs, err := s.loadMessageFiles(ctx, msg.ID)
		if err != nil {
			// Log error but don't fail the entire request
			s.logger.WithError(err).Warn("Failed to load message files")
			continue
		}
		msg.FileIDs = fileIDs
	}

	s.logger.WithField("count", len(messages)).Debug("Listed conversation messages")
	return messages, nil
}

// DeleteConversationMessages удаляет все сообщения беседы
func (s *SQLiteDB) DeleteConversationMessages(ctx context.Context, convID string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("conversation_id", convID).Debug("Deleting conversation messages")

	query := `DELETE FROM messages WHERE conversation_id = ?`

	result, err := s.db.ExecContext(ctx, query, convID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	s.logger.WithFields(map[string]any{
		"conversation_id": convID,
		"deleted_count":   rowsAffected,
	}).Info("Conversation messages deleted successfully")

	return nil
}

// ========================================
// Helper Methods
// ========================================

// scanConversation сканирует строку БД в модель Conversation
func (s *SQLiteDB) scanConversation(row scanner) (*models.Conversation, error) {
	var conv models.Conversation
	var metadataJSON []byte

	var tenantID sql.NullString
	var temperature sql.NullFloat64
	var systemPrompt sql.NullString
	var lastMessageAt sql.NullTime

	err := row.Scan(
		&conv.ID,
		&conv.Title,
		&conv.UserID,
		&tenantID,
		&conv.Model,
		&temperature,
		&systemPrompt,
		&conv.Status,
		&conv.IsArchived,
		&conv.IsPinned,
		&conv.CreatedAt,
		&conv.UpdatedAt,
		&lastMessageAt,
		&conv.MessageCount,
		&conv.TotalTokens,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if tenantID.Valid {
		conv.TenantID = &tenantID.String
	}
	if temperature.Valid {
		conv.Temperature = &temperature.Float64
	}
	if systemPrompt.Valid {
		str := systemPrompt.String
		conv.SystemPrompt = str
	}
	if lastMessageAt.Valid {
		conv.LastMessageAt = &lastMessageAt.Time
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &conv.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &conv, nil
}

// scanMessage сканирует строку БД в модель Message
func (s *SQLiteDB) scanMessage(row scanner) (*models.Message, error) {
	var msg models.Message
	var toolCallsJSON []byte
	var metadataJSON []byte

	var model sql.NullString
	var temperature sql.NullFloat64
	var toolCallID sql.NullString

	err := row.Scan(
		&msg.ID,
		&msg.ConversationID,
		&msg.Role,
		&msg.Content,
		&model,
		&temperature,
		&toolCallsJSON,
		&toolCallID,
		&msg.PromptTokens,
		&msg.CompletionTokens,
		&msg.TotalTokens,
		&msg.CreatedAt,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if model.Valid {
		str := model.String
		msg.Model = str
	}
	if temperature.Valid {
		msg.Temperature = &temperature.Float64
	}
	if toolCallID.Valid {
		str := toolCallID.String
		msg.ToolCallID = str
	}

	// Deserialize tool_calls if present
	if len(toolCallsJSON) > 0 {
		if err := json.Unmarshal(toolCallsJSON, &msg.ToolCalls); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool_calls: %w", err)
		}
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &msg, nil
}
