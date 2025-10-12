package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// Scheduler manages scheduled report generation and delivery.
type Scheduler struct {
	cron      *cron.Cron
	generator *ReportGenerator
	mailer    *EmailMailer
	logger    *logrus.Logger
	config    SchedulerConfig
	running   bool
}

// SchedulerConfig holds configuration for report scheduler.
type SchedulerConfig struct {
	Enabled   bool             `mapstructure:"enabled"`   // Enable/disable scheduler
	Schedules []ReportSchedule `mapstructure:"schedules"` // List of scheduled reports
}

// NewScheduler creates a new report scheduler instance.
func NewScheduler(
	generator *ReportGenerator,
	mailer *EmailMailer,
	logger *logrus.Logger,
	cfg SchedulerConfig,
) *Scheduler {
	return &Scheduler{
		cron:      cron.New(),
		generator: generator,
		mailer:    mailer,
		logger:    logger,
		config:    cfg,
		running:   false,
	}
}

// Start initializes all scheduled reports and begins cron scheduler.
//
// Parameters:
//   - ctx: Context for graceful shutdown
//
// Returns error if scheduler initialization fails.
func (s *Scheduler) Start(ctx context.Context) error {
	if !s.config.Enabled {
		s.logger.Info("Report scheduler is disabled")
		return nil
	}

	if len(s.config.Schedules) == 0 {
		s.logger.Warn("No report schedules configured")
		return nil
	}

	s.logger.Info("Starting report scheduler")

	// Register all schedules
	for _, schedule := range s.config.Schedules {
		if !schedule.Enabled {
			s.logger.WithField("schedule", schedule.Name).Info("Schedule disabled, skipping")
			continue
		}

		// Validate cron expression
		if _, err := cron.ParseStandard(schedule.Schedule); err != nil {
			return fmt.Errorf("invalid cron expression for schedule %s: %w", schedule.Name, err)
		}

		// Capture schedule in closure
		sched := schedule

		// Add cron job for this schedule
		_, err := s.cron.AddFunc(sched.Schedule, func() {
			s.executeReport(context.Background(), sched)
		})
		if err != nil {
			return fmt.Errorf("failed to add schedule %s: %w", schedule.Name, err)
		}

		s.logger.WithFields(logrus.Fields{
			"id":         schedule.ID,
			"name":       schedule.Name,
			"type":       schedule.Type,
			"schedule":   schedule.Schedule,
			"recipients": len(schedule.Recipients),
		}).Info("Report schedule registered")
	}

	// Start cron scheduler
	s.cron.Start()
	s.running = true

	s.logger.WithField("schedules", len(s.config.Schedules)).Info("Report scheduler started successfully")

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop halts the scheduler and waits for running jobs to complete.
func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.logger.Info("Stopping report scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.logger.Info("Report scheduler stopped")
}

// executeReport generates and sends a scheduled report.
func (s *Scheduler) executeReport(ctx context.Context, schedule ReportSchedule) {
	s.logger.WithFields(logrus.Fields{
		"schedule": schedule.Name,
		"type":     schedule.Type,
	}).Info("Executing scheduled report")

	start := time.Now()

	// Determine time period based on report type
	now := time.Now()
	var periodStart, periodEnd time.Time

	switch schedule.Type {
	case ReportTypeUsage, ReportTypePerformance:
		// Last 24 hours for usage and performance
		periodEnd = now
		periodStart = now.Add(-24 * time.Hour)
	case ReportTypeSystemHealth:
		// Current snapshot for system health
		periodEnd = now
		periodStart = now
	default:
		s.logger.WithField("type", schedule.Type).Error("Unknown report type")
		return
	}

	// Generate report
	report, err := s.generator.Generate(ctx, schedule.Type, periodStart, periodEnd)
	if err != nil {
		s.logger.WithError(err).WithField("schedule", schedule.Name).Error("Failed to generate report")
		return
	}

	// Prepare email subject
	subject := fmt.Sprintf("[Ollama Proxy] %s - %s", 
		formatReportType(schedule.Type),
		time.Now().Format("2006-01-02"),
	)

	// Send via email
	err = s.mailer.SendReport(ctx, EmailReport{
		Recipients: schedule.Recipients,
		Subject:    subject,
		Report:     report,
	})
	if err != nil {
		s.logger.WithError(err).WithField("schedule", schedule.Name).Error("Failed to send report")
		return
	}

	duration := time.Since(start)
	s.logger.WithFields(logrus.Fields{
		"schedule":   schedule.Name,
		"type":       schedule.Type,
		"recipients": len(schedule.Recipients),
		"duration":   duration,
	}).Info("Report sent successfully")
}

// formatReportType converts ReportType to human-readable string.
func formatReportType(rt ReportType) string {
	switch rt {
	case ReportTypeUsage:
		return "Usage Report"
	case ReportTypePerformance:
		return "Performance Report"
	case ReportTypeSystemHealth:
		return "System Health Report"
	case ReportTypeAll:
		return "Complete Report"
	default:
		return string(rt)
	}
}

// ExecuteManual manually triggers a report generation and sending.
//
// Useful for testing or on-demand report generation.
func (s *Scheduler) ExecuteManual(ctx context.Context, scheduleID string) error {
	// Find schedule by ID
	for _, schedule := range s.config.Schedules {
		if schedule.ID == scheduleID {
			s.executeReport(ctx, schedule)
			return nil
		}
	}
	return fmt.Errorf("schedule not found: %s", scheduleID)
}

// GetSchedules returns list of configured schedules.
func (s *Scheduler) GetSchedules() []ReportSchedule {
	return s.config.Schedules
}

// IsRunning returns whether scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	return s.running
}

