// Package sqlite implements RAG database operations for SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// ========================================
// RAG Data Sources
// ========================================

// CreateRAGDataSource создает новый источник данных RAG
func (s *SQLiteDB) CreateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
	tagsJSON, _ := json.Marshal(source.Tags)
	configJSON, _ := json.Marshal(source.Config)
	indexingConfigJSON, _ := json.Marshal(source.IndexingConfig)

	query := `
		INSERT INTO rag_data_sources (
			id, user_id, tenant_id, name, description, source_type,
			config, credentials_encrypted, status, sync_frequency,
			indexing_config, total_chunks, total_tokens, tags, is_shared,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		source.ID,
		source.UserID,
		source.TenantID,
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

// GetRAGDataSource получает источник по ID
func (s *SQLiteDB) GetRAGDataSource(ctx context.Context, id string) (*models.RAGDataSource, error) {
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
	var tenantID, lastSyncAt, lastSyncStatus, lastError, syncFrequency sql.NullString
	var lastChunkCount sql.NullInt64
	var tagsJSON, configJSON, indexingConfigJSON string
	var isSharedInt int

	err := s.db.QueryRowContext(ctx, query, id).Scan(
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

	source.IsShared = intToBool(isSharedInt)

	// Unmarshal JSON fields
	json.Unmarshal([]byte(tagsJSON), &source.Tags)
	json.Unmarshal([]byte(configJSON), &source.Config)
	json.Unmarshal([]byte(indexingConfigJSON), &source.IndexingConfig)

	return &source, nil
}

// UpdateRAGDataSource обновляет источник
func (s *SQLiteDB) UpdateRAGDataSource(ctx context.Context, source *models.RAGDataSource) error {
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
		source.LastSyncStatus,
		source.LastError,
		source.SyncFrequency,
		string(indexingConfigJSON),
		source.TotalChunks,
		source.TotalTokens,
		source.LastChunkCount,
		string(tagsJSON),
		boolToInt(source.IsShared),
		source.UpdatedAt,
		source.ID,
	)

	return err
}

// DeleteRAGDataSource удаляет источник
func (s *SQLiteDB) DeleteRAGDataSource(ctx context.Context, id string) error {
	query := "DELETE FROM rag_data_sources WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ListRAGDataSources возвращает список источников с фильтрацией
func (s *SQLiteDB) ListRAGDataSources(ctx context.Context, filter *storage.RAGDataSourceFilter) ([]*models.RAGDataSource, int, error) {
	if filter == nil {
		filter = &storage.RAGDataSourceFilter{}
	}

	// Build WHERE clause
	var whereClauses []string
	var args []interface{}

	if filter.UserID != nil {
		whereClauses = append(whereClauses, "user_id = ?")
		args = append(args, *filter.UserID)
	}

	if filter.TenantID != nil {
		whereClauses = append(whereClauses, "tenant_id = ?")
		args = append(args, *filter.TenantID)
	}

	if filter.SourceType != nil {
		whereClauses = append(whereClauses, "source_type = ?")
		args = append(args, *filter.SourceType)
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, *filter.Status)
	}

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

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query data sources: %w", err)
	}
	defer rows.Close()

	var sources []*models.RAGDataSource
	for rows.Next() {
		var source models.RAGDataSource
		var tenantID, lastSyncAt, lastSyncStatus, lastError, syncFrequency sql.NullString
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
			&lastSyncStatus,
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

		source.IsShared = intToBool(isSharedInt)

		// Unmarshal JSON
		json.Unmarshal([]byte(tagsJSON), &source.Tags)
		json.Unmarshal([]byte(configJSON), &source.Config)
		json.Unmarshal([]byte(indexingConfigJSON), &source.IndexingConfig)

		sources = append(sources, &source)
	}

	return sources, total, nil
}

// ========================================
// RAG Documents
// ========================================

// CreateRAGDocument создает новый документ RAG
func (s *SQLiteDB) CreateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	metadataJSON, _ := json.Marshal(doc.Metadata)

	query := `
		INSERT INTO rag_documents (
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, metadata, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		doc.ID,
		doc.SourceID,
		doc.Filename,
		doc.MimeType,
		doc.SizeBytes,
		doc.StorageBackend,
		doc.StoragePath,
		doc.StorageBucket,
		doc.Status,
		string(metadataJSON),
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	return err
}

// GetRAGDocument получает документ по ID
func (s *SQLiteDB) GetRAGDocument(ctx context.Context, id string) (*models.RAGDocument, error) {
	query := `
		SELECT 
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, processing_started_at, processing_completed_at, processing_error,
			metadata, total_chunks, total_tokens, created_at, updated_at
		FROM rag_documents
		WHERE id = ?
	`

	var doc models.RAGDocument
	var processingStartedAt, processingCompletedAt, processingError, storageBucket sql.NullString
	var metadataJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
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

	json.Unmarshal([]byte(metadataJSON), &doc.Metadata)

	return &doc, nil
}

// UpdateRAGDocument обновляет документ
func (s *SQLiteDB) UpdateRAGDocument(ctx context.Context, doc *models.RAGDocument) error {
	metadataJSON, _ := json.Marshal(doc.Metadata)

	query := `
		UPDATE rag_documents SET
			status = ?, processing_started_at = ?, processing_completed_at = ?,
			processing_error = ?, metadata = ?, total_chunks = ?, total_tokens = ?,
			updated_at = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		doc.Status,
		doc.ProcessingStartedAt,
		doc.ProcessingCompletedAt,
		doc.ProcessingError,
		string(metadataJSON),
		doc.TotalChunks,
		doc.TotalTokens,
		doc.UpdatedAt,
		doc.ID,
	)

	return err
}

// DeleteRAGDocument удаляет документ
func (s *SQLiteDB) DeleteRAGDocument(ctx context.Context, id string) error {
	query := "DELETE FROM rag_documents WHERE id = ?"
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// ListRAGDocuments возвращает список документов с фильтрацией
func (s *SQLiteDB) ListRAGDocuments(ctx context.Context, filter *storage.RAGDocumentFilter) ([]*models.RAGDocument, error) {
	if filter == nil {
		filter = &storage.RAGDocumentFilter{}
	}

	var whereClauses []string
	var args []interface{}

	if filter.SourceID != nil {
		whereClauses = append(whereClauses, "source_id = ?")
		args = append(args, *filter.SourceID)
	}

	if filter.Status != nil {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, *filter.Status)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT 
			id, source_id, filename, mime_type, size_bytes,
			storage_backend, storage_path, storage_bucket,
			status, processing_started_at, processing_completed_at, processing_error,
			metadata, total_chunks, total_tokens, created_at, updated_at
		FROM rag_documents
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	args = append(args, limit, filter.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var docs []*models.RAGDocument
	for rows.Next() {
		var doc models.RAGDocument
		var processingStartedAt, processingCompletedAt, processingError, storageBucket sql.NullString
		var metadataJSON string

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

		json.Unmarshal([]byte(metadataJSON), &doc.Metadata)

		docs = append(docs, &doc)
	}

	return docs, nil
}

// ========================================
// RAG Chunks
// ========================================

// CreateRAGChunk создает новый chunk
func (s *SQLiteDB) CreateRAGChunk(ctx context.Context, chunk *models.RAGChunk) error {
	metadataJSON, _ := json.Marshal(chunk.Metadata)

	query := `
		INSERT INTO rag_chunks (
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.DocumentID,
		chunk.SourceID,
		chunk.ChunkText,
		chunk.ChunkIndex,
		chunk.ChunkTokens,
		string(metadataJSON),
		chunk.StartOffset,
		chunk.EndOffset,
		chunk.CreatedAt,
	)

	return err
}

// GetRAGChunk получает chunk по ID
func (s *SQLiteDB) GetRAGChunk(ctx context.Context, id string) (*models.RAGChunk, error) {
	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE id = ?
	`

	var chunk models.RAGChunk
	var metadataJSON string
	var startOffset, endOffset sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
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

	json.Unmarshal([]byte(metadataJSON), &chunk.Metadata)

	return &chunk, nil
}

// ListRAGChunksByDocument возвращает все chunks документа
func (s *SQLiteDB) ListRAGChunksByDocument(ctx context.Context, documentID string) ([]*models.RAGChunk, error) {
	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE document_id = ?
		ORDER BY chunk_index
	`

	rows, err := s.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.RAGChunk
	for rows.Next() {
		var chunk models.RAGChunk
		var metadataJSON string
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

		json.Unmarshal([]byte(metadataJSON), &chunk.Metadata)

		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

// ListRAGChunksBySource возвращает chunks источника с пагинацией
func (s *SQLiteDB) ListRAGChunksBySource(ctx context.Context, sourceID string, limit, offset int) ([]*models.RAGChunk, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT 
			id, document_id, source_id, chunk_text, chunk_index, chunk_tokens,
			metadata, start_offset, end_offset, created_at
		FROM rag_chunks
		WHERE source_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, sourceID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query chunks: %w", err)
	}
	defer rows.Close()

	var chunks []*models.RAGChunk
	for rows.Next() {
		var chunk models.RAGChunk
		var metadataJSON string
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

		json.Unmarshal([]byte(metadataJSON), &chunk.Metadata)

		chunks = append(chunks, &chunk)
	}

	return chunks, nil
}

// DeleteRAGChunksByDocument удаляет все chunks документа
func (s *SQLiteDB) DeleteRAGChunksByDocument(ctx context.Context, documentID string) error {
	query := "DELETE FROM rag_chunks WHERE document_id = ?"
	_, err := s.db.ExecContext(ctx, query, documentID)
	return err
}

// ========================================
// RAG Jobs Queue
// ========================================

// CreateRAGJob создает новую задачу в очереди
func (s *SQLiteDB) CreateRAGJob(ctx context.Context, job *models.RAGJob) error {
	query := `
		INSERT INTO rag_jobs (
			job_type, status, payload, priority, attempts, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		job.JobType,
		job.Status,
		job.Payload,
		job.Priority,
		job.Attempts,
		job.CreatedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	job.ID = id
	return nil
}

// GetRAGJob получает задачу по ID
func (s *SQLiteDB) GetRAGJob(ctx context.Context, id string) (*models.RAGJob, error) {
	query := `
		SELECT 
			id, job_type, status, payload, result, priority, attempts,
			created_at, started_at, completed_at, error, locked_until
		FROM rag_jobs
		WHERE id = ?
	`

	var job models.RAGJob
	var startedAt, completedAt, errorMsg, lockedUntil sql.NullString
	var resultData sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.Payload,
		&resultData,
		&job.Priority,
		&job.Attempts,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&errorMsg,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if resultData.Valid {
		result := new(models.JobResult)
		json.Unmarshal([]byte(resultData.String), result)
		job.Result = result
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		job.LockedUntil = &t
	}

	return &job, nil
}

// UpdateRAGJob обновляет задачу
func (s *SQLiteDB) UpdateRAGJob(ctx context.Context, job *models.RAGJob) error {
	query := `
		UPDATE rag_jobs SET
			status = ?, result = ?, attempts = ?, started_at = ?,
			completed_at = ?, error = ?, locked_until = ?
		WHERE id = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		job.Status,
		job.Result,
		job.Attempts,
		job.StartedAt,
		job.CompletedAt,
		job.Error,
		job.LockedUntil,
		job.ID,
	)

	return err
}

// GetNextPendingRAGJob получает следующую pending задачу с блокировкой
func (s *SQLiteDB) GetNextPendingRAGJob(ctx context.Context) (*models.RAGJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Для SQLite используем простую логику: берем первую pending задачу
	query := `
		SELECT 
			id, job_type, status, payload, result, priority, attempts,
			created_at, started_at, completed_at, error, locked_until
		FROM rag_jobs
		WHERE status = 'pending' 
		AND (locked_until IS NULL OR locked_until < datetime('now'))
		ORDER BY priority DESC, created_at ASC
		LIMIT 1
	`

	var job models.RAGJob
	var startedAt, completedAt, errorMsg, lockedUntil sql.NullString
	var resultData sql.NullString

	err := s.db.QueryRowContext(ctx, query).Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&job.Payload,
		&resultData,
		&job.Priority,
		&job.Attempts,
		&job.CreatedAt,
		&startedAt,
		&completedAt,
		&errorMsg,
		&lockedUntil,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next job: %w", err)
	}

	if resultData.Valid {
		result := new(models.JobResult)
		json.Unmarshal([]byte(resultData.String), result)
		job.Result = result
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t, _ := time.Parse(time.RFC3339, completedAt.String)
		job.CompletedAt = &t
	}
	if errorMsg.Valid {
		job.Error = errorMsg.String
	}
	if lockedUntil.Valid {
		t, _ := time.Parse(time.RFC3339, lockedUntil.String)
		job.LockedUntil = &t
	}

	// Блокируем задачу на 5 минут
	lockUntil := time.Now().Add(5 * time.Minute)
	_, err = s.db.ExecContext(ctx, "UPDATE rag_jobs SET locked_until = ? WHERE id = ?", lockUntil, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	job.LockedUntil = &lockUntil

	return &job, nil
}

// CountRAGJobsByStatus подсчитывает задачи по статусу
func (s *SQLiteDB) CountRAGJobsByStatus(ctx context.Context, status string) (int, error) {
	query := "SELECT COUNT(*) FROM rag_jobs WHERE status = ?"
	
	var count int
	err := s.db.QueryRowContext(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}

	return count, nil
}

// DeleteOldRAGJobs удаляет старые задачи
func (s *SQLiteDB) DeleteOldRAGJobs(ctx context.Context, cutoffTime time.Time, statuses []string) (int, error) {
	if len(statuses) == 0 {
		return 0, nil
	}

	placeholders := strings.Repeat("?,", len(statuses)-1) + "?"
	query := fmt.Sprintf(`
		DELETE FROM rag_jobs
		WHERE created_at < ? AND status IN (%s)
	`, placeholders)

	args := []interface{}{cutoffTime}
	for _, status := range statuses {
		args = append(args, status)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(affected), nil
}

// UnlockExpiredRAGJobs разблокирует задачи с истекшим locked_until
func (s *SQLiteDB) UnlockExpiredRAGJobs(ctx context.Context, now time.Time) (int, error) {
	query := `
		UPDATE rag_jobs
		SET locked_until = NULL, status = 'pending'
		WHERE status = 'processing' AND locked_until IS NOT NULL AND locked_until < ?
	`

	result, err := s.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to unlock expired jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(affected), nil
}

// ========================================
// RAG Query Logs
// ========================================

// CreateRAGQueryLog создает лог запроса
func (s *SQLiteDB) CreateRAGQueryLog(ctx context.Context, log *models.RAGQueryLog) error {
	sourceIDsJSON, _ := json.Marshal(log.SourceIDs)

	query := `
		INSERT INTO rag_query_logs (
			user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		log.UserID,
		log.ConversationID,
		log.QueryText,
		string(sourceIDsJSON),
		log.ChunksRetrieved,
		log.ChunksUsed,
		log.SearchTimeMs,
		log.TotalTokensUsed,
		log.ResponseQualityScore,
		log.CreatedAt,
	)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	log.ID = id
	return nil
}

// GetRAGQueryLog получает лог по ID
func (s *SQLiteDB) GetRAGQueryLog(ctx context.Context, id int64) (*models.RAGQueryLog, error) {
	query := `
		SELECT 
			id, user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		FROM rag_query_logs
		WHERE id = ?
	`

	var log models.RAGQueryLog
	var userID, conversationID sql.NullString
	var sourceIDsJSON string
	var chunksUsed, totalTokensUsed sql.NullInt64
	var responseQualityScore sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&userID,
		&conversationID,
		&log.QueryText,
		&sourceIDsJSON,
		&log.ChunksRetrieved,
		&chunksUsed,
		&log.SearchTimeMs,
		&totalTokensUsed,
		&responseQualityScore,
		&log.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("query log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get query log: %w", err)
	}

	if userID.Valid {
		uid, _ := uuid.Parse(userID.String)
		log.UserID = &uid
	}
	if conversationID.Valid {
		cid, _ := uuid.Parse(conversationID.String)
		log.ConversationID = &cid
	}
	if chunksUsed.Valid {
		log.ChunksUsed = int(chunksUsed.Int64)
	}
	if totalTokensUsed.Valid {
		log.TotalTokensUsed = int(totalTokensUsed.Int64)
	}
	if responseQualityScore.Valid {
		score := float64(responseQualityScore.Float64)
		log.ResponseQualityScore = &score
	}

	json.Unmarshal([]byte(sourceIDsJSON), &log.SourceIDs)

	return &log, nil
}

// ListRAGQueryLogsByUser возвращает логи пользователя
func (s *SQLiteDB) ListRAGQueryLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*models.RAGQueryLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT 
			id, user_id, conversation_id, query_text, source_ids,
			chunks_retrieved, chunks_used, search_time_ms,
			total_tokens_used, response_quality_score, created_at
		FROM rag_query_logs
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.RAGQueryLog
	for rows.Next() {
		var log models.RAGQueryLog
		var userIDField, conversationID sql.NullString
		var sourceIDsJSON string
		var chunksUsed, totalTokensUsed sql.NullInt64
		var responseQualityScore sql.NullFloat64

		err := rows.Scan(
			&log.ID,
			&userIDField,
			&conversationID,
			&log.QueryText,
			&sourceIDsJSON,
			&log.ChunksRetrieved,
			&chunksUsed,
			&log.SearchTimeMs,
			&totalTokensUsed,
			&responseQualityScore,
			&log.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if userIDField.Valid {
			uid, _ := uuid.Parse(userIDField.String)
			log.UserID = &uid
		}
		if conversationID.Valid {
			cid, _ := uuid.Parse(conversationID.String)
			log.ConversationID = &cid
		}
		if chunksUsed.Valid {
			log.ChunksUsed = int(chunksUsed.Int64)
		}
		if totalTokensUsed.Valid {
			log.TotalTokensUsed = int(totalTokensUsed.Int64)
		}
		if responseQualityScore.Valid {
			score := float64(responseQualityScore.Float64)
			log.ResponseQualityScore = &score
		}

		json.Unmarshal([]byte(sourceIDsJSON), &log.SourceIDs)

		logs = append(logs, &log)
	}

	return logs, nil
}

// Helper functions
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

