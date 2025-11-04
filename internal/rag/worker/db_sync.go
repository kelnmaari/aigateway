// Package worker provides RAG job processing for database sync.
package worker

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// executeDBQuery выполняет синхронизацию из базы данных
func (w *RAGWorker) executeDBQuery(ctx context.Context, sourceID string) error {
	w.logger.WithField("source_id", sourceID).Info("Executing DB query")
	
	// Получаем source из БД
	source, err := w.db.GetRAGDataSource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get data source: %w", err)
	}
	
	// Удаляем старые данные перед синхронизацией
	w.logger.WithField("source_id", sourceID).Info("Deleting old chunks before sync")
	if err := w.db.DeleteChunksBySource(ctx, sourceID); err != nil {
		w.logger.WithError(err).Warn("Failed to delete old chunks, continuing anyway")
	}
	
	// Обновляем статус source на syncing
	source.Status = models.SourceStatusSyncing
	if err := w.db.UpdateRAGDataSource(ctx, source); err != nil {
		w.logger.WithError(err).Warn("Failed to update source status to syncing")
	}
	
	// Парсим конфигурацию
	connectionString, ok := source.Config["connection_string"].(string)
	if !ok {
		return w.handleSyncError(ctx, source, fmt.Errorf("connection_string not found in config"))
	}
	
	query, ok := source.Config["query"].(string)
	if !ok {
		return w.handleSyncError(ctx, source, fmt.Errorf("query not found in config"))
	}
	
	// Определяем тип БД из connection string
	dbType := "postgres" // default
	if dbTypeConfig, ok := source.Config["database_type"].(string); ok {
		dbType = dbTypeConfig
	}
	
	w.logger.WithField("database_type", dbType).Debug("Connecting to database")
	
	// Подключаемся к БД
	db, err := sql.Open(dbType, connectionString)
	if err != nil {
		return w.handleSyncError(ctx, source, fmt.Errorf("failed to connect to database: %w", err))
	}
	defer db.Close()
	
	// Проверяем подключение
	if err := db.PingContext(ctx); err != nil {
		return w.handleSyncError(ctx, source, fmt.Errorf("failed to ping database: %w", err))
	}
	
	w.logger.Info("Database connection established, executing query")
	
	// Выполняем запрос
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return w.handleSyncError(ctx, source, fmt.Errorf("failed to execute query: %w", err))
	}
	defer rows.Close()
	
	// Получаем имена колонок
	columns, err := rows.Columns()
	if err != nil {
		return w.handleSyncError(ctx, source, fmt.Errorf("failed to get columns: %w", err))
	}
	
	w.logger.WithField("columns_count", len(columns)).Debug("Query executed successfully")
	
	// Обрабатываем результаты
	totalChunks := 0
	totalTokens := int64(0)
	allChunks := []*models.RAGChunk{} // Собираем все chunks для batch embeddings (Version 1.14.0+)
	
	for rows.Next() {
		// Создаем слайс для сканирования
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			w.logger.WithError(err).Warn("Failed to scan row, skipping")
			continue
		}
		
		// Преобразуем в map
		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Преобразуем []byte в string для удобства
			if b, ok := val.([]byte); ok {
				rowData[col] = string(b)
			} else {
				rowData[col] = val
			}
		}
		
		// Создаем документ и chunk
		chunk, tokens, err := w.createDocumentFromRow(ctx, source, rowData, columns)
		if err != nil {
			w.logger.WithError(err).Warn("Failed to create document from row")
			continue
		}
		
		if chunk != nil {
			allChunks = append(allChunks, chunk)
		}
		totalChunks++
		totalTokens += tokens
	}
	
	if err := rows.Err(); err != nil {
		return w.handleSyncError(ctx, source, fmt.Errorf("error iterating rows: %w", err))
	}
	
	// Генерируем embeddings для всех chunks батчем (Version 1.14.0+)
	if len(allChunks) > 0 {
		if err := w.generateAndStoreEmbeddings(ctx, allChunks); err != nil {
			w.logger.WithError(err).Warn("Failed to generate embeddings for chunks")
		}
		
		// Сохраняем все chunks в БД
		for _, chunk := range allChunks {
			if err := w.db.CreateRAGChunk(ctx, chunk); err != nil {
				w.logger.WithError(err).WithField("chunk_id", chunk.ID).Error("Failed to save chunk")
				continue
			}
		}
	}
	
	w.logger.WithFields(map[string]interface{}{
		"source_id":    sourceID,
		"total_chunks": totalChunks,
		"total_tokens": totalTokens,
	}).Info("DB sync completed successfully")
	
	// Обновляем статистику source
	now := time.Now()
	syncStatus := models.SyncStatusSuccess
	source.Status = models.SourceStatusActive
	source.LastSyncAt = &now
	source.LastSyncStatus = &syncStatus
	source.TotalChunks = totalChunks
	source.TotalTokens = totalTokens
	source.LastChunkCount = totalChunks
	source.LastError = ""
	
	if err := w.db.UpdateRAGDataSource(ctx, source); err != nil {
		w.logger.WithError(err).Warn("Failed to update source stats after successful sync")
	}
	
	return nil
}

// createDocumentFromRow создает документ и chunk из строки БД (Version 1.14.0+: возвращает chunk для batch embeddings)
func (w *RAGWorker) createDocumentFromRow(
	ctx context.Context,
	source *models.RAGDataSource,
	rowData map[string]interface{},
	columns []string,
) (*models.RAGChunk, int64, error) {
	// Формируем content из данных строки
	var contentParts []string
	for _, col := range columns {
		if val, ok := rowData[col]; ok && val != nil {
			contentParts = append(contentParts, fmt.Sprintf("%s: %v", col, val))
		}
	}
	content := strings.Join(contentParts, "\n")
	
	// Создаем metadata
	metadata := models.DocumentMetadata(rowData)
	
	// Создаем документ
	now := time.Now()
	document := &models.RAGDocument{
		ID:                    w.generateDocumentID(),
		SourceID:              source.ID,
		Filename:              fmt.Sprintf("row_%d.txt", now.Unix()),
		MimeType:              "text/plain",
		SizeBytes:             int64(len(content)),
		StorageBackend:        "memory",
		StoragePath:           "",
		Status:                models.DocumentStatusCompleted,
		ProcessingStartedAt:   &now,
		ProcessingCompletedAt: &now,
		Metadata:              metadata,
		TotalChunks:           1, // Пока 1 row = 1 chunk
		TotalTokens:           int64(len(content) / 4),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	
	if err := w.db.CreateRAGDocument(ctx, document); err != nil {
		return nil, 0, fmt.Errorf("failed to create document: %w", err)
	}
	
	// Создаем chunk (упрощенная логика: 1 row = 1 chunk)
	// НЕ сохраняем в БД здесь - будет сохранено батчем после embeddings (Version 1.14.0+)
	estimatedTokens := int64(len(content) / 4) // Грубая оценка: 1 token ≈ 4 символа
	
	chunkMetadata := models.ChunkMetadata(rowData)
	
	chunk := &models.RAGChunk{
		ID:          w.generateChunkID(),
		DocumentID:  document.ID,
		SourceID:    source.ID,
		ChunkText:   content,
		ChunkIndex:  0,
		ChunkTokens: int(estimatedTokens),
		Metadata:    chunkMetadata,
		CreatedAt:   now,
	}
	
	w.logger.WithFields(map[string]interface{}{
		"document_id": document.ID,
		"chunk_id":    chunk.ID,
		"tokens":      estimatedTokens,
	}).Debug("Document and chunk created")
	
	return chunk, estimatedTokens, nil
}

// handleSyncError обрабатывает ошибку синхронизации
func (w *RAGWorker) handleSyncError(ctx context.Context, source *models.RAGDataSource, err error) error {
	w.logger.WithError(err).Error("DB sync failed")
	
	now := time.Now()
	syncStatus := models.SyncStatusFailed
	source.Status = models.SourceStatusError
	source.LastSyncAt = &now
	source.LastSyncStatus = &syncStatus
	source.LastError = err.Error()
	
	if updateErr := w.db.UpdateRAGDataSource(ctx, source); updateErr != nil {
		w.logger.WithError(updateErr).Warn("Failed to update source error status")
	}
	
	return err
}

// generateDocumentID, generateChunkID, randInt находятся в api_sync.go

// timePtr replaced with utils.Ptr[time.Time]

