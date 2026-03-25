// Package worker provides RAG job processing workers.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/models"
	"aigateway/internal/rag/embeddings"
	"aigateway/internal/rag/vector"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// executeAPISync выполняет синхронизацию данных из REST API
func (w *RAGWorker) executeAPISync(ctx context.Context, sourceID string) error {
	w.logger.WithField("source_id", sourceID).Info("Starting API sync")

	// 1. Получить source configuration
	source, err := w.db.GetRAGDataSource(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get RAG data source: %w", err)
	}

	// 2. Удалить старые данные перед синхронизацией
	w.logger.WithField("source_id", sourceID).Info("Deleting old chunks before sync")
	if err := w.db.DeleteChunksBySource(ctx, sourceID); err != nil {
		w.logger.WithError(err).Warn("Failed to delete old chunks, continuing anyway")
	}

	// 3. Извлечь API config
	apiURL, ok := source.Config["url"].(string)
	if !ok || apiURL == "" {
		apiURL, ok = source.Config["endpoint"].(string)
		if !ok || apiURL == "" {
			return fmt.Errorf("url or endpoint not found in config")
		}
	}

	method, ok := source.Config["method"].(string)
	if !ok || method == "" {
		method = "GET" // Default to GET
	}
	method = strings.ToUpper(method)

	w.logger.WithFields(logrus.Fields{
		"source_id": sourceID,
		"url":       apiURL,
		"method":    method,
	}).Info("Executing API request")

	// 3. Создать HTTP запрос
	req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// 4. Добавить headers из config
	if headers, ok := source.Config["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	// 5. Выполнить запрос
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 6. Читать response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":   sourceID,
		"body_length": len(body),
		"status_code": resp.StatusCode,
	}).Debug("API response received")

	// 7. Парсить JSON response
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		// Если не JSON, сохраняем как plain text
		w.logger.WithField("source_id", sourceID).Warn("Response is not JSON, treating as plain text")
		return w.processTextResponse(ctx, source, string(body))
	}

	// 8. Обработать JSON response
	return w.processJSONResponse(ctx, source, data)
}

// processTextResponse обрабатывает plain text ответ
func (w *RAGWorker) processTextResponse(ctx context.Context, source *models.RAGDataSource, text string) error {
	// Создаем один документ из текста
	now := time.Now()
	document := &models.RAGDocument{
		ID:                    w.generateDocumentID(),
		SourceID:              source.ID,
		Filename:              fmt.Sprintf("api_response_%d.txt", now.Unix()),
		MimeType:              "text/plain",
		SizeBytes:             int64(len(text)),
		StorageBackend:        "memory",
		StoragePath:           "",
		Status:                models.DocumentStatusCompleted,
		ProcessingStartedAt:   &now,
		ProcessingCompletedAt: &now,
		Metadata:              make(models.DocumentMetadata),
		TotalChunks:           0,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := w.db.CreateRAGDocument(ctx, document); err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	// Создаем chunks из текста
	chunks, totalTokens := w.createChunksFromText(source, document.ID, text)

	// Генерируем embeddings если embedder доступен (Version 1.14.0+)
	if err := w.generateAndStoreEmbeddings(ctx, chunks); err != nil {
		w.logger.WithError(err).Warn("Failed to generate embeddings, continuing without them")
	}

	// Сохраняем chunks в БД
	for _, chunk := range chunks {
		if err := w.db.CreateRAGChunk(ctx, chunk); err != nil {
			w.logger.WithError(err).Error("Failed to create chunk")
			continue
		}
	}

	// Обновляем document statistics
	document.TotalChunks = len(chunks)
	if err := w.db.UpdateRAGDocument(ctx, document); err != nil {
		w.logger.WithError(err).Warn("Failed to update document stats")
	}

	// Обновляем source statistics
	source.TotalChunks += len(chunks)
	source.TotalTokens += totalTokens
	source.LastChunkCount = len(chunks)
	source.LastSyncAt = &now
	success := models.SyncStatusSuccess
	source.LastSyncStatus = &success
	source.LastError = ""
	source.Status = models.SourceStatusActive

	if err := w.db.UpdateRAGDataSource(ctx, source); err != nil {
		return fmt.Errorf("failed to update source stats: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":    source.ID,
		"total_chunks": len(chunks),
		"total_tokens": totalTokens,
	}).Info("API sync completed successfully (text)")

	return nil
}

// processJSONResponse обрабатывает JSON ответ
func (w *RAGWorker) processJSONResponse(ctx context.Context, source *models.RAGDataSource, data any) error {
	now := time.Now()

	// Определяем data_path для извлечения массива данных
	dataPath, ok := source.Config["data_path"].(string)
	if ok && dataPath != "" {
		// Извлекаем данные по пути (например "data.items" или "results")
		data = extractByPath(data, dataPath)
	}

	// Проверяем является ли результат массивом
	items, ok := data.([]any)
	if !ok {
		// Если не массив, оборачиваем в массив
		items = []any{data}
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":   source.ID,
		"items_count": len(items),
	}).Info("Processing JSON items")

	var (
		totalChunks int
		totalTokens int64
	)

	// Обрабатываем каждый item
	for idx, item := range items {
		// Создаем документ для item
		document := &models.RAGDocument{
			ID:                    w.generateDocumentID(),
			SourceID:              source.ID,
			Filename:              fmt.Sprintf("api_item_%d_%d.json", now.Unix(), idx),
			MimeType:              "application/json",
			SizeBytes:             0, // will calculate
			StorageBackend:        "memory",
			StoragePath:           "",
			Status:                models.DocumentStatusCompleted,
			ProcessingStartedAt:   &now,
			ProcessingCompletedAt: &now,
			Metadata:              make(models.DocumentMetadata),
			TotalChunks:           0,
			CreatedAt:             now,
			UpdatedAt:             now,
		}

		// Конвертируем item в текст для chunking
		itemText := convertItemToText(item, source.Config)
		document.SizeBytes = int64(len(itemText))

		if err := w.db.CreateRAGDocument(ctx, document); err != nil {
			w.logger.WithError(err).WithField("item_idx", idx).Error("Failed to create document")
			continue
		}

		// Создаем chunks
		chunks, tokensAdded := w.createChunksFromText(source, document.ID, itemText)

		// Генерируем embeddings если embedder доступен (Version 1.14.0+)
		if err := w.generateAndStoreEmbeddings(ctx, chunks); err != nil {
			w.logger.WithError(err).Warn("Failed to generate embeddings for item, continuing")
		}

		// Сохраняем chunks в БД
		for _, chunk := range chunks {
			if err := w.db.CreateRAGChunk(ctx, chunk); err != nil {
				w.logger.WithError(err).Error("Failed to create chunk")
				continue
			}
		}

		// Обновляем document statistics
		document.TotalChunks = len(chunks)
		if err := w.db.UpdateRAGDocument(ctx, document); err != nil {
			w.logger.WithError(err).Warn("Failed to update document stats")
		}

		totalChunks += len(chunks)
		totalTokens += tokensAdded
	}

	// Обновляем source statistics
	source.TotalChunks += totalChunks
	source.TotalTokens += totalTokens
	source.LastChunkCount = totalChunks
	source.LastSyncAt = &now
	success := models.SyncStatusSuccess
	source.LastSyncStatus = &success
	source.LastError = ""
	source.Status = models.SourceStatusActive

	if err := w.db.UpdateRAGDataSource(ctx, source); err != nil {
		return fmt.Errorf("failed to update source stats: %w", err)
	}

	w.logger.WithFields(logrus.Fields{
		"source_id":    source.ID,
		"items_count":  len(items),
		"total_chunks": totalChunks,
		"total_tokens": totalTokens,
	}).Info("API sync completed successfully (JSON)")

	return nil
}

// createChunksFromText создает chunks из текста
func (w *RAGWorker) createChunksFromText(source *models.RAGDataSource, documentID string, text string) ([]*models.RAGChunk, int64) {
	// Получаем chunk size из indexing config
	maxChunkSize := 500 // Default
	chunkOverlap := 50  // Default

	if source.IndexingConfig.MaxChunkSize > 0 {
		maxChunkSize = source.IndexingConfig.MaxChunkSize
	}
	if source.IndexingConfig.ChunkOverlap > 0 {
		chunkOverlap = source.IndexingConfig.ChunkOverlap
	}

	// Простая chunking strategy: по предложениям
	chunks := []*models.RAGChunk{}
	var totalTokens int64
	chunkIndex := 0

	// Split by lines first
	lines := strings.Split(text, "\n")
	currentChunk := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// If adding this line exceeds chunk size, create new chunk
		if len(currentChunk)+len(line) > maxChunkSize && currentChunk != "" {
			chunk := w.createChunk(source.ID, documentID, currentChunk, chunkIndex)
			chunks = append(chunks, chunk)
			totalTokens += int64(chunk.ChunkTokens)
			chunkIndex++

			// Start new chunk with overlap
			overlapText := currentChunk
			if len(overlapText) > chunkOverlap {
				overlapText = overlapText[len(overlapText)-chunkOverlap:]
			}
			currentChunk = overlapText + "\n" + line
		} else {
			if currentChunk != "" {
				currentChunk += "\n"
			}
			currentChunk += line
		}
	}

	// Add last chunk
	if currentChunk != "" {
		chunk := w.createChunk(source.ID, documentID, currentChunk, chunkIndex)
		chunks = append(chunks, chunk)
		totalTokens += int64(chunk.ChunkTokens)
	}

	return chunks, totalTokens
}

// createChunk создает RAGChunk
func (w *RAGWorker) createChunk(sourceID, documentID, text string, index int) *models.RAGChunk {
	// Простой подсчет токенов (примерно 4 символа = 1 токен)
	tokens := max(len(text)/4, 1)

	return &models.RAGChunk{
		ID:          w.generateChunkID(),
		DocumentID:  documentID,
		SourceID:    sourceID,
		ChunkText:   text,
		ChunkIndex:  index,
		ChunkTokens: tokens,
		Metadata:    make(models.ChunkMetadata),
		CreatedAt:   time.Now(),
	}
}

// convertItemToText конвертирует JSON item в текст
func convertItemToText(item any, config models.SourceConfig) string {
	// Если указаны text_fields, используем только их
	if textFieldsRaw, ok := config["text_fields"]; ok {
		if textFields, ok := textFieldsRaw.([]any); ok {
			var parts []string
			if itemMap, ok := item.(map[string]any); ok {
				for _, fieldRaw := range textFields {
					if field, ok := fieldRaw.(string); ok {
						if value, exists := itemMap[field]; exists {
							parts = append(parts, fmt.Sprintf("%s: %v", field, value))
						}
					}
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, "\n")
			}
		}
	}

	// Иначе конвертируем весь item в JSON string
	data, _ := json.MarshalIndent(item, "", "  ")
	return string(data)
}

// extractByPath извлекает данные по пути (например "data.items")
func extractByPath(data any, path string) any {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if m, ok := current.(map[string]any); ok {
			current = m[part]
		} else {
			return data // Path not found, return original
		}
	}

	return current
}

// generateAndStoreEmbeddings генерирует embeddings для chunks и сохраняет в vector store (Version 1.14.0+)
func (w *RAGWorker) generateAndStoreEmbeddings(ctx context.Context, chunks []*models.RAGChunk) error {
	// Если embedder не настроен, пропускаем
	if w.embedder == nil || w.vectorStore == nil {
		return nil
	}

	if len(chunks) == 0 {
		return nil
	}

	w.logger.WithField("chunks_count", len(chunks)).Debug("Generating embeddings for chunks")

	// Собираем тексты для batch generation
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.ChunkText
	}

	// Генерируем embeddings батчем
	batchResp, err := w.embedder.EmbedBatch(ctx, embeddings.BatchEmbeddingRequest{
		Texts: texts,
		Model: "", // Используем default model из embedder
	})
	if err != nil {
		return fmt.Errorf("failed to generate batch embeddings: %w", err)
	}

	if len(batchResp.Embeddings) != len(chunks) {
		return fmt.Errorf("embeddings count mismatch: got %d, expected %d", len(batchResp.Embeddings), len(chunks))
	}

	// Сохраняем vectors в vector store
	for i, chunk := range chunks {
		embedding := batchResp.Embeddings[i]

		// Создаем vector document
		vectorDoc := vector.VectorDocument{
			ID:     chunk.ID,
			Text:   chunk.ChunkText,
			Vector: embedding.Vector,
			Metadata: map[string]any{
				"chunk_id":    chunk.ID,
				"document_id": chunk.DocumentID,
				"source_id":   chunk.SourceID,
				"chunk_index": chunk.ChunkIndex,
				"tokens":      chunk.ChunkTokens,
			},
			CreatedAt: chunk.CreatedAt,
		}

		// Сохраняем в vector store
		if err := w.vectorStore.Insert(ctx, vectorDoc); err != nil {
			w.logger.WithError(err).WithField("chunk_id", chunk.ID).Error("Failed to insert vector")
			// Продолжаем обработку остальных
			continue
		}
	}

	w.logger.WithFields(map[string]any{
		"chunks_count":     len(chunks),
		"embeddings_model": batchResp.Model,
		"total_tokens":     batchResp.TotalTokens,
	}).Info("Embeddings generated and stored successfully")

	return nil
}

// generateDocumentID генерирует уникальный ID для документа
func (w *RAGWorker) generateDocumentID() string {
	return uuid.New().String()
}

// generateChunkID генерирует уникальный ID для chunk
func (w *RAGWorker) generateChunkID() string {
	return uuid.New().String()
}
