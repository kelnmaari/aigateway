// Package processor handles document processing pipeline.
package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/rag/chunker"
	"ollama-openai-proxy/internal/storage"
)

// ProcessingStatus статус обработки документа
type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusCompleted  ProcessingStatus = "completed"
	StatusFailed     ProcessingStatus = "failed"
)

// DocumentProcessor обрабатывает документы в chunks
type DocumentProcessor struct {
	db         storage.Storage
	chunker    chunker.Chunker
	logger     *logrus.Logger
	maxWorkers int
}

// NewDocumentProcessor создает новый processor
func NewDocumentProcessor(
	db storage.Storage,
	chunker chunker.Chunker,
	logger *logrus.Logger,
	maxWorkers int,
) *DocumentProcessor {
	if maxWorkers <= 0 {
		maxWorkers = 4 // Default
	}

	return &DocumentProcessor{
		db:         db,
		chunker:    chunker,
		logger:     logger,
		maxWorkers: maxWorkers,
	}
}

// ProcessDocument обрабатывает документ и создает chunks
func (p *DocumentProcessor) ProcessDocument(ctx context.Context, documentID, documentText string) error {
	p.logger.WithFields(logrus.Fields{
		"document_id": documentID,
		"text_length": len(documentText),
	}).Info("Starting document processing")

	// Получаем документ из БД
	document, err := p.db.GetRAGDocument(ctx, documentID)
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	// Обновляем статус на processing
	document.Status = "processing"
	now := time.Now()
	document.UpdatedAt = &now
	if err := p.db.UpdateRAGDocument(ctx, document); err != nil {
		p.logger.WithError(err).Error("Failed to update document status to processing")
	}

	// Chunking опции
	chunkOptions := chunker.ChunkingOptions{
		MaxTokens:     512,
		Overlap:       50,
		Strategy:      "semantic",
		PreserveLines: true,
		Metadata: map[string]interface{}{
			"document_id": documentID,
			"source_id":   document.SourceID,
			"filename":    document.Filename,
		},
	}

	// Разбиваем на chunks
	chunks, err := p.chunker.Chunk(ctx, documentText, chunkOptions)
	if err != nil {
		// Обновляем статус на error
		document.Status = "error"
		document.ErrorMessage = func() *string { s := err.Error(); return &s }()
		now := time.Now()
		document.UpdatedAt = &now
		if updateErr := p.db.UpdateRAGDocument(ctx, document); updateErr != nil {
			p.logger.WithError(updateErr).Error("Failed to update document status to error")
		}
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"document_id": documentID,
		"chunks":      len(chunks),
	}).Info("Document chunked successfully")

	// Сохраняем chunks в БД
	totalTokens := int64(0)
	for _, chunk := range chunks {
		ragChunk := &models.RAGChunk{
			ID:          uuid.New().String(),
			DocumentID:  documentID,
			SourceID:    document.SourceID,
			ChunkText:   chunk.Text,
			ChunkIndex:  chunk.Index,
			ChunkTokens: chunk.Tokens,
			Metadata:    chunk.Metadata,
			StartOffset: &chunk.StartOffset,
			EndOffset:   &chunk.EndOffset,
			CreatedAt:   time.Now(),
		}

		if err := p.db.CreateRAGChunk(ctx, ragChunk); err != nil {
			p.logger.WithError(err).WithField("chunk_index", chunk.Index).Error("Failed to save chunk")
			// Продолжаем обработку остальных chunks
			continue
		}

		totalTokens += int64(chunk.Tokens)
	}

	// Обновляем документ: статус, chunk count, tokens
	document.Status = "completed"
	document.ChunkCount = len(chunks)
	document.TotalTokens = int(totalTokens)
	now = time.Now()
	document.ProcessedAt = &now
	document.UpdatedAt = &now

	if err := p.db.UpdateRAGDocument(ctx, document); err != nil {
		p.logger.WithError(err).Error("Failed to update document after processing")
		return fmt.Errorf("failed to update document: %w", err)
	}

	// Обновляем статистику в source
	source, err := p.db.GetRAGDataSource(ctx, document.SourceID)
	if err == nil {
		source.TotalChunks += len(chunks)
		source.TotalTokens += totalTokens
		source.LastChunkCount = &len(chunks)
		now := time.Now()
		source.UpdatedAt = &now

		if err := p.db.UpdateRAGDataSource(ctx, source); err != nil {
			p.logger.WithError(err).Error("Failed to update source statistics")
		}
	}

	p.logger.WithFields(logrus.Fields{
		"document_id":   documentID,
		"chunks_count":  len(chunks),
		"total_tokens":  totalTokens,
	}).Info("Document processing completed successfully")

	return nil
}

// ProcessDocumentBatch обрабатывает batch документов
func (p *DocumentProcessor) ProcessDocumentBatch(ctx context.Context, jobs []ProcessingJob) error {
	p.logger.WithField("batch_size", len(jobs)).Info("Processing document batch")

	for _, job := range jobs {
		if err := p.ProcessDocument(ctx, job.DocumentID, job.DocumentText); err != nil {
			p.logger.WithError(err).WithField("document_id", job.DocumentID).Error("Failed to process document in batch")
			// Продолжаем обработку остальных
			continue
		}
	}

	return nil
}

// ProcessingJob представляет задачу обработки документа
type ProcessingJob struct {
	DocumentID   string
	DocumentText string
	Priority     int
}

// GetPendingDocuments возвращает документы ожидающие обработки
func (p *DocumentProcessor) GetPendingDocuments(ctx context.Context, sourceID string, limit int) ([]ProcessingJob, error) {
	// Получаем документы со статусом pending или failed
	documents, err := p.db.ListRAGDocuments(ctx, &storage.RAGDocumentFilter{
		SourceID: &sourceID,
		Status:   func() *string { s := "pending"; return &s }(),
		Limit:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pending documents: %w", err)
	}

	jobs := make([]ProcessingJob, 0, len(documents))
	for _, doc := range documents {
		// В production здесь нужно загружать текст из storage
		// Пока используем заглушку
		jobs = append(jobs, ProcessingJob{
			DocumentID:   doc.ID,
			DocumentText: "", // TODO: Load from storage
			Priority:     1,
		})
	}

	return jobs, nil
}

