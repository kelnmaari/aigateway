// Package jobs provides background job management for user-initiated tasks
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
)

// JobExecutor executes a specific job type
type JobExecutor interface {
	Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error
}

// Service manages user background jobs
type Service struct {
	store     storage.Store
	logger    *logrus.Logger
	executors map[models.UserJobType]JobExecutor

	// Worker control
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	workerCnt int

	// Job queue
	jobQueue chan string // Job IDs to process
	mu       sync.RWMutex
}

// Config holds configuration for the job service
type Config struct {
	WorkerCount  int           // Number of concurrent workers (default: 3)
	QueueSize    int           // Job queue buffer size (default: 100)
	PollInterval time.Duration // How often to check for pending jobs (default: 5s)
	JobTimeout   time.Duration // Max time for a single job (default: 30m)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		WorkerCount:  3,
		QueueSize:    100,
		PollInterval: 5 * time.Second,
		JobTimeout:   30 * time.Minute,
	}
}

// NewService creates a new job service
func NewService(store storage.Store, logger *logrus.Logger, config Config) *Service {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 3
	}
	if config.QueueSize <= 0 {
		config.QueueSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		store:     store,
		logger:    logger,
		executors: make(map[models.UserJobType]JobExecutor),
		ctx:       ctx,
		cancel:    cancel,
		workerCnt: config.WorkerCount,
		jobQueue:  make(chan string, config.QueueSize),
	}
}

// RegisterExecutor registers a job executor for a specific job type
func (s *Service) RegisterExecutor(jobType models.UserJobType, executor JobExecutor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.executors[jobType] = executor
	s.logger.WithField("job_type", jobType).Info("Registered job executor")
}

// Start starts the job service workers
func (s *Service) Start() {
	s.logger.WithField("workers", s.workerCnt).Info("Starting user job service")

	// Start workers
	for i := 0; i < s.workerCnt; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// Start job poller
	s.wg.Add(1)
	go s.poller()
}

// Stop stops the job service gracefully
func (s *Service) Stop() {
	s.logger.Info("Stopping user job service")
	s.cancel()
	close(s.jobQueue)
	s.wg.Wait()
	s.logger.Info("User job service stopped")
}

// SubmitJob creates and queues a new job
func (s *Service) SubmitJob(ctx context.Context, job *models.UserJob) error {
	// Validate executor exists
	s.mu.RLock()
	_, exists := s.executors[job.JobType]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no executor registered for job type: %s", job.JobType)
	}

	// Create job in database
	if err := s.store.CreateUserJob(ctx, job); err != nil {
		return fmt.Errorf("create job: %w", err)
	}

	// Queue for processing
	select {
	case s.jobQueue <- job.ID:
		s.logger.WithFields(logrus.Fields{
			"job_id":   job.ID,
			"job_type": job.JobType,
			"user_id":  job.UserID,
		}).Info("Job queued for processing")
	default:
		s.logger.WithField("job_id", job.ID).Warn("Job queue full, job will be picked up by poller")
	}

	return nil
}

// CancelJob cancels a job
func (s *Service) CancelJob(ctx context.Context, jobID string) error {
	return s.store.CancelUserJob(ctx, jobID)
}

// GetJob retrieves a job by ID
func (s *Service) GetJob(ctx context.Context, jobID string) (*models.UserJob, error) {
	return s.store.GetUserJob(ctx, jobID)
}

// ListJobs lists jobs with filtering
func (s *Service) ListJobs(ctx context.Context, req *models.UserJobsRequest) ([]models.UserJob, int, error) {
	return s.store.ListUserJobs(ctx, req)
}

// GetActiveJobs returns all active jobs for a user
func (s *Service) GetActiveJobs(ctx context.Context, userID string) ([]models.UserJob, error) {
	return s.store.GetActiveUserJobs(ctx, userID)
}

// worker processes jobs from the queue
func (s *Service) worker(id int) {
	defer s.wg.Done()

	logger := s.logger.WithField("worker_id", id)
	logger.Debug("Worker started")

	for {
		select {
		case <-s.ctx.Done():
			logger.Debug("Worker stopping")
			return

		case jobID, ok := <-s.jobQueue:
			if !ok {
				logger.Debug("Job queue closed")
				return
			}
			s.processJob(jobID)
		}
	}
}

// poller periodically checks for pending jobs
func (s *Service) poller() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-ticker.C:
			s.pollPendingJobs()
		}
	}
}

// pollPendingJobs finds and queues pending jobs
func (s *Service) pollPendingJobs() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	status := models.UserJobStatusPending
	jobs, _, err := s.store.ListUserJobs(ctx, &models.UserJobsRequest{
		Status: &status,
		Limit:  50,
	})
	if err != nil {
		s.logger.WithError(err).Error("Failed to poll pending jobs")
		return
	}

	for _, job := range jobs {
		select {
		case s.jobQueue <- job.ID:
			// Queued
		default:
			// Queue full, will try next poll
			break
		}
	}
}

// processJob processes a single job
func (s *Service) processJob(jobID string) {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	logger := s.logger.WithField("job_id", jobID)

	// Get job details
	job, err := s.store.GetUserJob(ctx, jobID)
	if err != nil {
		logger.WithError(err).Error("Failed to get job")
		return
	}

	// Skip if not pending
	if job.Status != models.UserJobStatusPending {
		logger.WithField("status", job.Status).Debug("Job not pending, skipping")
		return
	}

	// Get executor
	s.mu.RLock()
	executor, exists := s.executors[job.JobType]
	s.mu.RUnlock()

	if !exists {
		logger.WithField("job_type", job.JobType).Error("No executor for job type")
		s.store.UpdateUserJobStatus(ctx, jobID, models.UserJobStatusFailed, "no executor for job type")
		return
	}

	// Mark as running
	if err := s.store.UpdateUserJobStatus(ctx, jobID, models.UserJobStatusRunning, ""); err != nil {
		logger.WithError(err).Error("Failed to mark job as running")
		return
	}

	logger.WithField("job_type", job.JobType).Info("Processing job")

	// Parse config
	var config *models.UserJobConfig
	if job.Config != "" {
		config = &models.UserJobConfig{}
		if err := json.Unmarshal([]byte(job.Config), config); err != nil {
			logger.WithError(err).Warn("Failed to parse job config")
		}
	}

	// Progress callback
	progressCb := func(progress int, msg string) {
		if err := s.store.UpdateUserJobProgress(ctx, jobID, progress, msg); err != nil {
			logger.WithError(err).Warn("Failed to update progress")
		}
	}

	// Execute job
	err = executor.Execute(ctx, job, config, progressCb)

	if err != nil {
		logger.WithError(err).Error("Job execution failed")
		s.store.UpdateUserJobStatus(ctx, jobID, models.UserJobStatusFailed, err.Error())
		return
	}

	// Mark as completed
	if err := s.store.UpdateUserJobStatus(ctx, jobID, models.UserJobStatusCompleted, ""); err != nil {
		logger.WithError(err).Error("Failed to mark job as completed")
		return
	}

	// Update progress to 100%
	s.store.UpdateUserJobProgress(ctx, jobID, 100, "Completed")

	logger.Info("Job completed successfully")
}
