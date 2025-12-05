// Package worker provides worker pool for GitLab MR analysis
package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
)

// Pool manages a pool of workers for processing analysis jobs
type Pool struct {
	store   storage.Store
	logger  *logrus.Logger
	config  PoolConfig

	// Worker state
	workers   []*Worker
	workersMu sync.Mutex

	// Job processing
	processor JobProcessor

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Stats (atomic for lock-free reads)
	activeJobs   int64
	completedJobs int64
	failedJobs    int64
}

// PoolConfig holds configuration for the worker pool
type PoolConfig struct {
	WorkerCount   int           // Number of workers (default: 3)
	PollInterval  time.Duration // How often to poll for jobs (default: 5s)
	JobTimeout    time.Duration // Max time for a job (default: 30m)
	MaxRetries    int           // Max retries per job (default: 3)
	RetryBackoff  time.Duration // Initial retry backoff (default: 1m)
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		WorkerCount:  3,
		PollInterval: 5 * time.Second,
		JobTimeout:   30 * time.Minute,
		MaxRetries:   3,
		RetryBackoff: 1 * time.Minute,
	}
}

// JobProcessor processes analysis jobs
type JobProcessor interface {
	ProcessJob(ctx context.Context, job *models.GitLabAnalysisJob) error
}

// NewPool creates a new worker pool
func NewPool(store storage.Store, processor JobProcessor, logger *logrus.Logger, config PoolConfig) *Pool {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 3
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.JobTimeout <= 0 {
		config.JobTimeout = 30 * time.Minute
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 1 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		store:     store,
		logger:    logger,
		config:    config,
		processor: processor,
		ctx:       ctx,
		cancel:    cancel,
		workers:   make([]*Worker, 0, config.WorkerCount),
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	p.logger.WithField("worker_count", p.config.WorkerCount).Info("Starting worker pool")

	p.workersMu.Lock()
	for i := 0; i < p.config.WorkerCount; i++ {
		worker := NewWorker(p, i)
		p.workers = append(p.workers, worker)
		p.wg.Add(1)
		go worker.Run()
	}
	p.workersMu.Unlock()
}

// Stop gracefully stops the worker pool
func (p *Pool) Stop(drainTimeout time.Duration) {
	p.logger.Info("Stopping worker pool...")
	
	// Signal all workers to stop
	p.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Worker pool stopped gracefully")
	case <-time.After(drainTimeout):
		p.logger.Warn("Worker pool stop timed out, some jobs may be incomplete")
	}
}

// EnqueueJob creates a job in the queue
func (p *Pool) EnqueueJob(ctx context.Context, job *models.GitLabAnalysisJob) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	job.Status = models.GitLabJobStatusPending
	job.MaxRetries = p.config.MaxRetries

	if err := p.store.CreateJob(ctx, job); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"mr_iid":   job.MRIID,
		"priority": job.Priority,
	}).Info("Job enqueued")

	return nil
}

// Stats returns current pool statistics
func (p *Pool) Stats() PoolStats {
	p.workersMu.Lock()
	activeWorkers := 0
	idleWorkers := 0
	for _, w := range p.workers {
		if w.IsActive() {
			activeWorkers++
		} else {
			idleWorkers++
		}
	}
	p.workersMu.Unlock()

	return PoolStats{
		TotalWorkers:   p.config.WorkerCount,
		ActiveWorkers:  activeWorkers,
		IdleWorkers:    idleWorkers,
		ActiveJobs:     atomic.LoadInt64(&p.activeJobs),
		CompletedJobs:  atomic.LoadInt64(&p.completedJobs),
		FailedJobs:     atomic.LoadInt64(&p.failedJobs),
	}
}

// PoolStats contains pool statistics
type PoolStats struct {
	TotalWorkers  int
	ActiveWorkers int
	IdleWorkers   int
	ActiveJobs    int64
	CompletedJobs int64
	FailedJobs    int64
}

// ============================================================================
// Worker
// ============================================================================

// Worker processes jobs from the queue
type Worker struct {
	pool     *Pool
	id       int
	workerID string
	
	active   int32 // atomic
	currentJob *models.GitLabAnalysisJob
	mu       sync.Mutex
}

// NewWorker creates a new worker
func NewWorker(pool *Pool, id int) *Worker {
	return &Worker{
		pool:     pool,
		id:       id,
		workerID: fmt.Sprintf("worker-%d-%s", id, uuid.New().String()[:8]),
	}
}

// Run starts the worker loop
func (w *Worker) Run() {
	defer w.pool.wg.Done()

	w.pool.logger.WithFields(logrus.Fields{
		"worker_id": w.workerID,
		"worker_num": w.id,
	}).Info("Worker started")

	ticker := time.NewTicker(w.pool.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.pool.ctx.Done():
			w.pool.logger.WithField("worker_id", w.workerID).Info("Worker stopping")
			return
		case <-ticker.C:
			w.processNextJob()
		}
	}
}

// processNextJob attempts to get and process the next job
func (w *Worker) processNextJob() {
	ctx := w.pool.ctx

	// Get next pending job
	job, err := w.pool.store.GetNextPendingJob(ctx)
	if err != nil {
		w.pool.logger.WithError(err).Error("Failed to get next job")
		return
	}
	if job == nil {
		return // No jobs available
	}

	// Try to claim the job
	if err := w.pool.store.ClaimJob(ctx, job.ID, w.workerID); err != nil {
		// Job was claimed by another worker
		return
	}

	// Mark as active
	w.setActive(true, job)
	atomic.AddInt64(&w.pool.activeJobs, 1)
	defer func() {
		atomic.AddInt64(&w.pool.activeJobs, -1)
		w.setActive(false, nil)
	}()

	w.pool.logger.WithFields(logrus.Fields{
		"worker_id": w.workerID,
		"job_id":    job.ID,
		"mr_iid":    job.MRIID,
	}).Info("Processing job")

	// Process with timeout and panic recovery
	jobCtx, cancel := context.WithTimeout(ctx, w.pool.config.JobTimeout)
	defer cancel()

	err = w.safeProcessJob(jobCtx, job)
	if err != nil {
		w.handleJobError(ctx, job, err)
		return
	}

	// Mark as completed
	if err := w.pool.store.CompleteJob(ctx, job.ID); err != nil {
		w.pool.logger.WithError(err).Error("Failed to mark job as completed")
	}
	atomic.AddInt64(&w.pool.completedJobs, 1)

	w.pool.logger.WithFields(logrus.Fields{
		"worker_id": w.workerID,
		"job_id":    job.ID,
	}).Info("Job completed")
}

// safeProcessJob processes a job with panic recovery
func (w *Worker) safeProcessJob(ctx context.Context, job *models.GitLabAnalysisJob) (err error) {
	defer func() {
		if r := recover(); r != nil {
			w.pool.logger.WithFields(logrus.Fields{
				"worker_id": w.workerID,
				"job_id":    job.ID,
				"panic":     r,
			}).Error("Panic recovered during job processing")
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return w.pool.processor.ProcessJob(ctx, job)
}

// handleJobError handles job failure and retry logic
func (w *Worker) handleJobError(ctx context.Context, job *models.GitLabAnalysisJob, err error) {
	w.pool.logger.WithError(err).WithFields(logrus.Fields{
		"worker_id":   w.workerID,
		"job_id":      job.ID,
		"retry_count": job.RetryCount,
	}).Error("Job failed")

	// Check if we should retry
	if job.RetryCount < job.MaxRetries {
		// Calculate backoff with exponential increase
		backoff := w.pool.config.RetryBackoff * time.Duration(1<<uint(job.RetryCount))
		nextRetry := time.Now().Add(backoff)

		if err := w.pool.store.RetryJob(ctx, job.ID, nextRetry); err != nil {
			w.pool.logger.WithError(err).Error("Failed to schedule retry")
			w.pool.store.FailJob(ctx, job.ID, err.Error())
		} else {
			w.pool.logger.WithFields(logrus.Fields{
				"job_id":     job.ID,
				"next_retry": nextRetry,
				"backoff":    backoff,
			}).Info("Job scheduled for retry")
		}
		return
	}

	// Max retries exceeded, mark as failed
	if err := w.pool.store.FailJob(ctx, job.ID, err.Error()); err != nil {
		w.pool.logger.WithError(err).Error("Failed to mark job as failed")
	}
	atomic.AddInt64(&w.pool.failedJobs, 1)
}

// setActive sets the worker's active status
func (w *Worker) setActive(active bool, job *models.GitLabAnalysisJob) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if active {
		atomic.StoreInt32(&w.active, 1)
		w.currentJob = job
	} else {
		atomic.StoreInt32(&w.active, 0)
		w.currentJob = nil
	}
}

// IsActive returns whether the worker is currently processing a job
func (w *Worker) IsActive() bool {
	return atomic.LoadInt32(&w.active) == 1
}

// CurrentJob returns the job currently being processed (if any)
func (w *Worker) CurrentJob() *models.GitLabAnalysisJob {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.currentJob
}

// WorkerID returns the worker's unique identifier
func (w *Worker) WorkerID() string {
	return w.workerID
}

