// Package sqlite provides SQLite implementation of audit-related database operations
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package postgresql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// ========================================
// Audit Events CRUD Operations
// ========================================

// CreateAuditEvent создает новое событие аудита
func (db *PostgreSQLDB) CreateAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("event_type", event.EventType).Debug("Creating audit event")

	// Serialize Metadata map to JSON - ensure empty object if nil/empty
	var metadataJSON []byte
	var err error
	if event.Metadata != nil && len(event.Metadata) > 0 {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_events (
			id, event_type, severity,
			actor_id, actor_type,
			target_id, target_type,
			action, resource, status, error_msg, metadata,
			ip_address, user_agent,
			timestamp
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = db.db.ExecContext(ctx, query,
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

	db.logger.WithField("event_id", event.ID).Debug("Audit event created successfully")
	return nil
}

// GetAuditEvents возвращает список событий аудита с фильтрами
func (db *PostgreSQLDB) GetAuditEvents(ctx context.Context, filters storage.AuditFilters) ([]*models.AuditEvent, int, error) {
	if db.db == nil {
		return nil, 0, fmt.Errorf("database not connected")
	}

	db.logger.WithField("filters", filters).Debug("Getting audit events")

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
	paramIndex := 1 // PostgreSQL uses $1, $2, $3...

	// Apply filters
	if filters.EventType != "" {
		query += fmt.Sprintf(" AND event_type = $%d", paramIndex)
		args = append(args, filters.EventType)
		paramIndex++
	}

	if filters.ActorID != "" {
		query += fmt.Sprintf(" AND actor_id = $%d", paramIndex)
		args = append(args, filters.ActorID)
		paramIndex++
	}

	if filters.Resource != "" {
		query += fmt.Sprintf(" AND resource = $%d", paramIndex)
		args = append(args, filters.Resource)
		paramIndex++
	}

	if filters.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", paramIndex)
		args = append(args, filters.Severity)
		paramIndex++
	}

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", paramIndex)
		args = append(args, filters.Status)
		paramIndex++
	}

	if !filters.FromDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", paramIndex)
		args = append(args, filters.FromDate)
		paramIndex++
	}

	if !filters.ToDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", paramIndex)
		args = append(args, filters.ToDate)
		paramIndex++
	}

	// Get total count before applying limit/offset
	countQuery := "SELECT COUNT(*) FROM (" + query + ") AS counted"
	var total int
	err := db.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit events: %w", err)
	}

	// Add ordering
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", paramIndex)
		args = append(args, filters.Limit)
		paramIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramIndex)
		args = append(args, filters.Offset)
		paramIndex++
	}

	// Execute query
	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer rows.Close()

	var events []*models.AuditEvent
	for rows.Next() {
		event, err := db.scanAuditEvent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit event: %w", err)
		}
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	db.logger.WithField("count", len(events)).Debug("Audit events retrieved successfully")
	return events, total, nil
}

// DeleteOldAuditEvents удаляет события аудита старше указанного времени (retention policy)
func (db *PostgreSQLDB) DeleteOldAuditEvents(ctx context.Context, olderThan time.Time) (int, error) {
	if db.db == nil {
		return 0, fmt.Errorf("database not connected")
	}

	db.logger.WithField("older_than", olderThan).Debug("Deleting old audit events")

	query := `DELETE FROM audit_events WHERE timestamp < $1`

	result, err := db.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit events: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	db.logger.WithField("deleted_count", rowsAffected).Info("Old audit events deleted")
	return int(rowsAffected), nil
}

// ========================================
// Helper Functions
// ========================================

// scanAuditEvent сканирует строку БД в модель AuditEvent
func (db *PostgreSQLDB) scanAuditEvent(row scanner) (*models.AuditEvent, error) {
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
