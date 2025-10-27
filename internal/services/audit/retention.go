// Package audit provides audit event retention policy management
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package audit

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

// RetentionPolicy manages automatic cleanup of old audit events
type RetentionPolicy struct {
	db              storage.Database
	logger          *logrus.Logger
	retentionPeriod time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// NewRetentionPolicy creates a new audit retention policy manager
// Default: retain 90 days, cleanup daily
func NewRetentionPolicy(db storage.Database, logger *logrus.Logger) *RetentionPolicy {
	return &RetentionPolicy{
		db:              db,
		logger:          logger,
		retentionPeriod: 90 * 24 * time.Hour,  // 90 days
		cleanupInterval: 24 * time.Hour,        // daily cleanup
		stopChan:        make(chan struct{}),
	}
}

// WithRetentionPeriod sets custom retention period
func (r *RetentionPolicy) WithRetentionPeriod(period time.Duration) *RetentionPolicy {
	r.retentionPeriod = period
	return r
}

// WithCleanupInterval sets custom cleanup interval
func (r *RetentionPolicy) WithCleanupInterval(interval time.Duration) *RetentionPolicy {
	r.cleanupInterval = interval
	return r
}

// Start begins the automatic cleanup process
func (r *RetentionPolicy) Start() {
	r.logger.WithFields(logrus.Fields{
		"retention_period": r.retentionPeriod,
		"cleanup_interval": r.cleanupInterval,
	}).Info("Starting audit event retention policy")

	// Run first cleanup immediately
	go r.runCleanup()

	// Start periodic cleanup
	go r.periodicCleanup()
}

// Stop stops the automatic cleanup process
func (r *RetentionPolicy) Stop() {
	r.logger.Info("Stopping audit event retention policy")
	close(r.stopChan)
}

// periodicCleanup runs cleanup on a schedule
func (r *RetentionPolicy) periodicCleanup() {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.runCleanup()
		case <-r.stopChan:
			r.logger.Info("Audit retention policy stopped")
			return
		}
	}
}

// runCleanup performs the actual cleanup
func (r *RetentionPolicy) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cutoffTime := time.Now().Add(-r.retentionPeriod)

	r.logger.WithField("cutoff_time", cutoffTime).Info("Running audit event cleanup")

	deletedCount, err := r.db.DeleteOldAuditEvents(ctx, cutoffTime)
	if err != nil {
		r.logger.WithError(err).Error("Failed to delete old audit events")
		return
	}

	if deletedCount > 0 {
		r.logger.WithFields(logrus.Fields{
			"deleted_count": deletedCount,
			"cutoff_time":   cutoffTime,
		}).Info("Audit event cleanup completed")
	} else {
		r.logger.Debug("No old audit events to clean up")
	}
}

// RunOnce runs cleanup once immediately (for manual trigger)
func (r *RetentionPolicy) RunOnce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cutoffTime := time.Now().Add(-r.retentionPeriod)

	r.logger.WithField("cutoff_time", cutoffTime).Info("Manual audit event cleanup triggered")

	deletedCount, err := r.db.DeleteOldAuditEvents(ctx, cutoffTime)
	if err != nil {
		r.logger.WithError(err).Error("Manual cleanup failed")
		return err
	}

	r.logger.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"cutoff_time":   cutoffTime,
	}).Info("Manual audit event cleanup completed")

	return nil
}


