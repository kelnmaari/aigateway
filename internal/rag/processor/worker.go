// Package processor provides document processing workers.
package processor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// WorkerPool manages a pool of workers for document processing
type WorkerPool struct {
	processor     *DocumentProcessor
	workers       int
	jobQueue      chan ProcessingJob
	logger        *logrus.Logger
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	
	// Metrics
	jobsProcessed atomic.Int64
	jobsFailed    atomic.Int64
	_pad1         [56]byte // Cache line padding
	
	isRunning     atomic.Bool
	_pad2         [63]byte // Cache line padding
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(processor *DocumentProcessor, workers int, queueSize int, logger *logrus.Logger) *WorkerPool {
	if workers <= 0 {
		workers = 4
	}
	if queueSize <= 0 {
		queueSize = 100
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		processor: processor,
		workers:   workers,
		jobQueue:  make(chan ProcessingJob, queueSize),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start запускает worker pool
func (wp *WorkerPool) Start() error {
	if wp.isRunning.Load() {
		return fmt.Errorf("worker pool already running")
	}
	
	wp.logger.WithField("workers", wp.workers).Info("Starting worker pool")
	wp.isRunning.Store(true)
	
	// Запускаем workers
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
	
	wp.logger.Info("Worker pool started successfully")
	return nil
}

// Stop останавливает worker pool
func (wp *WorkerPool) Stop() error {
	if !wp.isRunning.Load() {
		return fmt.Errorf("worker pool not running")
	}
	
	wp.logger.Info("Stopping worker pool...")
	
	// Закрываем канал с job'ами
	close(wp.jobQueue)
	
	// Отменяем context
	wp.cancel()
	
	// Ждем завершения всех workers
	wp.wg.Wait()
	
	wp.isRunning.Store(false)
	wp.logger.Info("Worker pool stopped")
	
	return nil
}

// Submit отправляет job на обработку
func (wp *WorkerPool) Submit(job ProcessingJob) error {
	if !wp.isRunning.Load() {
		return fmt.Errorf("worker pool not running")
	}
	
	select {
	case wp.jobQueue <- job:
		return nil
	case <-wp.ctx.Done():
		return fmt.Errorf("worker pool shutting down")
	default:
		return fmt.Errorf("job queue full")
	}
}

// worker обрабатывает jobs из очереди
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()
	
	wp.logger.WithField("worker_id", id).Info("Worker started")
	
	for {
		select {
		case job, ok := <-wp.jobQueue:
			if !ok {
				// Канал закрыт, выходим
				wp.logger.WithField("worker_id", id).Info("Worker shutting down")
				return
			}
			
			// Обрабатываем job
			wp.processJob(id, job)
			
		case <-wp.ctx.Done():
			wp.logger.WithField("worker_id", id).Info("Worker cancelled")
			return
		}
	}
}

// processJob обрабатывает один job
func (wp *WorkerPool) processJob(workerID int, job ProcessingJob) {
	startTime := time.Now()
	
	wp.logger.WithFields(logrus.Fields{
		"worker_id":   workerID,
		"document_id": job.DocumentID,
	}).Info("Processing job")
	
	// Создаем context с timeout
	ctx, cancel := context.WithTimeout(wp.ctx, 5*time.Minute)
	defer cancel()
	
	// Обрабатываем документ
	err := wp.processor.ProcessDocument(ctx, job.DocumentID, job.DocumentText)
	
	duration := time.Since(startTime)
	
	if err != nil {
		wp.jobsFailed.Add(1)
		wp.logger.WithFields(logrus.Fields{
			"worker_id":   workerID,
			"document_id": job.DocumentID,
			"duration_ms": duration.Milliseconds(),
			"error":       err.Error(),
		}).Error("Job processing failed")
	} else {
		wp.jobsProcessed.Add(1)
		wp.logger.WithFields(logrus.Fields{
			"worker_id":   workerID,
			"document_id": job.DocumentID,
			"duration_ms": duration.Milliseconds(),
		}).Info("Job processing completed")
	}
}

// Stats возвращает статистику worker pool
func (wp *WorkerPool) Stats() WorkerPoolStats {
	return WorkerPoolStats{
		Workers:       wp.workers,
		QueueSize:     len(wp.jobQueue),
		QueueCapacity: cap(wp.jobQueue),
		JobsProcessed: wp.jobsProcessed.Load(),
		JobsFailed:    wp.jobsFailed.Load(),
		IsRunning:     wp.isRunning.Load(),
	}
}

// WorkerPoolStats статистика worker pool
type WorkerPoolStats struct {
	Workers       int
	QueueSize     int
	QueueCapacity int
	JobsProcessed int64
	JobsFailed    int64
	IsRunning     bool
}

// WaitUntilEmpty ждет пока очередь не опустеет
func (wp *WorkerPool) WaitUntilEmpty(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	
	for {
		if len(wp.jobQueue) == 0 {
			return nil
		}
		
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for queue to empty")
		}
		
		time.Sleep(100 * time.Millisecond)
	}
}

