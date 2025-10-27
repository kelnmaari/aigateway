// Package sqlite provides SQLite implementation of audit-related database operations
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// ========================================
// Audit Events CRUD Operations
// ========================================

// CreateAuditEvent создает новое событие аудита
func (s *SQLiteDB) CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("event_type", event.EventType).Debug("Creating audit event")

	// Serialize Metadata map to JSON
	var metadataJSON []byte
	var err error
	if len(event.Metadata) > 0 {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
		}
	}

	query := `
		INSERT INTO audit_events (
			id, event_type, severity,
			actor_id, actor_type,
			target_id, target_type,
			action, resource, status, error_msg, metadata,
			ip_address, user_agent,
			timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		event.ID,
		event.EventType,
		event.Severity,
		event.ActorID,
		event.ActorType,
		event.TargetID,
		event.TargetType,
		event.Action,
		event.Resource,
		event.Status,
		event.ErrorMsg,
		metadataJSON, // Use JSON bytes instead of map
		event.IPAddress,
		event.UserAgent,
		event.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit event: %w", err)
	}

	s.logger.WithField("event_id", event.ID).Debug("Audit event created successfully")
	return nil
}

// GetAuditEvents возвращает список событий аудита с фильтрами
func (s *SQLiteDB) GetAuditEvents(ctx context.Context, filters storage.AuditFilters) ([]*models.AuditEvent, int, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("database not connected")
	}

	s.logger.WithField("filters", filters).Debug("Getting audit events")

	// Build base query with filters
	query := `
		SELECT 
			id, event_type, severity,
			actor_id, actor_type,
			target_id, target_type,
			action, resource, status, error_msg, metadata,
			ip_address, user_agent,
			timestamp
		FROM audit_events
		WHERE 1=1
	`

	var args []interface{}

	// Apply filters
	if filters.EventType != "" {
		query += " AND event_type = ?"
		args = append(args, filters.EventType)
	}

	if filters.ActorID != "" {
		query += " AND actor_id = ?"
		args = append(args, filters.ActorID)
	}

	if filters.Resource != "" {
		query += " AND resource = ?"
		args = append(args, filters.Resource)
	}

	if filters.Severity != "" {
		query += " AND severity = ?"
		args = append(args, filters.Severity)
	}

	if filters.Status != "" {
		query += " AND status = ?"
		args = append(args, filters.Status)
	}

	if !filters.FromDate.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filters.FromDate)
	}

	if !filters.ToDate.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filters.ToDate)
	}

	// Get total count before applying limit/offset
	countQuery := "SELECT COUNT(*) FROM (" + query + ")"
	var total int
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Add ordering
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []*models.AuditEvent
	for rows.Next() {
		event, err := s.scanAuditEvent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	s.logger.WithField("count", len(events)).Debug("Audit events retrieved successfully")
	return events, total, nil
}

// DeleteOldAuditEvents удаляет события аудита старше указанного времени (retention policy)
func (s *SQLiteDB) DeleteOldAuditEvents(ctx context.Context, olderThan time.Time) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not connected")
	}

	s.logger.WithField("older_than", olderThan).Debug("Deleting old audit events")

	query := `DELETE FROM audit_events WHERE timestamp < ?`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	s.logger.WithField("deleted_count", rowsAffected).Info("Old audit events deleted")
	return int(rowsAffected), nil
}

// ========================================
// Helper Functions
// ========================================

// scanAuditEvent сканирует строку БД в модель AuditEvent
func (s *SQLiteDB) scanAuditEvent(row scanner) (*models.AuditEvent, error) {
	var event models.AuditEvent
	var metadataJSON []byte

	err := row.Scan(
		&event.ID,
		&event.EventType,
		&event.Severity,
		&event.ActorID,
		&event.ActorType,
		&event.TargetID,
		&event.TargetType,
		&event.Action,
		&event.Resource,
		&event.Status,
		&event.ErrorMsg,
		&metadataJSON, // Scan as JSON bytes
		&event.IPAddress,
		&event.UserAgent,
		&event.Timestamp,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize metadata JSON to map
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata JSON: %w", err)
		}
	}

	return &event, nil
}

