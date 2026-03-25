// Package postgresql implements RAG database operations for PostgreSQL.
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// ========================================
// RAG Data Sources
// ========================================

// CreateRAGDataSource создает новый источник данных RAG
func (db *PostgreSQLDB) CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	tagsJSON, _ := json.Marshal(source.Tags)
	configJSON, _ := json.Marshal(source.Config)
	indexingConfigJSON, _ := json.Marshal(source.IndexingConfig)

	query := `
		INSERT INTO rag_data_sources (
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, sync_frequency,
			indexing_config, total_chunks, total_tokens, tags, is_shared,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`

	_, err := db.db.ExecContext(ctx, query,
		source.ID,
		source.UserID,
		source.TenantID,
		source.Name,
		source.Description,
		source.SourceType,
		configJSON,
		source.CredentialsEncrypted,
		source.Status,
		source.SyncFrequency,
		indexingConfigJSON,
		source.TotalChunks,
		source.TotalTokens,
		tagsJSON,
		source.IsShared,
		source.CreatedAt,
		source.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert data source: %w", err)
	}

	return nil
}

// GetRAGDataSource получает источник по ID
func (db *PostgreSQLDB) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
	query := `
		SELECT 
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, last_sync_at, last_sync_status,
			last_error, sync_frequency, indexing_config, total_chunks, total_tokens,
			last_chunk_count, tags, is_shared, created_at, updated_at
		FROM rag_data_sources
		WHERE id = $1
	`

	var source models.RAGDataSource
	var tenantID, lastSyncAt, lastSyncStatus, lastError, syncFrequency sql.NullString
	var lastChunkCount sql.NullInt64
	var tagsJSON, configJSON, indexingConfigJSON []byte

	err := db.db.QueryRowContext(ctx, query, id).Scan(
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
		&lastSyncStatus,
		&lastError,
		&syncFrequency,
		&indexingConfigJSON,
		&source.TotalChunks,
		&source.TotalTokens,
		&lastChunkCount,
		&tagsJSON,
		&source.IsShared,
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
		tidStr := tenantID.String
		source.TenantID = &tidStr
	}
	if lastSyncAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastSyncAt.String)
		source.LastSyncAt = &t
	}
	if lastSyncStatus.Valid {
		status := models.SyncStatus(lastSyncStatus.String)
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

	// Unmarshal JSON fields
	json.Unmarshal(tagsJSON, &source.Tags)
	json.Unmarshal(configJSON, &source.Config)
	json.Unmarshal(indexingConfigJSON, &source.IndexingConfig)

	return &source, nil
}

// UpdateRAGDataSource обновляет источник
func (db *PostgreSQLDB) UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	tagsJSON, _ := json.Marshal(source.Tags)
	configJSON, _ := json.Marshal(source.Config)
	indexingConfigJSON, _ := json.Marshal(source.IndexingConfig)

	query := `
		UPDATE rag_data_sources SET
			name = $1, description = $2, source_type = $3,
			config = $4, credentials_encrypted = $5, status = $6,
			last_sync_at = $7, last_sync_status = $8, last_error = $9,
			sync_frequency = $10, indexing_config = $11,
			total_chunks = $12, total_tokens = $13, last_chunk_count = $14,
			tags = $15, is_shared = $16, updated_at = $17
		WHERE id = $18
	`

	_, err := db.db.ExecContext(ctx, query,
		source.Name,
		source.Description,
		source.SourceType,
		configJSON,
		source.CredentialsEncrypted,
		source.Status,
		source.LastSyncAt,
		source.LastSyncStatus,
		source.LastError,
		source.SyncFrequency,
		indexingConfigJSON,
		source.TotalChunks,
		source.TotalTokens,
		source.LastChunkCount,
		tagsJSON,
		source.IsShared,
		source.UpdatedAt,
		source.ID,
	)

	return err
}

// DeleteRAGDataSource удаляет источник
func (db *PostgreSQLDB) DeleteRAGDataSource(ctx context.Context, id string) error {
	query := "DELETE FROM rag_data_sources WHERE id = $1"
	_, err := db.db.ExecContext(ctx, query, id)
	return err
}

// ListRAGDataSources возвращает список источников с фильтрацией
func (db *PostgreSQLDB) ListRAGDataSources(ctx context.Context, filter *storage.RAGDataSourceFilter) ([]*models.RAGDataSource, int, error) {
	if filter == nil {
		filter = &storage.RAGDataSourceFilter{}
	}

	// Build WHERE clause
	var whereClauses []string
	var args []any
	paramIndex := 1

	if filter.UserID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("user_id = $%d", paramIndex))
		args = append(args, *filter.UserID)
		paramIndex++
	}

	if filter.TenantID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("tenant_id = $%d", paramIndex))
		args = append(args, *filter.TenantID)
		paramIndex++
	}

	if filter.SourceType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("source_type = $%d", paramIndex))
		args = append(args, *filter.SourceType)
		paramIndex++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramIndex))
		args = append(args, *filter.Status)
		paramIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM rag_data_sources " + whereClause
	var total int
	err := db.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count data sources: %w", err)
	}

	// Query with pagination
	query := `
		SELECT 
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, last_sync_at, last_sync_status,
			last_error, sync_frequency, indexing_config, total_chunks, total_tokens,
			last_chunk_count, tags, is_shared, created_at, updated_at
		FROM rag_data_sources
		` + whereClause + `
		ORDER BY created_at DESC
	`

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", paramIndex)
		args = append(args, filter.Limit)
		paramIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramIndex)
		args = append(args, filter.Offset)
		paramIndex++
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query data sources: %w", err)
	}
	defer rows.Close()

	var sources []*models.RAGDataSource
	for rows.Next() {
		var source models.RAGDataSource
		var tenantID, lastSyncAt, lastSyncStatus, lastError, syncFrequency sql.NullString
		var lastChunkCount sql.NullInt64
		var tagsJSON, configJSON, indexingConfigJSON []byte

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
			&lastSyncStatus,
			&lastError,
			&syncFrequency,
			&indexingConfigJSON,
			&source.TotalChunks,
			&source.TotalTokens,
			&lastChunkCount,
			&tagsJSON,
			&source.IsShared,
			&source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan data source: %w", err)
		}

		// Parse nullable fields
		if tenantID.Valid {
			tidStr := tenantID.String
			source.TenantID = &tidStr
		}
		if lastSyncAt.Valid {
			t, _ := time.Parse(time.RFC3339, lastSyncAt.String)
			source.LastSyncAt = &t
		}
		if lastSyncStatus.Valid {
			status := models.SyncStatus(lastSyncStatus.String)
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

		// Unmarshal JSON fields
		json.Unmarshal(tagsJSON, &source.Tags)
		json.Unmarshal(configJSON, &source.Config)
		json.Unmarshal(indexingConfigJSON, &source.IndexingConfig)

		sources = append(sources, &source)
	}

	return sources, total, nil
}

// ========================================
// RAG Documents
// ========================================

// CreateRAGDocument создает новый документ RAG
func (db *PostgreSQLDB) CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	metadataJSON, _ := json.Marshal(doc.Metadata)

	query := `
		INSERT INTO rag_documents (
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := db.db.ExecContext(ctx, query,
		doc.ID,
		doc.SourceID,
		doc.Filename,
		doc.MimeType,
		doc.SizeBytes,
		doc.StorageBackend,
		doc.StoragePath,
		doc.StorageBucket,
		doc.Status,
		metadataJSON,
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	return err
}

// GetRAGDocument получает документ по ID
func (db *PostgreSQLDB) GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error) {
	query := `
		SELECT 
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, processing_started_at, processing_completed_at, processing_error,
			metadata, total_chunks, total_tokens, created_at, updated_at
		FROM rag_documents
		WHERE id = $1
	`

	var doc models.RAGDocument
	var processingStartedAt, processingCompletedAt, processingError, storageBucket sql.NullString
	var metadataJSON []byte

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&doc.SourceID,
		&doc.Filename,
		&doc.MimeType,
		&doc.SizeBytes,
		&doc.StorageBackend,
		&doc.StoragePath,
		&storageBucket,
		&doc.Status,
		&processingStartedAt,
		&processingCompletedAt,
		&processingError,
		&metadataJSON,
		&doc.TotalChunks,
		&doc.TotalTokens,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if storageBucket.Valid {
		doc.StorageBucket = storageBucket.String
	}
	if processingStartedAt.Valid {
		t, _ := time.Parse(time.RFC3339, processingStartedAt.String)
		doc.ProcessingStartedAt = &t
	}
	if processingCompletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, processingCompletedAt.String)
		doc.ProcessingCompletedAt = &t
	}
	if processingError.Valid {
		doc.ProcessingError = processingError.String
	}

	json.Unmarshal(metadataJSON, &doc.Metadata)

	return &doc, nil
}

// UpdateRAGDocument обновляет документ
func (db *PostgreSQLDB) UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	metadataJSON, _ := json.Marshal(doc.Metadata)

	query := `
		UPDATE rag_documents SET
			status = $1, processing_started_at = $2, processing_completed_at = $3,
			processing_error = $4, metadata = $5, total_chunks = $6, total_tokens = $7,
			updated_at = $8
		WHERE id = $9
	`

	_, err := db.db.ExecContext(ctx, query,
		doc.Status,
		doc.ProcessingStartedAt,
		doc.ProcessingCompletedAt,
		doc.ProcessingError,
		metadataJSON,
		doc.TotalChunks,
		doc.TotalTokens,
		doc.UpdatedAt,
		doc.ID,
	)

	return err
}

// DeleteRAGDocument удаляет документ
func (db *PostgreSQLDB) DeleteRAGDocument(ctx context.Context, id string) error {
	query := "DELETE FROM rag_documents WHERE id = $1"
	_, err := db.db.ExecContext(ctx, query, id)
	return err
}

// ListRAGDocuments возвращает список документов с фильтрацией
func (db *PostgreSQLDB) ListRAGDocuments(ctx context.Context, filter *storage.RAGDocumentFilter) ([]*models.RAGDocument, error) {
	if filter == nil {
		filter = &storage.RAGDocumentFilter{}
	}

	var whereClauses []string
	var args []any
	paramIndex := 1

	if filter.SourceID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("source_id = $%d", paramIndex))
		args = append(args, *filter.SourceID)
		paramIndex++
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", paramIndex))
		args = append(args, *filter.Status)
		paramIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := `
		SELECT 
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, processing_started_at, processing_completed_at, processing_error,
			metadata, total_chunks, total_tokens, created_at, updated_at
		FROM rag_documents
		` + whereClause + `
		ORDER BY created_at DESC
	`

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", paramIndex, paramIndex+1)
	args = append(args, limit, filter.Offset)

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []*models.RAGDocument
	for rows.Next() {
		var doc models.RAGDocument
		var processingStartedAt, processingCompletedAt, processingError, storageBucket sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&doc.ID,
			&doc.SourceID,
			&doc.Filename,
			&doc.MimeType,
			&doc.SizeBytes,
			&doc.StorageBackend,
			&doc.StoragePath,
			&storageBucket,
			&doc.Status,
			&processingStartedAt,
			&processingCompletedAt,
			&processingError,
			&metadataJSON,
			&doc.TotalChunks,
			&doc.TotalTokens,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		if storageBucket.Valid {
			doc.StorageBucket = storageBucket.String
		}
		if processingStartedAt.Valid {
			t, _ := time.Parse(time.RFC3339, processingStartedAt.String)
			doc.ProcessingStartedAt = &t
		}
		if processingCompletedAt.Valid {
			t, _ := time.Parse(time.RFC3339, processingCompletedAt.String)
			doc.ProcessingCompletedAt = &t
		}
		if processingError.Valid {
			doc.ProcessingError = processingError.String
		}

		json.Unmarshal(metadataJSON, &doc.Metadata)

		documents = append(documents, &doc)
	}

	return documents, nil
}

// ========================================
// RAG Chunks
// ========================================

// CreateRAGChunk создает новый chunk
func (db *PostgreSQLDB) CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	metadataJSON, _ := json.Marshal(chunk.Metadata)

	query := `
		INSERT INTO rag_chunks (
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := db.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.DocumentID,
		chunk.SourceID,
		chunk.ChunkText,
		chunk.ChunkIndex,
		chunk.ChunkTokens,
		metadataJSON,
		chunk.StartOffset,
		chunk.EndOffset,
		chunk.CreatedAt,
	)

	return err
}

// GetRAGChunk получает chunk по ID
func (db *PostgreSQLDB) GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error) {
	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE id = $1
	`

	var chunk models.RAGChunk
	var metadataJSON []byte
	var startOffset, endOffset sql.NullInt64

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&chunk.ID,
		&chunk.DocumentID,
		&chunk.SourceID,
		&chunk.ChunkText,
		&chunk.ChunkIndex,
		&chunk.ChunkTokens,
		&metadataJSON,
		&startOffset,
		&endOffset,
		&chunk.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chunk not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk: %w", err)
	}

	if startOffset.Valid {
		offset := int(startOffset.Int64)
		chunk.StartOffset = &offset
	}
	if endOffset.Valid {
		offset := int(endOffset.Int64)
		chunk.EndOffset = &offset
	}

	json.Unmarshal(metadataJSON, &chunk.Metadata)

	return &chunk, nil
}

// ListRAGChunksByDocument возвращает все chunks документа
func (db *PostgreSQLDB) ListRAGChunksByDocument(ctx context.Context, documentID string) ([]*models.RAGChunk, error) {
	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE document_id = $1
		ORDER BY chunk_index
	`

	rows, err := db.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.RAGChunk
	for rows.Next() {
		var chunk models.RAGChunk
		var metadataJSON []byte
		var startOffset, endOffset sql.NullInt64

		err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.SourceID,
			&chunk.ChunkText,
			&chunk.ChunkIndex,
			&chunk.ChunkTokens,
			&metadataJSON,
			&startOffset,
			&endOffset,
			&chunk.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		if startOffset.Valid {
			offset := int(startOffset.Int64)
			chunk.StartOffset = &offset
		}
		if endOffset.Valid {
			offset := int(endOffset.Int64)
			chunk.EndOffset = &offset
		}

		json.Unmarshal(metadataJSON, &chunk.Metadata)

		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

// ListRAGChunksBySource возвращает chunks источника с пагинацией
func (db *PostgreSQLDB) ListRAGChunksBySource(ctx context.Context, sourceID string, limit, offset int) ([]*models.RAGChunk, error) {
	if limit <= 0 {
		limit = 100
	}

	db.logger.WithFields(map[string]any{
		"source_id": sourceID,
		"limit":     limit,
		"offset":    offset,
	}).Debug("ListRAGChunksBySource called")

	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE source_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.db.QueryContext(ctx, query, sourceID, limit, offset)
	if err != nil {
		db.logger.WithError(err).Error("Failed to query chunks from database")
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.RAGChunk
	rowCount := 0
	for rows.Next() {
		rowCount++
		var chunk models.RAGChunk
		var metadataJSON []byte
		var startOffset, endOffset sql.NullInt64

		err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.SourceID,
			&chunk.ChunkText,
			&chunk.ChunkIndex,
			&chunk.ChunkTokens,
			&metadataJSON,
			&startOffset,
			&endOffset,
			&chunk.CreatedAt,
		)

		if err != nil {
			db.logger.WithError(err).Error("Failed to scan chunk row")
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}

		if startOffset.Valid {
			offset := int(startOffset.Int64)
			chunk.StartOffset = &offset
		}
		if endOffset.Valid {
			offset := int(endOffset.Int64)
			chunk.EndOffset = &offset
		}

		json.Unmarshal(metadataJSON, &chunk.Metadata)

		chunks = append(chunks, &chunk)
	}

	db.logger.WithFields(map[string]any{
		"source_id":       sourceID,
		"rows_scanned":    rowCount,
		"chunks_returned": len(chunks),
	}).Info("ListRAGChunksBySource completed")

	return chunks, nil
}

// DeleteRAGChunksByDocument удаляет все chunks документа
func (db *PostgreSQLDB) DeleteRAGChunksByDocument(ctx context.Context, documentID string) error {
	query := "DELETE FROM rag_chunks WHERE document_id = $1"
	_, err := db.db.ExecContext(ctx, query, documentID)
	return err
}

// DeleteChunksBySource удаляет все chunks для указанного источника
func (db *PostgreSQLDB) DeleteChunksBySource(ctx context.Context, sourceID string) error {
	// Также удаляем связанные векторы
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Удаляем векторы
	_, err = tx.ExecContext(ctx, "DELETE FROM rag_vectors WHERE source_id = $1", sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}

	// Удаляем chunks
	_, err = tx.ExecContext(ctx, "DELETE FROM rag_chunks WHERE source_id = $1", sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	// Удаляем документы
	_, err = tx.ExecContext(ctx, "DELETE FROM rag_documents WHERE source_id = $1", sourceID)
	if err != nil {
		return fmt.Errorf("failed to delete documents: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RAG Jobs Queue and Query Logs are implemented in rag_jobs_logs.go
