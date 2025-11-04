// Package worker provides RAG job processing workers.
package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"aigateway/internal/models"
	"aigateway/internal/rag/embeddings"
	"aigateway/internal/rag/vector"
	"aigateway/internal/storage"
)

// RAGWorker обрабатывает RAG jobs из очереди
type RAGWorker struct {
	db            storage.Database
	embedder      embeddings.Embedder  // Для генерации embeddings (Version 1.14.0+)
	vectorStore   vector.VectorStore   // Для сохранения vectors (Version 1.14.0+)
	logger        *logrus.Logger
	pollInterval  time.Duration
	maxRetries    int
	stopCh        chan struct{}
	doneCh        chan struct{}
}

// NewRAGWorker создает новый RAG Worker
func NewRAGWorker(db storage.Database, embedder embeddings.Embedder, vectorStore vector.VectorStore, logger *logrus.Logger) *RAGWorker {
	return &RAGWorker{
		db:           db,
		embedder:     embedder,
		vectorStore:  vectorStore,
		logger:       logger,
		pollInterval: 5 * time.Second,  // Проверяем каждые 5 секунд
		maxRetries:   3,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}
}

// Start запускает worker
func (w *RAGWorker) Start(ctx context.Context) {
	w.logger.Info("RAG Worker started")
	
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("RAG Worker stopped by context")
			close(w.doneCh)
			return
		case <-w.stopCh:
			w.logger.Info("RAG Worker stopped")
			close(w.doneCh)
			return
		case <-ticker.C:
			if err := w.processNextJob(ctx); err != nil {
				w.logger.WithError(err).Error("Failed to process job")
			}
		}
	}
}

// Stop останавливает worker
func (w *RAGWorker) Stop() {
	w.logger.Info("Stopping RAG Worker...")
	close(w.stopCh)
	<-w.doneCh
	w.logger.Info("RAG Worker stopped successfully")
}

// processNextJob обрабатывает следующую job из очереди
func (w *RAGWorker) processNextJob(ctx context.Context) error {
	// Получаем следующую pending job
	job, err := w.db.GetNextPendingRAGJob(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next job: %w", err)
	}
	
	// Нет pending jobs
	if job == nil {
		return nil
	}
	
	w.logger.WithFields(logrus.Fields{
		"job_id":    job.ID,
		"job_type":  job.JobType,
		"attempts":  job.Attempts,
		"source_id": job.Payload["source_id"],
	}).Info("Processing RAG job")
	
	// Обновляем статус на processing
	job.Status = models.JobStatusProcessing
	job.Attempts++
	startedAt := time.Now()
	job.StartedAt = &startedAt
	
	// Блокируем job на 5 минут
	lockUntil := startedAt.Add(5 * time.Minute)
	job.LockedUntil = &lockUntil
	
	if err := w.db.UpdateRAGJob(ctx, job); err != nil {
		w.logger.WithError(err).Error("Failed to update job status to processing")
		return err
	}
	
	// Обрабатываем job
	err = w.executeJob(ctx, job)
	
	// Обновляем результат
	completedAt := time.Now()
	job.CompletedAt = &completedAt
	
	if err != nil {
		job.Status = models.JobStatusFailed
		job.Error = err.Error()
		
		w.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"attempts": job.Attempts,
			"error":    err.Error(),
		}).Error("Job failed")
		
		// Если превышен лимит попыток
		if job.Attempts >= w.maxRetries {
			w.logger.WithField("job_id", job.ID).Warn("Job max retries exceeded")
		}
	} else {
		job.Status = models.JobStatusCompleted
		
		w.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"duration": completedAt.Sub(startedAt).Seconds(),
		}).Info("Job completed successfully")
	}
	
	// Сохраняем результат
	if err := w.db.UpdateRAGJob(ctx, job); err != nil {
		w.logger.WithError(err).Error("Failed to update job result")
		return err
	}
	
	return nil
}

// executeJob выполняет job в зависимости от типа
func (w *RAGWorker) executeJob(ctx context.Context, job *models.RAGJob) error {
	sourceID, ok := job.Payload["source_id"].(string)
	if !ok {
		return fmt.Errorf("invalid source_id in payload")
	}
	
	w.logger.WithFields(logrus.Fields{
		"job_id":    job.ID,
		"job_type":  job.JobType,
		"source_id": sourceID,
	}).Debug("Executing job")
	
	switch job.JobType {
	case models.JobTypeAPISync:
		return w.executeAPISync(ctx, sourceID)
	case models.JobTypeDBQuery:
		return w.executeDBQuery(ctx, sourceID)
	case models.JobTypeWebScrape:
		return w.executeWebScrape(ctx, sourceID)
	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}

// executeAPISync реализован в api_sync.go

// executeDBQuery реализован в db_sync.go

// executeWebScrape реализован в web_scrape.go

