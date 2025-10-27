// Package sqlite implements SQLite storage backend.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"aigateway/internal/models"
	"aigateway/internal/storage"

	"github.com/google/uuid"
)

// Verify that SQLiteDB implements RAGDataSourceRepository
var _ storage.RAGDataSourceRepository = (*SQLiteDB)(nil)

// CreateDataSource создает новый источник данных
func (s *SQLiteDB) CreateDataSource(ctx context.Context, source *models.RAGDataSource) error {
	// Convert arrays/objects to JSON
	tagsJSON, err := json.Marshal(source.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	configJSON, err := json.Marshal(source.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	indexingConfigJSON, err := json.Marshal(source.IndexingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal indexing_config: %w", err)
	}

	query := `
		INSERT INTO rag_data_sources (
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, sync_frequency,
			indexing_config, total_chunks, total_tokens, tags, is_shared,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		source.ID.String(),
		source.UserID.String(),
		nullableUUID(source.TenantID),
		source.Name,
		source.Description,
		source.SourceType,
		string(configJSON),
		source.CredentialsEncrypted,
		source.Status,
		source.SyncFrequency,
		string(indexingConfigJSON),
		source.TotalChunks,
		source.TotalTokens,
		string(tagsJSON),
		boolToInt(source.IsShared),
		source.CreatedAt,
		source.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert data source: %w", err)
	}

	return nil
}

// GetDataSourceByID получает источник по ID
func (s *SQLiteDB) GetDataSourceByID(ctx context.Context, id uuid.UUID) (*models.RAGDataSource, error) {
	query := `
		SELECT 
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, last_sync_at, last_sync_status,
			last_error, sync_frequency, indexing_config, total_chunks, total_tokens,
			last_chunk_count, tags, is_shared, created_at, updated_at
		FROM rag_data_sources
		WHERE id = ?
	`

	var source models.RAGDataSource
	var tenantID sql.NullString
	var lastSyncAt sql.NullTime
	var lastSyncStatusStr sql.NullString
	var lastError sql.NullString
	var syncFrequency sql.NullString
	var lastChunkCount sql.NullInt64
	var tagsJSON, configJSON, indexingConfigJSON string
	var isSharedInt int

	err := s.db.QueryRowContext(ctx, query, id.String()).Scan(
		&source.ID,
		&source.UserID,
		&tenantID,
		&source.Name,
		&source.Description,
		&source.SourceType,
		&configJSON,
		&source.CredentialsEncrypted,
		&source.Status,
		&lastSyncAt,
		&lastSyncStatusStr,
		&lastError,
		&syncFrequency,
		&indexingConfigJSON,
		&source.TotalChunks,
		&source.TotalTokens,
		&lastChunkCount,
		&tagsJSON,
		&isSharedInt,
		&source.CreatedAt,
		&source.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("data source not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get data source: %w", err)
	}

	// Parse nullable fields
	if tenantID.Valid {
		tid, _ := uuid.Parse(tenantID.String)
		source.TenantID = &tid
	}
	if lastSyncAt.Valid {
		source.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatusStr.Valid {
		status := models.SyncStatus(lastSyncStatusStr.String)
		source.LastSyncStatus = &status
	}
	if lastError.Valid {
		source.LastError = lastError.String
	}
	if syncFrequency.Valid {
		source.SyncFrequency = &syncFrequency.String
	}
	if lastChunkCount.Valid {
		count := int(lastChunkCount.Int64)
		source.LastChunkCount = count
	}

	source.IsShared = intToBool(isSharedInt)

	// Unmarshal JSON fields
	if err := json.Unmarshal([]byte(tagsJSON), &source.Tags); err != nil {
		source.Tags = []string{}
	}
	if err := json.Unmarshal([]byte(configJSON), &source.Config); err != nil {
		source.Config = make(models.SourceConfig)
	}
	if err := json.Unmarshal([]byte(indexingConfigJSON), &source.IndexingConfig); err != nil {
		source.IndexingConfig = models.IndexingConfig{}
	}

	return &source, nil
}

// ListDataSources возвращает список источников с фильтрацией
func (s *SQLiteDB) ListDataSources(ctx context.Context, filter storage.DataSourceFilter) ([]models.RAGDataSource, int, error) {
	// Build WHERE clause
	var whereClauses []string
	var args []interface{}

	if filter.UserID != nil {
		whereClauses = append(whereClauses, "user_id = ?")
		args = append(args, filter.UserID.String())
	}

	if filter.TenantID != nil {
		whereClauses = append(whereClauses, "tenant_id = ?")
		args = append(args, filter.TenantID.String())
	}

	if filter.SourceType != nil {
		whereClauses = append(whereClauses, "source_type = ?")
		args = append(args, string(*filter.SourceType))
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, string(*filter.Status))
	}

	// TODO: implement tags filtering (requires JSON operations)

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM rag_data_sources %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count data sources: %w", err)
	}

	// Get data
	query := fmt.Sprintf(`
		SELECT 
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, last_sync_at, last_sync_status,
			last_error, sync_frequency, indexing_config, total_chunks, total_tokens,
			last_chunk_count, tags, is_shared, created_at, updated_at
		FROM rag_data_sources
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query data sources: %w", err)
	}
	defer rows.Close()

	var sources []models.RAGDataSource
	for rows.Next() {
		var source models.RAGDataSource
		var tenantID sql.NullString
		var lastSyncAt sql.NullTime
		var lastSyncStatusStr sql.NullString
		var lastError sql.NullString
		var syncFrequency sql.NullString
		var lastChunkCount sql.NullInt64
		var tagsJSON, configJSON, indexingConfigJSON string
		var isSharedInt int

		err := rows.Scan(
			&source.ID,
			&source.UserID,
			&tenantID,
			&source.Name,
			&source.Description,
			&source.SourceType,
			&configJSON,
			&source.CredentialsEncrypted,
			&source.Status,
			&lastSyncAt,
			&lastSyncStatusStr,
			&lastError,
			&syncFrequency,
			&indexingConfigJSON,
			&source.TotalChunks,
			&source.TotalTokens,
			&lastChunkCount,
			&tagsJSON,
			&isSharedInt,
			&source.CreatedAt,
			&source.UpdatedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan data source: %w", err)
		}

		// Parse nullable fields
		if tenantID.Valid {
			tid, _ := uuid.Parse(tenantID.String)
			source.TenantID = &tid
		}
		if lastSyncAt.Valid {
			source.LastSyncAt = &lastSyncAt.Time
		}
		if lastSyncStatusStr.Valid {
			status := models.SyncStatus(lastSyncStatusStr.String)
			source.LastSyncStatus = &status
		}
		if lastError.Valid {
			source.LastError = lastError.String
		}
		if syncFrequency.Valid {
			source.SyncFrequency = &syncFrequency.String
		}
		if lastChunkCount.Valid {
			source.LastChunkCount = int(lastChunkCount.Int64)
		}

		source.IsShared = intToBool(isSharedInt)

		// Unmarshal JSON
		if err := json.Unmarshal([]byte(tagsJSON), &source.Tags); err != nil {
			source.Tags = []string{}
		}
		json.Unmarshal([]byte(configJSON), &source.Config)
		json.Unmarshal([]byte(indexingConfigJSON), &source.IndexingConfig)

		sources = append(sources, source)
	}

	return sources, total, nil
}

// UpdateDataSource обновляет источник
func (s *SQLiteDB) UpdateDataSource(ctx context.Context, source *models.RAGDataSource) error {
	tagsJSON, _ := json.Marshal(source.Tags)
	configJSON, _ := json.Marshal(source.Config)
	indexingConfigJSON, _ := json.Marshal(source.IndexingConfig)

	query := `
		UPDATE rag_data_sources SET
			name = ?, description = ?, source_type = ?,
			config = ?, credentials_encrypted = ?, status = ?,
			last_sync_at = ?, last_sync_status = ?, last_error = ?,
			sync_frequency = ?, indexing_config = ?,
			total_chunks = ?, total_tokens = ?, last_chunk_count = ?,
			tags = ?, is_shared = ?, updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		source.Name,
		source.Description,
		source.SourceType,
		string(configJSON),
		source.CredentialsEncrypted,
		source.Status,
		source.LastSyncAt,
		nullableString((*string)(source.LastSyncStatus)),
		source.LastError,
		source.SyncFrequency,
		string(indexingConfigJSON),
		source.TotalChunks,
		source.TotalTokens,
		source.LastChunkCount,
		string(tagsJSON),
		boolToInt(source.IsShared),
		source.UpdatedAt,
		source.ID.String(),
	)

	return err
}

// DeleteDataSource удаляет источник
func (s *SQLiteDB) DeleteDataSource(ctx context.Context, id uuid.UUID) error {
	query := "DELETE FROM rag_data_sources WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, id.String())
	return err
}

// UpdateSourceStatus обновляет статус источника
func (s *SQLiteDB) UpdateSourceStatus(ctx context.Context, id uuid.UUID, status models.SourceStatus, errorMsg string) error {
	query := "UPDATE rag_data_sources SET status = ?, last_error = ?, updated_at = ? WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, string(status), errorMsg, sql.NullTime{}, id.String())
	return err
}

// UpdateSyncInfo обновляет информацию о синхронизации
func (s *SQLiteDB) UpdateSyncInfo(ctx context.Context, id uuid.UUID, status models.SyncStatus, chunkCount int) error {
	query := `
		UPDATE rag_data_sources 
		SET last_sync_at = ?, last_sync_status = ?, last_chunk_count = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, sql.NullTime{}, string(status), chunkCount, sql.NullTime{}, id.String())
	return err
}

// UpdateStatistics обновляет статистику источника
func (s *SQLiteDB) UpdateStatistics(ctx context.Context, id uuid.UUID, totalChunks int, totalTokens int64) error {
	query := "UPDATE rag_data_sources SET total_chunks = ?, total_tokens = ?, updated_at = ? WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, totalChunks, totalTokens, sql.NullTime{}, id.String())
	return err
}

// Helper functions
func nullableUUID(id *uuid.UUID) interface{} {
	if id == nil {
		return nil
	}
	return id.String()
}

func nullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// boolToInt and intToBool helpers moved to rag.go to avoid redeclaration

