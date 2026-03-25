// Package concurrent provides concurrent processing utilities
// Version: v3.0.8 - Worker pool и batch processing
package concurrent

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// WorkerPool manages concurrent task execution with bounded parallelism
type WorkerPool struct {
	workers   int
	queueSize int
	taskQueue chan Task
	wg        sync.WaitGroup
	logger    *logrus.Logger
	metrics   *PoolMetrics
	ctx       context.Context
	cancel    context.CancelFunc
}

// Task represents a unit of work
type Task struct {
	ID      string
	Execute func(ctx context.Context) error
	OnError func(error)
}

// PoolMetrics tracks pool performance with cache line padding
type PoolMetrics struct {
	TotalTasks     int64
	_pad1          [56]byte
	CompletedTasks int64
	_pad2          [56]byte
	FailedTasks    int64
	_pad3          [56]byte
	ActiveWorkers  int32
	_pad4          [60]byte
}

// PoolConfig configuration for worker pool
type PoolConfig struct {
	Workers   int // Number of concurrent workers
	QueueSize int // Task queue buffer size
	Logger    *logrus.Logger
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(config PoolConfig) *WorkerPool {
	if config.Workers <= 0 {
		config.Workers = 10 // Default: 10 workers
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100 // Default: 100 task buffer
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:   config.Workers,
		queueSize: config.QueueSize,
		taskQueue: make(chan Task, config.QueueSize),
		logger:    config.Logger,
		metrics:   &PoolMetrics{},
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start workers
	for i := 0; i < config.Workers; i++ {
		pool.wg.Add(1)
		go pool.worker(i)
	}

	pool.logger.WithFields(logrus.Fields{
		"workers":    config.Workers,
		"queue_size": config.QueueSize,
	}).Info("✅ Worker pool started")

	return pool
}

// Submit submits a task for execution (blocking if queue full)
func (p *WorkerPool) Submit(task Task) error {
	select {
	case p.taskQueue <- task:
		atomic.AddInt64(&p.metrics.TotalTasks, 1)
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// TrySubmit attempts to submit a task (non-blocking)
func (p *WorkerPool) TrySubmit(task Task) bool {
	select {
	case p.taskQueue <- task:
		atomic.AddInt64(&p.metrics.TotalTasks, 1)
		return true
	default:
		return false
	}
}

// worker processes tasks from the queue
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	atomic.AddInt32(&p.metrics.ActiveWorkers, 1)
	defer atomic.AddInt32(&p.metrics.ActiveWorkers, -1)

	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				return // Channel closed
			}

			// Execute task
			if err := task.Execute(p.ctx); err != nil {
				atomic.AddInt64(&p.metrics.FailedTasks, 1)
				if task.OnError != nil {
					task.OnError(err)
				} else {
					p.logger.WithFields(logrus.Fields{
						"task_id": task.ID,
						"error":   err.Error(),
					}).Error("❌ Task failed")
				}
			} else {
				atomic.AddInt64(&p.metrics.CompletedTasks, 1)
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// Shutdown gracefully shuts down the pool
func (p *WorkerPool) Shutdown(timeout time.Duration) error {
	// Close task queue
	close(p.taskQueue)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("✅ Worker pool shut down gracefully")
		return nil
	case <-time.After(timeout):
		p.cancel() // Force cancel
		<-done
		p.logger.Warn("⚠️ Worker pool shut down forcefully")
		return context.DeadlineExceeded
	}
}

// GetMetrics returns pool metrics snapshot
func (p *WorkerPool) GetMetrics() PoolMetrics {
	return PoolMetrics{
		TotalTasks:     atomic.LoadInt64(&p.metrics.TotalTasks),
		CompletedTasks: atomic.LoadInt64(&p.metrics.CompletedTasks),
		FailedTasks:    atomic.LoadInt64(&p.metrics.FailedTasks),
		ActiveWorkers:  atomic.LoadInt32(&p.metrics.ActiveWorkers),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Batch Processing
// ─────────────────────────────────────────────────────────────────────────────

// BatchProcessor processes items in parallel batches
type BatchProcessor[T any] struct {
	batchSize int
	pool      *WorkerPool
	logger    *logrus.Logger
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor[T any](batchSize int, pool *WorkerPool) *BatchProcessor[T] {
	if batchSize <= 0 {
		batchSize = 10
	}

	return &BatchProcessor[T]{
		batchSize: batchSize,
		pool:      pool,
		logger:    pool.logger,
	}
}

// Process processes items in batches using the worker pool
func (bp *BatchProcessor[T]) Process(ctx context.Context, items []T, fn func(T) error) error {
	if len(items) == 0 {
		return nil
	}

	// Split into batches
	batches := bp.splitBatches(items)

	// Track errors
	var mu sync.Mutex
	var errors []error

	// Submit batches
	for batchIdx, batch := range batches {
		batchCopy := batch // Capture loop variable

		task := Task{
			ID: "batch-" + string(rune(batchIdx)),
			Execute: func(ctx context.Context) error {
				for _, item := range batchCopy {
					if err := fn(item); err != nil {
						return err
					}
				}
				return nil
			},
			OnError: func(err error) {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			},
		}

		if err := bp.pool.Submit(task); err != nil {
			return err
		}
	}

	// Wait for completion (simplified - poll metrics)
	total := int64(len(batches))
	for {
		metrics := bp.pool.GetMetrics()
		if metrics.CompletedTasks+metrics.FailedTasks >= total {
			break
		}
		time.Sleep(10 * time.Millisecond)

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if len(errors) > 0 {
		return errors[0] // Return first error
	}

	return nil
}

// splitBatches splits items into batches
func (bp *BatchProcessor[T]) splitBatches(items []T) [][]T {
	var batches [][]T

	for i := 0; i < len(items); i += bp.batchSize {
		end := min(i+bp.batchSize, len(items))
		batches = append(batches, items[i:end])
	}

	return batches
}

// ─────────────────────────────────────────────────────────────────────────────
// Parallel Map
// ─────────────────────────────────────────────────────────────────────────────

// ParallelMap applies function to all items in parallel
func ParallelMap[T, R any](ctx context.Context, items []T, fn func(T) (R, error), workers int) ([]R, error) {
	if len(items) == 0 {
		return []R{}, nil
	}

	if workers <= 0 {
		workers = 10
	}

	results := make([]R, len(items))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	// Channel for work items
	work := make(chan int, len(items))
	for i := range items {
		work <- i
	}
	close(work)

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Go(func() {

			for idx := range work {
				select {
				case <-ctx.Done():
					return
				default:
				}

				result, err := fn(items[idx])
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					return
				}

				results[idx] = result
			}
		})
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}
