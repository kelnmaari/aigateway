// Package queue provides job queue implementations.
package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// PostgresQueue implements Queue using PostgreSQL table
type PostgresQueue struct {
	db     storage.Storage
	logger *logrus.Logger
}

// NewPostgresQueue creates new PostgreSQL-backed queue
func NewPostgresQueue(db storage.Storage, logger *logrus.Logger) *PostgresQueue {
	return &PostgresQueue{
		db:     db,
		logger: logger,
	}
}

// Enqueue adds a job to the queue
func (q *PostgresQueue) Enqueue(ctx context.Context, job Job) error {
	// Serialize payload
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ragJob := &models.RAGJob{
		ID:        job.ID,
		JobType:   job.Type,
		Status:    "pending",
		Payload:   payloadJSON,
		Priority:  job.Priority,
		Attempts:  0,
		CreatedAt: time.Now(),
	}

	if err := q.db.CreateRAGJob(ctx, ragJob); err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	q.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"job_type": job.Type,
		"priority": job.Priority,
	}).Debug("Job enqueued")

	return nil
}

// Dequeue retrieves next job from queue (FIFO with priority)
func (q *PostgresQueue) Dequeue(ctx context.Context) (*Job, error) {
	// Get next pending job with locking (FOR UPDATE SKIP LOCKED)
	ragJob, err := q.db.GetNextPendingRAGJob(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, fmt.Errorf("failed to get next job: %w", err)
	}

	// Deserialize payload
	var payload map[string]interface{}
	if err := json.Unmarshal(ragJob.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	job := &Job{
		ID:        ragJob.ID,
		Type:      ragJob.JobType,
		Payload:   payload,
		Priority:  ragJob.Priority,
		Attempts:  ragJob.Attempts,
		CreatedAt: ragJob.CreatedAt,
	}

	q.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"job_type": job.Type,
	}).Debug("Job dequeued")

	return job, nil
}

// ProcessJob marks job as processing and locks it
func (q *PostgresQueue) ProcessJob(ctx context.Context, jobID string, fn func(context.Context, Job) error) error {
	// Get job
	ragJob, err := q.db.GetRAGJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Update status to processing
	ragJob.Status = "processing"
	ragJob.Attempts++
	now := time.Now()
	ragJob.UpdatedAt = &now

	// Lock until 5 minutes from now
	lockUntil := now.Add(5 * time.Minute)
	ragJob.LockedUntil = &lockUntil

	if err := q.db.UpdateRAGJob(ctx, ragJob); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Deserialize payload
	var payload map[string]interface{}
	if err := json.Unmarshal(ragJob.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	job := Job{
		ID:        ragJob.ID,
		Type:      ragJob.JobType,
		Payload:   payload,
		Priority:  ragJob.Priority,
		Attempts:  ragJob.Attempts,
		CreatedAt: ragJob.CreatedAt,
	}

	// Execute job
	startTime := time.Now()
	err = fn(ctx, job)
	duration := time.Since(startTime)

	// Update job result
	now = time.Now()
	ragJob.UpdatedAt = &now

	if err != nil {
		ragJob.Status = "failed"
		errMsg := err.Error()
		ragJob.Error = &errMsg

		// Retry logic: если attempts < 3, переводим обратно в pending
		if ragJob.Attempts < 3 {
			ragJob.Status = "pending"
			ragJob.LockedUntil = nil // Unlock
		}

		q.logger.WithFields(logrus.Fields{
			"job_id":      jobID,
			"attempts":    ragJob.Attempts,
			"duration_ms": duration.Milliseconds(),
			"error":       err.Error(),
		}).Error("Job processing failed")
	} else {
		ragJob.Status = "completed"

		// Serialize result
		result := map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"completed_at": now,
		}
		resultJSON, _ := json.Marshal(result)
		ragJob.Result = resultJSON

		q.logger.WithFields(logrus.Fields{
			"job_id":      jobID,
			"duration_ms": duration.Milliseconds(),
		}).Info("Job completed successfully")
	}

	if err := q.db.UpdateRAGJob(ctx, ragJob); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	return nil
}

// GetQueueSize returns number of pending jobs
func (q *PostgresQueue) GetQueueSize(ctx context.Context) (int, error) {
	count, err := q.db.CountRAGJobsByStatus(ctx, "pending")
	if err != nil {
		return 0, fmt.Errorf("failed to count pending jobs: %w", err)
	}
	return count, nil
}

// CleanupOldJobs удаляет старые completed/failed jobs
func (q *PostgresQueue) CleanupOldJobs(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	deleted, err := q.db.DeleteOldRAGJobs(ctx, cutoffTime, []string{"completed", "failed"})
	if err != nil {
		return fmt.Errorf("failed to delete old jobs: %w", err)
	}

	q.logger.WithFields(logrus.Fields{
		"deleted":      deleted,
		"cutoff_time":  cutoffTime,
	}).Info("Old jobs cleaned up")

	return nil
}

// UnlockExpiredJobs разблокирует jobs с истекшим LockedUntil
func (q *PostgresQueue) UnlockExpiredJobs(ctx context.Context) error {
	now := time.Now()

	unlocked, err := q.db.UnlockExpiredRAGJobs(ctx, now)
	if err != nil {
		return fmt.Errorf("failed to unlock expired jobs: %w", err)
	}

	if unlocked > 0 {
		q.logger.WithField("unlocked", unlocked).Info("Expired jobs unlocked")
	}

	return nil
}


