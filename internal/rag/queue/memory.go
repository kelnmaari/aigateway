// Package queue provides job queue implementations for RAG system.
package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"ollama-openai-proxy/internal/models"
)

var (
	ErrQueueFull  = errors.New("queue is full")
	ErrQueueEmpty = errors.New("queue is empty")
)

// MemoryQueue простая in-memory job queue для v1.13.1
type MemoryQueue struct {
	jobs      chan *models.RAGJob
	mu        sync.RWMutex
	idCounter int64
	logger    *logrus.Logger
	closed    bool
}

// NewMemoryQueue создает новый in-memory queue
func NewMemoryQueue(bufferSize int, logger *logrus.Logger) *MemoryQueue {
	return &MemoryQueue{
		jobs:      make(chan *models.RAGJob, bufferSize),
		logger:    logger,
		idCounter: 1,
	}
}

// Enqueue добавляет задачу в очередь
func (q *MemoryQueue) Enqueue(ctx context.Context, job *models.RAGJob) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errors.New("queue is closed")
	}
	
	// Assign ID
	job.ID = q.idCounter
	q.idCounter++
	job.Status = models.JobStatusPending
	job.CreatedAt = time.Now()
	job.Attempts = 0
	
	q.mu.Unlock()
	
	select {
	case q.jobs <- job:
		q.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"job_type": job.JobType,
		}).Info("Job enqueued")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// Dequeue получает следующую задачу из очереди
func (q *MemoryQueue) Dequeue(ctx context.Context) (*models.RAGJob, error) {
	select {
	case job := <-q.jobs:
		now := time.Now()
		job.Status = models.JobStatusProcessing
		job.StartedAt = &now
		job.Attempts++
		
		q.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"job_type": job.JobType,
			"attempt":  job.Attempts,
		}).Info("Job dequeued")
		
		return job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// DequeueNonBlocking получает задачу без блокировки
func (q *MemoryQueue) DequeueNonBlocking() (*models.RAGJob, error) {
	select {
	case job := <-q.jobs:
		now := time.Now()
		job.Status = models.JobStatusProcessing
		job.StartedAt = &now
		job.Attempts++
		
		q.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"job_type": job.JobType,
		}).Info("Job dequeued (non-blocking)")
		
		return job, nil
	default:
		return nil, ErrQueueEmpty
	}
}

// MarkCompleted помечает задачу как завершенную
func (q *MemoryQueue) MarkCompleted(jobID int64, result models.JobResult) error {
	q.logger.WithFields(logrus.Fields{
		"job_id": jobID,
	}).Info("Job completed")
	
	// В memory queue мы просто логируем, задача уже удалена из очереди
	return nil
}

// MarkFailed помечает задачу как провалившуюся и возможно повторяет
func (q *MemoryQueue) MarkFailed(job *models.RAGJob, err error) error {
	job.Error = err.Error()
	
	if job.Attempts < job.MaxAttempts {
		// Retry: добавляем обратно в очередь
		q.logger.WithFields(logrus.Fields{
			"job_id":  job.ID,
			"attempt": job.Attempts,
			"error":   err.Error(),
		}).Warn("Job failed, retrying")
		
		// Reset status для retry
		job.Status = models.JobStatusPending
		
		select {
		case q.jobs <- job:
			return nil
		default:
			return ErrQueueFull
		}
	}
	
	// Max attempts reached
	job.Status = models.JobStatusFailed
	now := time.Now()
	job.CompletedAt = &now
	
	q.logger.WithFields(logrus.Fields{
		"job_id":  job.ID,
		"attempt": job.Attempts,
		"error":   err.Error(),
	}).Error("Job failed permanently")
	
	return nil
}

// Size возвращает текущее количество задач в очереди
func (q *MemoryQueue) Size() int {
	return len(q.jobs)
}

// Close закрывает очередь
func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if q.closed {
		return nil
	}
	
	q.closed = true
	close(q.jobs)
	
	q.logger.Info("Memory queue closed")
	return nil
}

// Worker обрабатывает задачи из очереди
type Worker struct {
	ID        int
	Queue     *MemoryQueue
	Handler   JobHandler
	Logger    *logrus.Logger
	StopChan  chan struct{}
	WaitGroup *sync.WaitGroup
}

// JobHandler интерфейс обработчика задач
type JobHandler interface {
	Handle(ctx context.Context, job *models.RAGJob) (models.JobResult, error)
}

// Start запускает worker
func (w *Worker) Start(ctx context.Context) {
	w.WaitGroup.Add(1)
	go func() {
		defer w.WaitGroup.Done()
		
		w.Logger.WithField("worker_id", w.ID).Info("Worker started")
		
		for {
			select {
			case <-w.StopChan:
				w.Logger.WithField("worker_id", w.ID).Info("Worker stopped")
				return
			case <-ctx.Done():
				w.Logger.WithField("worker_id", w.ID).Info("Worker context cancelled")
				return
			default:
				// Try to get a job
				job, err := w.Queue.DequeueNonBlocking()
				if err == ErrQueueEmpty {
					// No jobs, sleep a bit
					time.Sleep(1 * time.Second)
					continue
				}
				if err != nil {
					w.Logger.WithError(err).Error("Failed to dequeue job")
					continue
				}
				
				// Process job
				w.processJob(ctx, job)
			}
		}
	}()
}

// processJob обрабатывает задачу
func (w *Worker) processJob(ctx context.Context, job *models.RAGJob) {
	w.Logger.WithFields(logrus.Fields{
		"worker_id": w.ID,
		"job_id":    job.ID,
		"job_type":  job.JobType,
	}).Info("Processing job")
	
	result, err := w.Handler.Handle(ctx, job)
	
	if err != nil {
		if markErr := w.Queue.MarkFailed(job, err); markErr != nil {
			w.Logger.WithError(markErr).Error("Failed to mark job as failed")
		}
		return
	}
	
	if err := w.Queue.MarkCompleted(job.ID, result); err != nil {
		w.Logger.WithError(err).Error("Failed to mark job as completed")
	}
}

// Stop останавливает worker
func (w *Worker) Stop() {
	close(w.StopChan)
}

